package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

var client = http.Client{Timeout: time.Millisecond}

type ResourceError struct {
	URL string
	Err error
}

func (re *ResourceError) Error() string {
	return fmt.Sprintf(
		"Resource error: URL: %s, err: %v",
		re.URL,
		re.Err,
	)
}

// Unwrap is here, so others can access the underlying error with errors.Is or errors.As
func (re *ResourceError) Unwrap() error {
	return re.Err
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handler)

	fmt.Println("starting server at :8080")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handler(w http.ResponseWriter, r *http.Request) {
	err := getRemoteResource()
	if err != nil {
		var resourceErr *ResourceError
		if errors.As(err, &resourceErr) {
			fmt.Printf("resource %s err: %s\n", resourceErr.URL, resourceErr.Err)
			http.Error(w, "remote resource error", http.StatusInternalServerError)
		} else {
			fmt.Printf("internal error: %+v\n", err)
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
		return &ResourceError{URL: url, Err: err}
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &ResourceError{URL: url, Err: fmt.Errorf("unexpected status %s", resp.Status)}
	}

	return nil
}
