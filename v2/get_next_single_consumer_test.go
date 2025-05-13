package v2

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGetNextSingleConsumer(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	max := 1000000

	wg := &sync.WaitGroup{}
	list := NewConcurrentList[int]()

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(200 * time.Millisecond)
		for i := range max {
			// add some delays, so the consumer needs to block
			if i%(max/100) == 0 {
				time.Sleep(10 * time.Millisecond)
			}
			list.Push(i)
		}
	}()

	for {
		item, err := list.GetNext(ctx)
		if err != nil {
			// this should only return an error, when the context is done
			if ctx.Err() == nil {
				t.Error("unexpected error:", err)
				t.FailNow()
			}
			break
		}

		// fmt.Println("yielding item:", item)
		if item == max-1 {
			cancel()
		}

	}

	wg.Wait()
}
