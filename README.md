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

	pool := gopherpool.NewPool(<number_of_workers>, <queue_capacity>, <worker_queue_capacity>)

	for {

		for x := 0; x < 200; x++ {
			w := &Work{}
			pool.Push(w)
		}
		break
	}
	pool.Shutdown()
	fmt.Println(pool.Stats())

}

type Work struct {}

func (p *Work) DoWork() {
	return
}
```
