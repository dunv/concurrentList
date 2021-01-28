package concurrentList

type ConcurrentListOption interface {
	apply(*concurrentListOptions)
}

type concurrentListOptions struct {
	sortByFunc          *func(i, j interface{}) bool
	persistChanges      bool
	persistRootPath     string
	persistItemType     interface{}
	persistFileNameFunc *func(i interface{}) string
	persistErrorHandler *func(error)
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

func WithPersistance(rootPath string, itemType interface{}, fileNameFunc func(i interface{}) string, errorHandler func(error)) ConcurrentListOption {
	return newFuncConcurrentListOption(func(o *concurrentListOptions) {
		o.persistChanges = true
		o.persistRootPath = rootPath
		o.persistItemType = itemType
		o.persistFileNameFunc = &fileNameFunc
		o.persistErrorHandler = &errorHandler
	})
}
