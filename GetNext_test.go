package concurrentList

import (
	"context"
	"testing"
	"time"
)

// This will get stuck in a deadlock, if it fails
func TestGetNext(t *testing.T) {
	list := NewConcurrentList()
	insertItems := []map[int]bool{}
	verifyItems := []map[int]bool{}

	totalProducer := 100
	totalItemsPerProducer := 1000
	totalConsumer := 100
	bufferSize := totalProducer * totalItemsPerProducer

	ctx, cancel := context.WithCancel(context.Background())

	// Create fixture
	readChannel := make(chan []int, bufferSize)
	for i := 0; i < totalProducer; i++ {
		insertItems = append(insertItems, map[int]bool{})
		verifyItems = append(verifyItems, map[int]bool{})
		for j := 0; j < totalItemsPerProducer; j++ {
			insertItems[i][j] = false
			verifyItems[i][j] = false
		}
	}

	// Create consumers
	for i := 0; i < totalConsumer; i++ {
		go consumer(list, &readChannel, ctx, t)
	}

	// Create producers
	for i := 0; i < totalProducer; i++ {
		go producer(insertItems[i], i, list)
	}

	// Validate
	allValid := make(chan bool)
	go func() {
		for item := range readChannel {
			verifyItems[item[0]][item[1]] = true
			if verify(verifyItems) {
				allValid <- true
			}
		}
	}()

	// Wait until validation is done, then cancel all contexts
	<-allValid
	cancel()

	// Wait until everything is cleaned up
	for {
		wait, signal := list.Debug()
		if wait > 0 && signal > 0 {
			time.Sleep(1 * time.Millisecond)
			continue
		}
		break
	}
}

func verify(verifyItems []map[int]bool) bool {
	for producerKey := range verifyItems {
		for _, itemValue := range verifyItems[producerKey] {
			if !itemValue {
				return false
			}
		}
	}
	return true
}

func consumer(list *ConcurrentList, readChannel *chan []int, ctx context.Context, t *testing.T) {
	for {
		item, err := list.GetNext(ctx)
		if err != nil {
			if err == ErrEmptyList {
				return
			}
		}

		parsed, ok := item.([]int)
		if ok {
			*readChannel <- parsed
		} else {
			t.Errorf("received unexpected item %v", item)
		}
	}
}

func producer(insertItems map[int]bool, producerIndex int, list *ConcurrentList) {
	for index := range insertItems {
		tmp1 := producerIndex
		tmp2 := index
		list.Push([]int{tmp1, tmp2})
	}
}
