package concurrentList

import (
	"strings"
	"testing"
	"time"
)

func TestSuccessfulGetNextWithTimeout(t *testing.T) {
	list := NewConcurrentList()

	go func() {
		<-time.After(1 * time.Millisecond)
		list.Append("testItem")
	}()

	// Successful getNext
	item, err := list.GetNextWithTimeout(time.Second)
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
}

func TestFailedGetNextWithTimeout(t *testing.T) {
	list := NewConcurrentList()

	go func() {
		<-time.After(time.Second)
		list.Append("asyncItem")
	}()

	_, err := list.GetNextWithTimeout(time.Millisecond)
	if err == nil {
		t.Error("got item back, but did not expect it")
		return
	}
	if err != ErrEmptyList {
		t.Error("receiver wrong error")
		return
	}

}
