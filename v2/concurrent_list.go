package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ErrEmptyList is returned if one tries to get items from an empty list
var ErrEmptyList = errors.New("list is empty")

// ConcurrentList is a thread-safe datastructure which holds a list of items
// if desired these items can be automatically sorted or the list persisted on the HDD upon each change
// Any goroutine which calls GetNext() will block until an item is available (they are guaranteed to
// to continue in the same order GetNext() is called) or a passed context expires
type ConcurrentList[T any] struct {
	// Hold data
	data []T
	// Condition for waiting reads which also contains the mutex
	// protecting the data
	cond *sync.Cond
	// Options
	opts concurrentListOptions[T]
}

// Constructor for creating a ConcurrentList (is required for initializing subscriber channels)
func NewConcurrentList[T any](opts ...ConcurrentListOption[T]) *ConcurrentList[T] {
	mergedOpts := concurrentListOptions[T]{
		lessFunc: nil,
	}
	for _, opt := range opts {
		opt.apply(&mergedOpts)
	}

	l := &ConcurrentList[T]{
		data: []T{},
		cond: sync.NewCond(&sync.Mutex{}),
		opts: mergedOpts,
	}

	// Reconstruct persisted list
	if mergedOpts.persistChanges {
		err := l.persistenceLoad()
		if err != nil && mergedOpts.persistErrorHandler != nil {
			(*mergedOpts.persistErrorHandler)(err)
		}

		if l.opts.lessFunc != nil {
			sort.Slice(l.data, func(i, j int) bool {
				return (*l.opts.lessFunc)(l.data[i], l.data[j])
			})
		}
	}

	if mergedOpts.ttlEnabled {
		go func() {
			for {
				l.DeleteWithFilter(func(item T) bool {
					ttlAttribute := (*mergedOpts.ttlFunc)(item)
					return time.Since(ttlAttribute) > *mergedOpts.ttlDuration
				})
				time.Sleep(*mergedOpts.ttlCheckInverval)
			}
		}()
	}

	return l

}

// Append to the end of the list
func (l *ConcurrentList[T]) Push(items ...T) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	l.data = append(l.data, items...)
	if l.opts.lessFunc != nil {
		sort.Slice(l.data, func(i, j int) bool {
			return (*l.opts.lessFunc)(l.data[i], l.data[j])
		})
	}

	// Write a single file per item in a directory
	if l.opts.persistChanges {
		for _, item := range items {
			err := l.persistenceCreateFile(item)
			if err != nil && l.opts.persistErrorHandler != nil {
				(*l.opts.persistErrorHandler)(err)
			}
		}
	}

	l.cond.Signal()
}

// Shift attempts to get the "oldest" item from the list
// Will return ErrEmptyList if the list is empty
func (l *ConcurrentList[T]) Shift() (T, error) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	return l.shift()
}

// Allows to "peek" into the list without removing the item
func (l *ConcurrentList[T]) Peek() (T, error) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	if len(l.data) < 1 {
		var t T
		return t, ErrEmptyList
	}

	firstElement := l.data[0]
	return firstElement, nil
}

// Gets the "oldest" item in the list. Blocks until an item is available or the
// passed in context expires
func (l *ConcurrentList[T]) GetNext(ctx context.Context) (T, error) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	// wake up all waiting routines (broadcast) when the context expires
	// we don't know if this one is woken up, if we only wake
	// up a "random" one (which is the behaviour of signal)
	stop := context.AfterFunc(ctx, func() {
		l.cond.Broadcast()
	})
	// do not call afterFunc if this function completes before
	// the context expires
	defer stop()

	// Wait until we have something or the context expired
	for {
		if ctx.Err() != nil {
			var noop T
			return noop, ctx.Err()
		}
		if len(l.data) > 0 {
			break
		}

		// Hint: Wait does the following
		//  - release the lock
		//  - suspends the current goroutine until the condition is signaled
		//  - reacquires the lock
		// that is why we can get away with locking the mutex when the function
		// begins and unlocking it with a simple defer
		l.cond.Wait()
	}

	return l.shift()
}

// GetWithFilter will get all items of the list which match a predicate WITHOUT changing the list
// ("peek" into the list's items)
func (l *ConcurrentList[T]) GetWithFilter(predicate func(item T) bool) []T {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	filteredItems := []T{}
	for _, item := range l.data {
		if predicate(item) {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems
}

// DeleteWithFilter will get and remove all items of the list which match a predicate
func (l *ConcurrentList[T]) DeleteWithFilter(predicate func(item T) bool) []T {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()

	nonFilteredItems := []T{}
	filteredItems := []T{}
	for _, item := range l.data {
		if !predicate(item) {
			nonFilteredItems = append(nonFilteredItems, item)
		} else {
			filteredItems = append(filteredItems, item)
		}
	}

	// Delete all filtered files in the persistance directory
	if l.opts.persistChanges {
		for _, item := range filteredItems {
			err := l.persistenceDeleteFile(item)
			if err != nil && l.opts.persistErrorHandler != nil {
				(*l.opts.persistErrorHandler)(err)
			}
		}
	}

	// Keep non-filtered items
	l.data = nonFilteredItems

	// Return filtered ones
	return filteredItems
}

// Length returns the length of the list
func (l *ConcurrentList[T]) Length() int {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	return len(l.data)
}

// internal helper function for getting the first item. the caller needs to make sure the collection is locked
func (l *ConcurrentList[T]) shift() (T, error) {
	if len(l.data) < 1 {
		var noop T
		return noop, ErrEmptyList
	}

	firstElement := l.data[0]
	l.data = l.data[1:len(l.data)]

	// Delete the single file in our persistanceDirectory
	if l.opts.persistChanges {
		err := l.persistenceDeleteFile(firstElement)
		if err != nil && l.opts.persistErrorHandler != nil {
			(*l.opts.persistErrorHandler)(err)
		}
	}

	return firstElement, nil
}

func (l *ConcurrentList[T]) persistenceLoad() error {
	files, err := os.ReadDir(l.opts.persistRootPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		marshaled, err := os.ReadFile(filepath.Join(l.opts.persistRootPath, file.Name()))
		if err != nil {
			return err
		}
		var elem T
		err = json.Unmarshal(marshaled, &elem)
		if err != nil {
			return err
		}
		// Make sure we are not storing a pointer to our item
		l.data = append(l.data, elem)
	}

	return nil
}

func (l *ConcurrentList[T]) persistenceCreateFile(item T) error {
	marshaled, err := json.Marshal(item)
	if err != nil {
		return err
	}
	itemPath := filepath.Join(l.opts.persistRootPath, (*l.opts.persistFileNameFunc)(item))
	file, err := os.Create(itemPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("err closing file: %s\n", err)
		}
	}()

	_, err = file.Write(marshaled)
	if err != nil {
		return err
	}
	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (l *ConcurrentList[T]) persistenceDeleteFile(item interface{}) error {
	itemPath := filepath.Join(l.opts.persistRootPath, (*l.opts.persistFileNameFunc)(item.(T)))
	return os.Remove(itemPath)
}
