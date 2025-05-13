package v2

import (
	"context"
	"testing"
)

func TestGetNext(t *testing.T) {
	list := NewConcurrentList[[]int]()
	insertItems := []map[int]bool{}
	verifyItems := []map[int]bool{}

	totalProducer := 100
	totalItemsPerProducer := 1000
	totalConsumer := 100
	bufferSize := totalProducer * totalItemsPerProducer

	// Create fixture
	readChannel := make(chan []int, bufferSize)
	for i := range totalProducer {
		insertItems = append(insertItems, map[int]bool{})
		verifyItems = append(verifyItems, map[int]bool{})
		for j := range totalItemsPerProducer {
			insertItems[i][j] = false
			verifyItems[i][j] = false
		}
	}

	// Create consumers
	for range totalConsumer {
		go consumer(list, readChannel, t.Context())
	}

	// Create producers
	for i := range totalProducer {
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

	// Wait until validation is done
	<-allValid
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

func consumer(list *ConcurrentList[[]int], readChannel chan []int, ctx context.Context) {
	for {
		item, err := list.GetNext(ctx)
		if err != nil {
			return
		}
		readChannel <- item
	}
}

func producer(insertItems map[int]bool, producerIndex int, list *ConcurrentList[[]int]) {
	for index := range insertItems {
		tmp1 := producerIndex
		tmp2 := index
		list.Push([]int{tmp1, tmp2})
	}
}
