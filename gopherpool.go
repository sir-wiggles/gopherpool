package gopherpool

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Work interface {
	DoWork()
}

type (
	payload struct {
		work          Work
		resultChannel chan bool
	}

	Pool struct {
		queueChannel          chan payload
		workChannel           chan Work
		queueShutdownChannel  chan bool
		workerShutdownChannel chan bool
		workerWaitGroup       *sync.WaitGroup
		queuedWork            int32
		activeWorkers         int32
		queueCapacity         int32
		numberOfWorkers       int
		processedJobs         uint64
	}
)

// Starts all the threads needed for the pool
func NewPool(numberOfWorkers, queueCapacity, workQueueCapacity int32) *Pool {

	if numberOfWorkers <= 0 {
		numberOfWorkers = 1
	}

	if queueCapacity <= 0 {
		queueCapacity = 1
	}

	if workQueueCapacity <= 0 {
		workQueueCapacity = numberOfWorkers
	}

	pool := Pool{
		queueChannel:          make(chan payload, queueCapacity),
		workChannel:           make(chan Work, workQueueCapacity),
		queueShutdownChannel:  make(chan bool),
		workerShutdownChannel: make(chan bool, numberOfWorkers),
		workerWaitGroup:       new(sync.WaitGroup),
		queueCapacity:         queueCapacity,
		numberOfWorkers:       int(numberOfWorkers),
	}

	go pool.distributor()

	// +1 for the distributor
	pool.workerWaitGroup.Add(int(numberOfWorkers) + 1)
	for workerID := 0; workerID < int(numberOfWorkers); workerID++ {
		go pool.runWorker(workerID)
	}
	return &pool
}

// Shutdown the pool
func (pool *Pool) Shutdown() bool {
	defer close(pool.queueChannel)
	defer close(pool.workChannel)
	defer close(pool.queueShutdownChannel)
	defer close(pool.workerShutdownChannel)

	pool.queueShutdownChannel <- true

	pool.workerWaitGroup.Wait()
	fmt.Println("Gopher pool closed")

	return true
}

// How you put message on the work queue
func (pool *Pool) Push(w Work) bool {
	queueItem := payload{w, make(chan bool)}
	defer close(queueItem.resultChannel)
	pool.queueChannel <- queueItem
	return <-queueItem.resultChannel
}

// Returns the count of queued works
func (pool *Pool) QueuedWork() int32 {
	return atomic.AddInt32(&pool.queuedWork, 0)
}

// Returns the count of active workers
func (pool *Pool) ActiveWorkers() int32 {
	return atomic.AddInt32(&pool.activeWorkers, 0)
}

// Looping gopher pulling work off the work queue until something comes on the
// shutdown channel
func (pool *Pool) runWorker(id int) {
	defer pool.workerWaitGroup.Done()
	for {
		select {
		case <-pool.workerShutdownChannel:
			fmt.Printf(
				"Gopher %3d drowing, %3d gophers remaining\n",
				id,
				pool.activeWorkers,
			)
			return
		case work := <-pool.workChannel:
			atomic.AddInt32(&pool.queuedWork, -1)
			pool.doWork(work)
		}
	}

}

// Where the actual work gets done
func (pool *Pool) doWork(work Work) {
	defer atomic.AddInt32(&pool.activeWorkers, -1)
	atomic.AddInt32(&pool.activeWorkers, 1)
	work.DoWork()
	atomic.AddUint64(&pool.processedJobs, 1)
}

// Takes messages from the outside world and puts them on the work queue
func (pool *Pool) distributor() {
	defer pool.workerWaitGroup.Done()
	for {
		select {
		case <-pool.queueShutdownChannel:
			fmt.Println("Closing gopher pool")
			for worker := 0; worker < pool.numberOfWorkers; worker++ {
				pool.workerShutdownChannel <- true
			}
			return
		case queueItem := <-pool.queueChannel:
			if atomic.AddInt32(&pool.queuedWork, 0) == pool.queueCapacity {
				queueItem.resultChannel <- false
				continue
			}
			atomic.AddInt32(&pool.queuedWork, 1)
			pool.workChannel <- queueItem.work
			queueItem.resultChannel <- true
		}
	}
}
