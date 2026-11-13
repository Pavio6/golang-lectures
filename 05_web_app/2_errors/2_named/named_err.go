package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	client = http.Client{Timeout: time.Millisecond}

	ErrResource = errors.New("resource error")
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handler)
	mux.HandleFunc("GET /is", handlerIs)
	fmt.Println("starting server at :8080")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handler(w http.ResponseWriter, r *http.Request) {
	err := getRemoteResource()
	if err != nil {
		fmt.Printf("error happened: %+v\n", err)

		switch err { // switch cannot unwrap errors
		case ErrResource:
			http.Error(w, "remote resource error", http.StatusInternalServerError)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	fmt.Fprint(w, "all is OK")
}

func getRemoteResource() error {
	url := "http://127.0.0.1:9999/pages?id=123"
	resp, err := client.Get(url)

	if err != nil {
		return ErrResource
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: unexpected status %s", ErrResource, resp.Status)
	}

	return nil
}

func handlerIs(w http.ResponseWriter, r *http.Request) {
	err := getRemoteResourceAndSomeWork()
	if errors.Is(err, ErrResource) {
		http.Error(w, "remote resource error", http.StatusInternalServerError)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "all is OK")
}

func getRemoteResourceAndSomeWork() error {
	err := getRemoteResource()
	if err != nil {
		return fmt.Errorf("getRemoteResource: %w", err) // wrap error
	}

	// do work with it

	return nil
}
