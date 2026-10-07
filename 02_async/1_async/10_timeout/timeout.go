package main

import (
	"fmt"
	"time"
)

func longSQLQuery() chan bool {
	ch := make(chan bool, 1)
	go func() {
		time.Sleep(2 * time.Second) // 4 * time.Second
		ch <- true
	}()
	return ch
}

func main() {
	// При 1 выполнится таймаут, при 3 - выполнится операция
	timer := time.NewTimer(3 * time.Second)
	select {
	case <-timer.C:
		fmt.Println("timer.C timeout happened")
	case result := <-longSQLQuery():
		// Освобождаем ресурс (но с Go 1.23 сборщик мусора сам умеет освобождать неиспользуемые таймеры)
		if !timer.Stop() {
			<-timer.C
		}
		fmt.Println("operation result:", result)
	}
}
