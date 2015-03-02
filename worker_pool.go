package work

import (
	"github.com/garyburd/redigo/redis"
	"reflect"
	// "fmt"
)

type WorkerPool struct {
	concurrency uint
	namespace   string // eg, "myapp-work"
	pool        *redis.Pool

	contextType reflect.Type
	jobTypes    map[string]*jobType

	workers []*worker
}

func NewWorkerPool(ctx interface{}, concurrency uint, namespace string, pool *redis.Pool) *WorkerPool {
	// todo: validate ctx
	// todo: validate concurrency
	wp := &WorkerPool{
		concurrency: concurrency,
		namespace:   namespace,
		pool:        pool,
		contextType: reflect.TypeOf(ctx),
		jobTypes:    make(map[string]*jobType),
	}

	for i := uint(0); i < wp.concurrency; i++ {
		w := newWorker(wp.namespace, wp.pool, wp.jobTypes)
		wp.workers = append(wp.workers, w)
	}

	return wp
}

func (wp *WorkerPool) Middleware() *WorkerPool {
	return wp
}

func (wp *WorkerPool) Job(name string, fn interface{}) *WorkerPool {
	return wp.JobWithOptions(name, JobOptions{Priority: 1, MaxFails: 3}, fn)
}

// TODO: depending on how many JobOptions there are it might be good to explode the options
// because it's super awkward for Priority and MaxRetries to be zero-valued
func (wp *WorkerPool) JobWithOptions(name string, jobOpts JobOptions, fn interface{}) *WorkerPool {
	jt := &jobType{
		Name:           name,
		DynamicHandler: reflect.ValueOf(fn),
		JobOptions:     jobOpts,
	}
	if gh, ok := fn.(func(*Job) error); ok {
		jt.IsGeneric = true
		jt.GenericHandler = gh
	}

	wp.jobTypes[name] = jt

	for _, w := range wp.workers {
		w.updateJobTypes(wp.jobTypes)
	}

	return wp
}

func (wp *WorkerPool) Start() {
	// todo: what if already started?
	for _, w := range wp.workers {
		w.start()
	}
}

func (wp *WorkerPool) Stop() {
}

func (wp *WorkerPool) Join() {
	for _, w := range wp.workers {
		w.join()
	}
}