package v2

import (
	"testing"
)

func TestDeleteWithFilter(t *testing.T) {
	length := 50
	list := NewConcurrentList[int]()

	for i := 0; i < length; i++ {
		list.Push(i)
	}

	if list.Length() != length {
		t.Errorf("appending did not work as expected (expected length: %d, actual length: %d)", length, list.Length())
		return
	}

	items := list.DeleteWithFilter(func(item int) bool {
		return item < length/2
	})

	if len(items) != length/2 {
		t.Error("did not receive the correct amount of items")
	}

	if list.Length() != length/2 {
		t.Errorf("getWithFilter modified the list (expected length: %d, actual length: %d)", length, list.Length())
		return
	}

	for _, item := range items {
		if item > length/2 {
			t.Errorf("list still contains an item greater than %d", length/2)
			return
		}
	}
}
