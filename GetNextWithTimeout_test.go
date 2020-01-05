package concurrentList

import (
	"strings"
	"testing"
	"time"
)

func TestGetNextWithTimeout(t *testing.T) {
	list := NewConcurrentList()
	list.Append("testItem")

	// Successful getNext
	item, err := list.GetNextWithTimeout(time.Millisecond)
	if err != nil {
		t.Error(err)
		return
	}
	if _, ok := item.(string); !ok {
		t.Errorf("did not get an item back")
		return
	}
	if strings.Compare(item.(string), "testItem") != 0 {
		t.Errorf("did not get the correct item back (expected: %s, actual: %s)", "testItem", item.(string))
		return
	}

	// Start asynchronous call to get another (this should complete after the synchronous one following fails)
	done := make(chan bool)
	go func(done chan<- bool) {
		item, err = list.GetNextWithTimeout(time.Second)
		if err != nil {
			t.Error("expected item, but did get one")
			return
		}
		if _, ok := item.(string); !ok {
			t.Errorf("did not get an item back")
			return
		}
		if strings.Compare(item.(string), "asyncItem") != 0 {
			t.Errorf("did not get the correct item back (expected: %s, actual: %s)", "asyncItem", item.(string))
			return
		}
		done <- true
	}(done)

	// Unsuccessful getNext
	_, err = list.GetNextWithTimeout(time.Millisecond)
	if err == nil {
		t.Error("got item back, but did not expect it")
		return
	}
	if err != ErrEmptyList {
		t.Error("receiver wrong error")
		return
	}

	// Publish another item
	list.Append("asyncItem")

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Error("did not complete in time")
	}
}
