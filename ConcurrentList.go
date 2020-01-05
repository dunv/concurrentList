package concurrentList

import (
	"errors"
	"sync"
	"time"
)

// ErrEmptyList is returned if one tries to get items from an empty list
var ErrEmptyList = errors.New("list is empty")

// ConcurrentList data-structure which holds all data
type ConcurrentList struct {
	data                 []interface{}
	mutex                sync.Mutex
	nextAddedSubscribers []*chan bool
}

// NewConcurrentList is the constructor for creating a ConcurrentList (is required for initializing subscriber channels)
func NewConcurrentList() *ConcurrentList {
	return &ConcurrentList{
		data:                 []interface{}{},
		mutex:                sync.Mutex{},
		nextAddedSubscribers: make([]*chan bool, 0),
	}
}

// Append to list. This method should never block
func (l *ConcurrentList) Append(item interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.data = append(l.data, item)

	if len(l.nextAddedSubscribers) > 0 {
		subscriber := l.nextAddedSubscribers[0]
		l.nextAddedSubscribers = l.nextAddedSubscribers[1:len(l.nextAddedSubscribers)]
		go func() {
			*subscriber <- true
		}()
	}
}

// Shift will attempt to get the "oldest" item from the list
// Will return EMPTY_LIST if the list is empty
func (l *ConcurrentList) Shift() (interface{}, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if len(l.data) < 1 {
		return nil, ErrEmptyList
	}

	firstElement := l.data[0]
	l.data = l.data[1:len(l.data)]
	return firstElement, nil
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

// GetNext will get the "oldest" item from the list. Will block until an item is available
// The returned error is of type EMTPY_LIST and should NEVER occur. I kept this in
// to facilitate troubleshooting
func (l *ConcurrentList) GetNext() (interface{}, error) {
	l.mutex.Lock()
	if len(l.data) > 0 {
		defer l.mutex.Unlock()
		return l.shiftWithoutLock()
	}

	getNextChannel := make(chan bool)
	l.nextAddedSubscribers = append(l.nextAddedSubscribers, &getNextChannel)
	l.mutex.Unlock()

	<-getNextChannel
	return l.GetNext()
}

// GetNextWithTimeout will get the "oldest" item from the list. Will block until an item is available OR the specified
// duration passed. The returned error is of type EMTPY_LIST and should NEVER occur. I kept this in
// to facilitate troubleshooting
func (l *ConcurrentList) GetNextWithTimeout(timeout time.Duration) (interface{}, error) {
	l.mutex.Lock()
	var getNextChannel *chan bool
	if len(l.data) > 0 {
		l.mutex.Unlock()
		return l.Shift()
	}

	tmp := make(chan bool)
	getNextChannel = &tmp
	l.nextAddedSubscribers = append(l.nextAddedSubscribers, getNextChannel)
	l.mutex.Unlock()

	select {
	case <-*getNextChannel:
		return l.GetNext()
	case <-time.After(timeout):
		l.mutex.Lock()
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
	l.data = nonFilteredItems
	return filteredItems
}

// Length returns the length of the list
func (l *ConcurrentList) Length() int {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return len(l.data)
}
