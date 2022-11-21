package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestWithSorting(t *testing.T) {
	type test struct {
		item     string
		priority int
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	list := NewConcurrentList(WithSorting(func(i, j test) bool {
		return i.priority > j.priority
	}))

	list.Push(test{
		item:     "prio500",
		priority: 500,
	})
	list.Push(test{
		item:     "prio200",
		priority: 200,
	})
	list.Push(test{
		item:     "prio100",
		priority: 100,
	})
	list.Push(test{
		item:     "prio300",
		priority: 300,
	})

	item1, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item1)
	assert.Equal(t, "prio500", item1.item)

	item2, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item2)
	assert.Equal(t, "prio300", item2.item)

	item3, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item3)
	assert.Equal(t, "prio200", item3.item)

	item4, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item4)
	assert.Equal(t, "prio100", item4.item)
}

func TestWithSortingMultiple(t *testing.T) {
	type test struct {
		item     string
		priority int
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	list := NewConcurrentList(WithSorting(func(i, j test) bool {
		return i.priority > j.priority
	}))

	list.Push(test{
		item:     "prio500",
		priority: 500,
	}, test{
		item:     "prio200",
		priority: 200,
	}, test{
		item:     "prio100",
		priority: 100,
	}, test{
		item:     "prio300",
		priority: 300,
	})

	item1, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item1)
	assert.Equal(t, "prio500", item1.item)

	item2, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item2)
	assert.Equal(t, "prio300", item2.item)

	item3, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item3)
	assert.Equal(t, "prio200", item3.item)

	item4, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item4)
	assert.Equal(t, "prio100", item4.item)
}
