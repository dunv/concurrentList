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
)

// ErrEmptyList is returned if one tries to get items from an empty list
var ErrEmptyList = errors.New("list is empty")

// ConcurrentList data-structure which holds all data
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

// NewConcurrentList is the constructor for creating a ConcurrentList (is required for initializing subscriber channels)
func NewConcurrentList(opts ...ConcurrentListOption) *ConcurrentList {
	mergedOpts := concurrentListOptions{
		sortByFunc: nil,
	}
	for _, opt := range opts {
		opt.apply(&mergedOpts)
	}

	lock := new(sync.Mutex)

	runningSignalRoutines := int64(0)
	runningWaitRoutines := int64(0)

	data := []interface{}{}

	// Reconstruct persisted list
	if mergedOpts.persistChanges {
		files, err := ioutil.ReadDir(mergedOpts.persistRootPath)
		if err != nil {
			(*mergedOpts.persistErrorHandler)(err)
		} else {
			for _, file := range files {
				tmp := reflect.New(reflect.TypeOf(mergedOpts.persistItemType)).Interface()
				marshaled, err := ioutil.ReadFile(filepath.Join(mergedOpts.persistRootPath, file.Name()))
				if err != nil {
					(*mergedOpts.persistErrorHandler)(err)
				} else {
					err = json.Unmarshal(marshaled, &tmp)
					if err != nil {
						(*mergedOpts.persistErrorHandler)(err)

					} else {
						data = append(data, tmp)
					}
				}
			}
		}
	}

	return &ConcurrentList{
		data:                  data,
		lock:                  lock,
		notEmpty:              sync.NewCond(lock),
		opts:                  mergedOpts,
		runningSignalRoutines: &runningSignalRoutines,
		runningWaitRoutines:   &runningWaitRoutines,
	}
}

// Append to list. This method should never block
func (l *ConcurrentList) Push(item interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.data = append(l.data, item)
	if l.opts.sortByFunc != nil {
		sort.Slice(l.data, func(i, j int) bool {
			return (*l.opts.sortByFunc)(l.data[i], l.data[j])
		})
	}

	// Write a single file per item in a directory
	if l.opts.persistChanges {
		marshaled, err := json.Marshal(item)
		if err != nil {
			(*l.opts.persistErrorHandler)(err)
		} else {
			itemPath := filepath.Join(l.opts.persistRootPath, (*l.opts.persistFileNameFunc)(item))
			err = ioutil.WriteFile(itemPath, marshaled, 0644)
			if err != nil {
				(*l.opts.persistErrorHandler)(err)
			}
		}
	}
	// fmt.Println("count", len(l.data))

	l.notEmpty.Signal()
}

// Shift will attempt to get the "oldest" item from the list
// Will return ErrEmptyList if the list is empty
func (l *ConcurrentList) Shift() (interface{}, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.shift()
}

// internal helper function
func (l *ConcurrentList) shift() (interface{}, error) {
	if len(l.data) < 1 {
		return nil, ErrEmptyList
	}

	firstElement := l.data[0]
	l.data = l.data[1:len(l.data)]

	// Delete the single file in our persistanceDirectory
	if l.opts.persistChanges {
		itemPath := filepath.Join(l.opts.persistRootPath, (*l.opts.persistFileNameFunc)(firstElement))
		err := os.Remove(itemPath)
		if err != nil {
			(*l.opts.persistErrorHandler)(err)
		}
	}

	// fmt.Println("count", len(l.data))

	return firstElement, nil
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

// GetWithFilter will get all items of the list which match a predicate ("peak" into the list's items)
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

func (l *ConcurrentList) Debug() (int64, int64) {
	return atomic.LoadInt64(l.runningWaitRoutines), atomic.LoadInt64(l.runningSignalRoutines)
}
