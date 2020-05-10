package concurrentList

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

	list := NewConcurrentList(WithSorting(func(i, j interface{}) bool {
		return i.(test).priority > j.(test).priority
	}))

	list.Append(test{
		item:     "prio500",
		priority: 500,
	})
	list.Append(test{
		item:     "prio200",
		priority: 200,
	})
	list.AddToTop(test{
		item:     "prio100",
		priority: 100,
	})
	list.AddToTop(test{
		item:     "prio300",
		priority: 300,
	})

	item1, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item1)
	assert.Equal(t, "prio500", item1.(test).item)

	item2, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item2)
	assert.Equal(t, "prio300", item2.(test).item)

	item3, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item3)
	assert.Equal(t, "prio200", item3.(test).item)

	item4, err := list.GetNext(ctx)
	assert.NoError(t, err)
	assert.IsType(t, test{}, item4)
	assert.Equal(t, "prio100", item4.(test).item)
}
