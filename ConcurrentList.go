package concurrentList

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ErrEmptyList is returned if one tries to get items from an empty list
var ErrEmptyList = errors.New("list is empty")

// ConcurrentList is a thread-safe datastructure which holds a list of items (interfaces{})
// if desired these items can be automatically sorted or the list persisted on the HDD upon each change
// Any goroutine which calls GetNext() will block until an item is available (they are guaranteed to
// to continue in the same order GetNext() is called) or a passed context expires
type ConcurrentList struct {
	// Hold data
	data []interface{}

	// Protect list
	lock *sync.Mutex

	// Condition for waiting reads
	notEmpty *sync.Cond

	// Options
	opts concurrentListOptions

	// debug
	runningSignalRoutines *int64
	runningWaitRoutines   *int64
}

// Constructor for creating a ConcurrentList (is required for initializing subscriber channels)
func NewConcurrentList(opts ...ConcurrentListOption) *ConcurrentList {
	mergedOpts := concurrentListOptions{
		lessFunc: nil,
	}
	for _, opt := range opts {
		opt.apply(&mergedOpts)
	}

	lock := new(sync.Mutex)

	runningSignalRoutines := int64(0)
	runningWaitRoutines := int64(0)

	list := &ConcurrentList{
		data:                  []interface{}{},
		lock:                  lock,
		notEmpty:              sync.NewCond(lock),
		opts:                  mergedOpts,
		runningSignalRoutines: &runningSignalRoutines,
		runningWaitRoutines:   &runningWaitRoutines,
	}

	// Reconstruct persisted list
	if mergedOpts.persistChanges {
		err := list.persistenceLoad()
		if err != nil && mergedOpts.persistErrorHandler != nil {
			(*mergedOpts.persistErrorHandler)(err)
		}
	}

	if mergedOpts.ttlEnabled {
		go func() {
			for {
				list.DeleteWithFilter(func(item interface{}) bool {
					ttlAttribute := (*mergedOpts.ttlFunc)(item)
					return time.Since(ttlAttribute) > *mergedOpts.ttlDuration
				})
				time.Sleep(*mergedOpts.ttlCheckInverval)
			}
		}()
	}

	return list

}

// Append to the end of the list
func (l *ConcurrentList) Push(item interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.data = append(l.data, item)
	if l.opts.lessFunc != nil {
		sort.Slice(l.data, func(i, j int) bool {
			return (*l.opts.lessFunc)(l.data[i], l.data[j])
		})
	}

	// Write a single file per item in a directory
	if l.opts.persistChanges {
		err := l.persistenceCreateFile(item)
		if err != nil && l.opts.persistErrorHandler != nil {
			(*l.opts.persistErrorHandler)(err)
		}
	}

	// fmt.Println("count", len(l.data))

	l.notEmpty.Signal()
}

// Shift attempts to get the "oldest" item from the list
// Will return ErrEmptyList if the list is empty
func (l *ConcurrentList) Shift() (interface{}, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.shift()
}

func (l *ConcurrentList) Peek() (interface{}, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if len(l.data) < 1 {
		return nil, ErrEmptyList
	}

	firstElement := l.data[0]
	return firstElement, nil
}

// Gets the "oldest" item in the list. Blocks until an item is available or the
// passed in context expires
func (l *ConcurrentList) GetNext(ctx context.Context) (interface{}, error) {
	l.lock.Lock()
	atomic.AddInt64(l.runningWaitRoutines, 1)
	// fmt.Printf("waitCount %d\n", *l.runningWaitRoutines)

	useCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start one routine which wakes the other one up after the context expired
	go func() {
		atomic.AddInt64(l.runningSignalRoutines, 1)
		// fmt.Printf("signalerCount %d\n", *l.runningSignalRoutines)
		<-useCtx.Done()
		l.notEmpty.Signal()
		atomic.AddInt64(l.runningSignalRoutines, -1)
		// fmt.Printf("signalerCount %d\n", *l.runningSignalRoutines)
	}()

	// Wait until we have something or the context expired
	for len(l.data) == 0 || ctx.Err() != nil {
		if err := ctx.Err(); err != nil {
			atomic.AddInt64(l.runningWaitRoutines, -1)
			// fmt.Printf("waitCount %d\n", *l.runningWaitRoutines)
			l.lock.Unlock()
			return nil, ErrEmptyList
		}
		l.notEmpty.Wait()
	}

	data, err := l.shift()
	atomic.AddInt64(l.runningWaitRoutines, -1)
	// fmt.Printf("waitCount %d\n", *l.runningWaitRoutines)
	l.lock.Unlock()

	return data, err
}

// GetWithFilter will get all items of the list which match a predicate WITHOUT changing the list
// ("peek" into the list's items)
func (l *ConcurrentList) GetWithFilter(predicate func(item interface{}) bool) []interface{} {
	l.lock.Lock()
	defer l.lock.Unlock()

	filteredItems := []interface{}{}
	for _, item := range l.data {
		if predicate(item) {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems
}

// DeleteWithFilter will get and remove all items of the list which match a predicate
func (l *ConcurrentList) DeleteWithFilter(predicate func(item interface{}) bool) []interface{} {
	l.lock.Lock()
	defer l.lock.Unlock()

	nonFilteredItems := []interface{}{}
	filteredItems := []interface{}{}
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
func (l *ConcurrentList) Length() int {
	l.lock.Lock()
	defer l.lock.Unlock()
	return len(l.data)
}

// for testing. The metrics tell the caller how many goroutines are
// running in order to service the concurrentList
func (l *ConcurrentList) debug() (int64, int64) {
	return atomic.LoadInt64(l.runningWaitRoutines), atomic.LoadInt64(l.runningSignalRoutines)
}

// internal helper function for getting the first item. the caller needs to make sure the collection is locked
func (l *ConcurrentList) shift() (interface{}, error) {
	if len(l.data) < 1 {
		return nil, ErrEmptyList
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

	// fmt.Println("count", len(l.data))

	return firstElement, nil
}

func (l *ConcurrentList) persistenceLoad() error {
	files, err := ioutil.ReadDir(l.opts.persistRootPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		tmp := reflect.New(reflect.TypeOf(l.opts.persistItemType)).Interface()
		marshaled, err := ioutil.ReadFile(filepath.Join(l.opts.persistRootPath, file.Name()))
		if err != nil {
			return err
		}
		err = json.Unmarshal(marshaled, &tmp)
		if err != nil {
			return err
		}
		// Make sure we are not storing a pointer to our item
		l.data = append(l.data, reflect.ValueOf(tmp).Elem().Interface())
	}

	return nil
}

func (l *ConcurrentList) persistenceCreateFile(item interface{}) error {
	marshaled, err := json.Marshal(item)
	if err != nil {
		return err
	}
	itemPath := filepath.Join(l.opts.persistRootPath, (*l.opts.persistFileNameFunc)(item))
	file, err := os.Create(itemPath)
	if err != nil {
		return err
	}
	defer file.Close()

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

func (l *ConcurrentList) persistenceDeleteFile(item interface{}) error {
	itemPath := filepath.Join(l.opts.persistRootPath, (*l.opts.persistFileNameFunc)(item))
	return os.Remove(itemPath)
}
