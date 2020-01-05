[![Build Status](https://travis-ci.org/dunv/concurrentList.svg?branch=master)](https://travis-ci.org/dunv/concurrentList)
[![GoDoc](https://godoc.org/github.com/dunv/concurrentList?status.svg)](https://godoc.org/github.com/dunv/concurrentList)
[![codecov](https://codecov.io/gh/dunv/concurrentList/branch/master/graph/badge.svg)](https://codecov.io/gh/dunv/concurrentList)

# concurrentList

A simple implementation of a concurrent list, which supports getting multiple items by filter, blocking reads and blocking reads with timeout.
See GoDoc (badge above) and tests for more examples.

```go
func test() {
    list := NewConcurrentList()

    go func(list *ConcurrentList) {
        list.Append("test")
    }(list)

    go func(list *ConcurrentList) {
        item, err := list.GetNext()
        if err == concurrentList.ErrEmptyList {
            ...
        }
        fmt.Println("got", item.(string))
    }(list)

    ...
}
```