package concurrentList

import (
	"fmt"
	"testing"
	"time"
)

// This will get stuck in a deadlock, if it fails
func TestShift(t *testing.T) {
	list := NewConcurrentList()
	insertItems := []map[int]bool{}
	verifyItems := []map[int]bool{}

	totalProducer := 10
	totalItemsPerProducer := 10000
	totalConsumer := 1000
	bufferSize := totalProducer * totalItemsPerProducer

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

	start := time.Now()

	// Create consumers
	for i := 0; i < totalConsumer; i++ {
		go consumer(list, &readChannel, t)
	}

	// Create producers
	for i := 0; i < totalProducer; i++ {
		go producer(insertItems[i], i, list)
	}

	// Validate
	for item := range readChannel {
		verifyItems[item[0]][item[1]] = true
		// fmt.Println("consumed", item[0], item[1])
		complete := true
		for producerKey := range verifyItems {
			for _, itemValue := range verifyItems[producerKey] {
				if !itemValue {
					complete = false
				}
			}
		}

		if complete {
			fmt.Printf("\n\nTook %s \n", time.Since(start))
			return
		}
	}

}

func consumer(list *ConcurrentList, readChannel *chan []int, t *testing.T) {
	for {
		item, err := list.GetNext()
		if err != nil {
			t.Error("error", err)
			continue
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
		list.Append([]int{tmp1, tmp2})
	}
}
