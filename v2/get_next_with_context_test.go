package v2

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetNextWithContextNoRecv(t *testing.T) {
	l := NewConcurrentList[int]()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	_, err := l.GetNext(ctx)
	require.Equal(t, context.DeadlineExceeded, err)
}

func TestGetNextWithContextRecv(t *testing.T) {
	// first GetNext then Push
	l := NewConcurrentList[int]()

	ret := make(chan int)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), 1000*time.Millisecond)
		defer cancel()
		recv, err := l.GetNext(ctx)
		if err != nil {
			ret <- 0
			return
		}
		ret <- recv
	}()

	time.Sleep(10 * time.Millisecond)
	l.Push(1)

	select {
	case recv := <-ret:
		require.Equal(t, 1, recv)
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout")
	}
}
