package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

var client = http.Client{Timeout: time.Millisecond}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handler)
	fmt.Println("starting server at :8080")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handler(w http.ResponseWriter, r *http.Request) {
	err := getRemoteResource()
	if err != nil {
		fmt.Printf("error happened: %+v\n", err)
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	fmt.Fprint(w, "all is OK")
}

func getRemoteResource() error {
	url := "http://127.0.0.1:9999/pages?id=123"
	resp, err := client.Get(url)

	if err != nil {
		return fmt.Errorf("getRemoteResource: %w at %s", err, url)
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("getRemoteResource: unexpected status %s", resp.Status)
	}

	return nil
}
