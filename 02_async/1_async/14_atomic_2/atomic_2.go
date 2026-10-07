package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

var totalOperations int32

func inc() {
	atomic.AddInt32(&totalOperations, 1) // атомарно
	// totalOperations++
}

func main() {
	for i := 0; i < 1000; i++ {
		go inc()
	}

	time.Sleep(100 * time.Millisecond)
	fmt.Println("total operation = ", totalOperations)
}
