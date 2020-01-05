package concurrentList

import (
	"testing"
)

func TestGetWithFilter(t *testing.T) {
	length := 50
	list := NewConcurrentList()

	for i := 0; i < length; i++ {
		list.Append(i)
	}

	if list.Length() != length {
		t.Errorf("appending did not work as expected (expected length: %d, actual length: %d)", length, list.Length())
		return
	}

	items := list.GetWithFilter(func(item interface{}) bool {
		return item.(int) < length/2
	})

	if len(items) != length/2 {
		t.Error("did not receive the correct amount of items")
	}

	if list.Length() != length {
		t.Errorf("getWithFilter modified the list (expected length: %d, actual length: %d)", length, list.Length())
		return
	}
}
