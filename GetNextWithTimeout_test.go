package concurrentList

import (
	"strings"
	"testing"
	"time"
)

func TestGetNextWithTimeout(t *testing.T) {
	list := NewConcurrentList()
	list.Append("testItem")
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

	_, err = list.GetNextWithTimeout(time.Millisecond)
	if err == nil {
		t.Error("got item back, but did not expect it")
		return
	}

	if err != EMPTY_LIST {
		t.Error("received wrong error")
		return
	}
}
