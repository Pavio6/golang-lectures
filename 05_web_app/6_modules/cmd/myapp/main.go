package main

import (
	"fmt"
	"myapp/internal/user"
)

func main() {
	u := user.NewUser(42, "rvasily")
	fmt.Println("my user:", u)
}
