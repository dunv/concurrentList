package concurrentList

import (
	"testing"
	"time"
)

func TestShift(t *testing.T) {
	length := 50
	list := NewConcurrentList()

	// Asynchronously append to list
	go func(list *ConcurrentList, length int) {
		for i := 0; i < length; i++ {
			list.Push(i)
		}
	}(list, length)

	// Aynchronously consume from list until we are happy with all returned data or a timeout occurs
	done := make(chan bool)
	go func(list *ConcurrentList, length int, done chan<- bool) {
		lastShifted := -1
		for {
			tmp, err := list.Shift()

			if err == ErrEmptyList && lastShifted < length-1 {
				time.Sleep(time.Millisecond)
				continue
			}

			if err == ErrEmptyList && lastShifted == length-1 {
				done <- true
				return
			}

			if lastShifted >= tmp.(int) {
				t.Error("did not preserve order")
				done <- true
				return
			}

			lastShifted = tmp.(int)
		}
	}(list, length, done)

	// Implement a timeout so this test fails if it takes too long
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("did not complete in time")
		return
	}
}
