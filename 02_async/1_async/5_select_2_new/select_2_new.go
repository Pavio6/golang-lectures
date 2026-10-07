package main

import (
	"fmt"
)

// Read 2 channels till they are closed

func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	go func() {
		ch2 <- 0
		close(ch2)
		ch1 <- 1
		ch1 <- 2
		ch1 <- 3
		ch1 <- 4
		ch1 <- 5
		ch1 <- 6
		close(ch1)
	}()

	for {
		select {
		case v1, ok := <-ch1:
			if !ok {
				fmt.Print("ch1 is closed; ")
				ch1 = nil // Try to comment it
			}
			fmt.Println("chan1 val", v1)
		case v2, ok := <-ch2:
			if !ok {
				fmt.Print("ch2 is closed; ")
				ch2 = nil
			}
			fmt.Println("chan2 val", v2)
		}

		if ch1 == nil && ch2 == nil {
			fmt.Print("Both channels are closed")
			break
		}
	}
}
