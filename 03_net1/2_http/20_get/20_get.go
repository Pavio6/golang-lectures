package main

import (
	"fmt"
	"net/http"
)

// curl "localhost:8080/?param=100500" -d "key=my_key"

func handler(w http.ResponseWriter, r *http.Request) {
	myParam := r.URL.Query().Get("param")
	if myParam != "" {
		fmt.Fprintln(w, "`myParam` is", myParam)
	}

	key := r.FormValue("key")
	if key != "" {
		fmt.Fprintln(w, "`key` is", key)
	}

	fmt.Fprintln(w, "\nparam and key are checked")
}

func main() {
	http.HandleFunc("/", handler)

	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}
