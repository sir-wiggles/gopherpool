package cesspool

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
		work   Work
		result chan bool
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
	}
)

func NewPool(numberOfWorkers int, queueCapacity int32) *Pool {
	var workers int
	if numberOfWorkers == 0 {
		workers = 1
	} else {
		workers = numberOfWorkers
	}

	pool := Pool{
		queueChannel:          make(chan payload),
		workChannel:           make(chan Work, workers),
		queueShutdownChannel:  make(chan bool),
		workerShutdownChannel: make(chan bool, workers),
		workerWaitGroup:       new(sync.WaitGroup),
		queueCapacity:         queueCapacity,
		numberOfWorkers:       workers,
	}

	go pool.distributor()

	fmt.Println(workers)
	pool.workerWaitGroup.Add(workers + 1)
	for workerID := 0; workerID < workers; workerID++ {
		go pool.runWorker(workerID)
	}
	return &pool
}

func (pool *Pool) Shutdown() bool {
	defer close(pool.queueChannel)
	defer close(pool.workChannel)
	defer close(pool.queueShutdownChannel)
	defer close(pool.workerShutdownChannel)

	pool.queueShutdownChannel <- true

	pool.workerWaitGroup.Wait()
	fmt.Println("Pool closed")

	return true
}

func (pool *Pool) Push(w Work) bool {
	queueItem := payload{w, make(chan bool, 1)}
	defer close(queueItem.result)
	pool.queueChannel <- queueItem
	return <-queueItem.result
}

func (pool *Pool) QueuedWork() int32 {
	return atomic.AddInt32(&pool.queuedWork, 0)
}

func (pool *Pool) ActiveWorkers() int32 {
	return atomic.AddInt32(&pool.activeWorkers, 0)
}

func (pool *Pool) runWorker(id int) {
	defer pool.workerWaitGroup.Done()
	for {
		select {
		case <-pool.workerShutdownChannel:
			fmt.Printf("Worker %d going down\n", id)
			return
		case work := <-pool.workChannel:
			atomic.AddInt32(&pool.queuedWork, -1)
			pool.doWork(work)
		}
	}

}

func (pool *Pool) doWork(work Work) {
	defer atomic.AddInt32(&pool.activeWorkers, -1)
	atomic.AddInt32(&pool.activeWorkers, 1)
	work.DoWork()
}

func (pool *Pool) distributor() {
	defer pool.workerWaitGroup.Done()
	for {
		select {
		case <-pool.queueShutdownChannel:
			fmt.Println("Closing pool")
			for worker := 0; worker < pool.numberOfWorkers; worker++ {
				pool.workerShutdownChannel <- true
			}
			return
		case queueItem := <-pool.queueChannel:
			if atomic.AddInt32(&pool.queuedWork, 0) == pool.queueCapacity {
				queueItem.result <- false
				continue
			}
			atomic.AddInt32(&pool.queuedWork, 1)
			pool.workChannel <- queueItem.work
			queueItem.result <- true
		}
	}
}
