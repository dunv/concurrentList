package concurrentList

type ConcurrentListOption interface {
	apply(*concurrentListOptions)
}

type concurrentListOptions struct {
	sortByFunc *func(i, j interface{}) bool
}

type funcConcurrentListOption struct {
	f func(*concurrentListOptions)
}

func (fdo *funcConcurrentListOption) apply(do *concurrentListOptions) {
	fdo.f(do)
}

func newFuncConcurrentListOption(f func(*concurrentListOptions)) *funcConcurrentListOption {
	return &funcConcurrentListOption{f: f}
}

func WithSorting(sortByFunc func(i, j interface{}) bool) ConcurrentListOption {
	return newFuncConcurrentListOption(func(o *concurrentListOptions) {
		o.sortByFunc = &sortByFunc
	})
}
