gopherpool
==========

Go thread pool 

```Go
package main

import (
	"fmt"
	"gopherpool"
)

func main() {

	pool := gopherpool.NewPool(
		<number_of_workers>, // max number of gophers to allow running 
		<queue_capacity>,  // the channel work is pushed to
		<worker_queue_capacity>, // the channel workers read from 
	)
	// <queue_capacity> + <worker_queue_capacity> == total amount of work
	//	that can be queued before Push returns a false

	for x := 0; x < 200; x++ {
		w := &Work{}
		for {
			// returns true is push successful or false if not.
			ok := pool.Push(w)
			if !ok {
				fmt.Println("Queue full, please wait and try again")
				time.Sleep(time.Millisecond * 200)
			} else {
				break
			}
		}
	}
	// brings down all the workers and cleans up after itself
	pool.Shutdown()
	// Can be called anytime really, just gives current stats 
	fmt.Println(pool.Stats())

}

type Work struct {}

// must satisfy the DoWork interface for the pool to run it.
func (p *Work) DoWork() {
	return
}
```
