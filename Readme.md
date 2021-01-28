[![Build Status](https://travis-ci.org/dunv/concurrentList.svg?branch=master)](https://travis-ci.org/dunv/concurrentList)
[![GoDoc](https://godoc.org/github.com/dunv/concurrentList?status.svg)](https://godoc.org/github.com/dunv/concurrentList)
[![codecov](https://codecov.io/gh/dunv/concurrentList/branch/master/graph/badge.svg)](https://codecov.io/gh/dunv/concurrentList)

# concurrentList

A simple implementation of a concurrent list, which supports

- thread-safe `Push` and `Shift` operation
- automatic sorting option (a.k.a priorityQueue)
- blocking reads (`list.GetNext(ctx context.Context)`)
- peeking into the contents of the list without modifying it
- thread-safe removal of multiple items
- optional persistence of the list (i.e. across reboots) by means of writing a file per item in the list in a predefined folder

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
