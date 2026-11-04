package main

import "fmt"

type printer func(string)

func main() {

	var basicPrinter printer = func(phrase string) {
		fmt.Println(phrase)
	}

	//basicPrinter("hi")

	bordersDecorator := func(next printer) printer {
		return func(s string) {
			fmt.Println("--------")
			// if auth?
			next(s)
			fmt.Println("--------")
		}
	}

	borderedPrinter := bordersDecorator(basicPrinter)
	borderedPrinter("hello")

	extraBordersDecorator := func(next printer) printer {
		return func(s string) {
			fmt.Println("========")
			next(s)
			fmt.Println("========")
		}
	}

	extraPrinter := bordersDecorator(extraBordersDecorator(basicPrinter))
	extraPrinter("hi, sweetie")

}
