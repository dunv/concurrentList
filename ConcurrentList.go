package concurrentList

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrEmptyList is returned if one tries to get items from an empty list
var ErrEmptyList = errors.New("err: list is empty")

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

	return &ConcurrentList{
		data:     []interface{}{},
		lock:     lock,
		notEmpty: sync.NewCond(lock),
		opts:     mergedOpts,
	}
}

// Append to list. This method should never block
func (l *ConcurrentList) Append(item interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.data = append(l.data, item)
	if l.opts.sortByFunc != nil {
		sort.Slice(l.data, func(i, j int) bool {
			return (*l.opts.sortByFunc)(l.data[i], l.data[j])
		})
	}

	l.notEmpty.Signal()
}

func (l *ConcurrentList) AddToTop(item interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.data = append([]interface{}{item}, l.data...)
	if l.opts.sortByFunc != nil {
		sort.Slice(l.data, func(i, j int) bool {
			return (*l.opts.sortByFunc)(l.data[i], l.data[j])
		})
	}

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

// GetNext will get the "oldest" item from the list.
// Will block until an item is available or the context expires
func (l *ConcurrentList) GetNext(ctx context.Context) (interface{}, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	useCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start one routine which wakes the other one up after the context expired
	go func() {
		<-useCtx.Done()
		l.notEmpty.Signal()
	}()

	// Wait until we have something or the context expired
	for len(l.data) == 0 && ctx.Err() == nil {
		l.notEmpty.Wait()
	}

	// If only context expired: return empty list
	if err := ctx.Err(); err != nil {
		return nil, ErrEmptyList
	}

	// If we have something -> return it
	return l.shift()
}

// GetNextWithTimeout will get the "oldest" item from the list. Will block until an item is available OR the specified duration passed.
func (l *ConcurrentList) GetNextWithTimeout(timeout time.Duration) (interface{}, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	return l.GetNext(ctx)
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
