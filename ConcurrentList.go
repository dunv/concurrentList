package concurrentList

import (
	"errors"
	"sync"
	"time"
)

var EMPTY_LIST = errors.New("list is empty")

type ConcurrentList struct {
	data                 []interface{}
	mutex                sync.Mutex
	nextAddedSubscribers []*chan bool
}

func NewConcurrentList() *ConcurrentList {
	return &ConcurrentList{
		data:                 []interface{}{},
		mutex:                sync.Mutex{},
		nextAddedSubscribers: make([]*chan bool, 0),
	}
}

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

func (l *ConcurrentList) Shift() (interface{}, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if len(l.data) < 1 {
		return nil, EMPTY_LIST
	}
	firstElement := l.data[0]
	l.data = l.data[1:len(l.data)]
	return firstElement, nil
}

func (l *ConcurrentList) shiftWithoutLock() (interface{}, error) {
	if len(l.data) < 1 {
		return nil, EMPTY_LIST
	}
	firstElement := l.data[0]
	l.data = l.data[1:len(l.data)]
	return firstElement, nil
}

func (l *ConcurrentList) GetNext() (interface{}, error) {
	l.mutex.Lock()
	if len(l.data) > 0 {
		defer l.mutex.Unlock()
		return l.shiftWithoutLock()
	} else {
		getNextChannel := make(chan bool)
		l.nextAddedSubscribers = append(l.nextAddedSubscribers, &getNextChannel)
		l.mutex.Unlock()

		<-getNextChannel
		return l.GetNext()
	}
}

func (l *ConcurrentList) GetNextWithTimeout(timeout time.Duration) (interface{}, error) {
	l.mutex.Lock()
	var getNextChannel *chan bool
	if len(l.data) > 0 {
		l.mutex.Unlock()
		return l.Shift()
	} else {
		tmp := make(chan bool)
		getNextChannel = &tmp
		l.nextAddedSubscribers = append(l.nextAddedSubscribers, getNextChannel)
		l.mutex.Unlock()

		select {
		case <-*getNextChannel:
			return l.GetNext()
		case <-time.After(timeout):
			l.mutex.Lock()
			newSubscriberList := make([]*chan bool, 0)
			for index := range l.nextAddedSubscribers {
				if l.nextAddedSubscribers[index] != getNextChannel {
					newSubscriberList = append(newSubscriberList, l.nextAddedSubscribers[index])
				}
			}
			l.nextAddedSubscribers = newSubscriberList
			l.mutex.Unlock()
			return nil, EMPTY_LIST
		}
	}
}

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

func (l *ConcurrentList) Length() int {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return len(l.data)
}
