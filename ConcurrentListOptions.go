package concurrentList

import "time"

type ConcurrentListOption interface {
	apply(*concurrentListOptions)
}

type concurrentListOptions struct {
	lessFunc            *func(i, j interface{}) bool
	persistChanges      bool
	persistRootPath     string
	persistItemType     interface{}
	persistFileNameFunc *func(i interface{}) string
	persistErrorHandler *func(error)
	ttlEnabled          bool
	ttlDuration         *time.Duration
	ttlCheckInverval    *time.Duration
	ttlFunc             *func(i interface{}) time.Time
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

// WithSorting will automatically sort the contents of the list everytime
// an item is pushed according to the passed function
// WithSorting can also be used to create a priorityQueue
func WithSorting(lessFunc func(i, j interface{}) bool) ConcurrentListOption {
	return newFuncConcurrentListOption(func(o *concurrentListOptions) {
		o.lessFunc = &lessFunc
	})
}

// WithPersistence adds persistence in terms of "one file per item in the list" on the harddrive
// Whenever anything is added or removed a file with the json-marshaled contents is put into or removed from a directory.
// The caller needs to make sure that the directory of rootPath exists and is writable by the process
// fileNameFunc determines the fileName of every item-file
// itemType is required so the types can be reconstructed from the contents of the rootFolder
// an optional errorHandler can be passed if the caller wants to process perstisting errors
func WithPersistence(rootPath string, itemType interface{}, fileNameFunc func(i interface{}) string, errorHandler ...func(error)) ConcurrentListOption {
	return newFuncConcurrentListOption(func(o *concurrentListOptions) {
		o.persistChanges = true
		o.persistRootPath = rootPath
		o.persistItemType = itemType
		o.persistFileNameFunc = &fileNameFunc

		if len(errorHandler) == 1 {
			o.persistErrorHandler = &errorHandler[0]
		}
	})
}

// WithTTL adds a time-to-live to every item in the list
// ATTENTION: Currently the user is required to add an attribute to every item which contains the timestamp of when it is added
// Required parameters are
// - ttl: 						how long will an item linger in the list until it is deleted automatically
// - ttlCheckInterval: 			in which interval are the ttl's of the items checked
// - ttlFunc: 					this func is called for every item in order to extract the timestamp of when it was added
func WithTTL(ttl time.Duration, ttlCheckInterval time.Duration, ttlFunc func(item interface{}) time.Time) ConcurrentListOption {
	return newFuncConcurrentListOption(func(o *concurrentListOptions) {
		o.ttlEnabled = true
		o.ttlDuration = &ttl
		o.ttlFunc = &ttlFunc
		o.ttlCheckInverval = &ttlCheckInterval
	})
}
