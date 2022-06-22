package v2

import (
	"testing"
	"time"
)

func TestShift(t *testing.T) {
	length := 50
	list := NewConcurrentList[int]()

	// Asynchronously append to list
	go func(list *ConcurrentList[int], length int) {
		for i := 0; i < length; i++ {
			list.Push(i)
		}
	}(list, length)

	// Aynchronously consume from list until we are happy with all returned data or a timeout occurs
	done := make(chan bool)
	go func(list *ConcurrentList[int], length int, done chan<- bool) {
		lastShifted := -1
		for {
			item, err := list.Shift()

			if err == ErrEmptyList && lastShifted < length-1 {
				time.Sleep(time.Millisecond)
				continue
			}

			if err == ErrEmptyList && lastShifted == length-1 {
				done <- true
				return
			}

			if lastShifted >= item {
				t.Error("did not preserve order")
				done <- true
				return
			}

			lastShifted = item
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
