package concurrentList

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrEmptyList is returned if one tries to get items from an empty list
var ErrEmptyList = errors.New("list is empty")

// ConcurrentList data-structure which holds all data
type ConcurrentList struct {
	// Hold data
	data []interface{}

	// Protect list
	mutex sync.Mutex

	// Keep track of subscribers in a list. This way we can preserve order
	nextAddedSubscribers []*chan bool

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

	return &ConcurrentList{
		data:                 []interface{}{},
		mutex:                sync.Mutex{},
		nextAddedSubscribers: make([]*chan bool, 0),
		opts:                 mergedOpts,
	}
}

// Append to list. This method should never block
func (l *ConcurrentList) Append(item interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.data = append(l.data, item)

	if l.opts.sortByFunc != nil {
		sort.Slice(l.data, func(i, j int) bool {
			return (*l.opts.sortByFunc)(l.data[i], l.data[j])
		})
	}

	if len(l.nextAddedSubscribers) > 0 {
		subscriber := l.nextAddedSubscribers[0]
		l.nextAddedSubscribers = l.nextAddedSubscribers[1:len(l.nextAddedSubscribers)]
		go func() {
			// TODO: fix this, goroutine leaks
			*subscriber <- true
		}()
	}
}

func (l *ConcurrentList) AddToTop(item interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.data = append([]interface{}{item}, l.data...)

	if len(l.nextAddedSubscribers) > 0 {
		subscriber := l.nextAddedSubscribers[0]
		l.nextAddedSubscribers = l.nextAddedSubscribers[1:len(l.nextAddedSubscribers)]
		go func() {
			// TODO: fix this, goroutine leaks
			*subscriber <- true
		}()
	}
}

// Shift will attempt to get the "oldest" item from the list
// Will return ErrEmptyList if the list is empty
func (l *ConcurrentList) Shift() (interface{}, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.shiftWithoutLock()
}

// internal helper function
func (l *ConcurrentList) shiftWithoutLock() (interface{}, error) {
	if len(l.data) < 1 {
		return nil, ErrEmptyList
	}

	firstElement := l.data[0]
	l.data = l.data[1:len(l.data)]
	return firstElement, nil
}

func (l *ConcurrentList) Peek() (interface{}, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if len(l.data) < 1 {
		return nil, ErrEmptyList
	}

	firstElement := l.data[0]
	return firstElement, nil
}

// GetNext will get the "oldest" item from the list. Will block until an item is available
// The returned error is of type EMTPY_LIST and should NEVER occur. I kept this in
// to facilitate troubleshooting
func (l *ConcurrentList) GetNext() (interface{}, error) {
	l.mutex.Lock()

	// If we have an item in the list -> return it immediately
	if len(l.data) > 0 {
		defer l.mutex.Unlock()
		return l.shiftWithoutLock()
	}

	// If the list is empty -> create a return-channel on which we will publish when another item is available
	getNextChannel := make(chan bool)
	l.nextAddedSubscribers = append(l.nextAddedSubscribers, &getNextChannel)
	l.mutex.Unlock()

	// Wait until another item has been received
	<-getNextChannel

	// Return it
	return l.GetNext()
}

// GetNextWithTimeout will get the "oldest" item from the list. Will block until an item is available OR the specified
// duration passed. The returned error is of type ErrEmptyList and should NEVER occur. I kept this in
// to facilitate troubleshooting
func (l *ConcurrentList) GetNextWithTimeout(timeout time.Duration) (interface{}, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	return l.GetNextWithContext(ctx)
}

func (l *ConcurrentList) GetNextWithContext(ctx context.Context) (interface{}, error) {
	l.mutex.Lock()

	// If we have an item in the list -> return it immediately
	if len(l.data) > 0 {
		l.mutex.Unlock()
		return l.Shift()
	}

	// If the list is empty -> create a return-channel on which we will publish when another item is available
	tmp := make(chan bool)
	getNextChannel := &tmp
	l.nextAddedSubscribers = append(l.nextAddedSubscribers, getNextChannel)
	l.mutex.Unlock()

	select {
	case <-*getNextChannel:
		// We either receive an item in time
		return l.GetNext()
	case <-ctx.Done():
		// ... or not -> remove ourselves from subscriber list an return ErrEmptyList
		l.mutex.Lock()
		// Remove from subscriber-list, as timeout occured (essentially recreating the list)
		var newSubscriberList []*chan bool
		for index := range l.nextAddedSubscribers {
			if l.nextAddedSubscribers[index] != getNextChannel {
				newSubscriberList = append(newSubscriberList, l.nextAddedSubscribers[index])
			}
		}
		l.nextAddedSubscribers = newSubscriberList
		l.mutex.Unlock()
		return nil, ErrEmptyList
	}
}

// GetWithFilter will get all items of the list which match a predicate ("peak" into the list's items)
func (l *ConcurrentList) GetWithFilter(predicate func(item interface{}) bool) []interface{} {
	l.mutex.Lock()
	defer l.mutex.Unlock()

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
	l.mutex.Lock()
	defer l.mutex.Unlock()

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
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return len(l.data)
}
