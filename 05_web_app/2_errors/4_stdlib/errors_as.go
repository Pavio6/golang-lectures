package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

// This example uses the standard errors package; a third-party errors package
// is not needed to inspect wrapped errors.
var client = http.Client{Timeout: 10000 * time.Millisecond}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handler)

	fmt.Println("starting server at :8080")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handler(w http.ResponseWriter, r *http.Request) {
	err := getRemoteResource()
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			log.Printf("request to %s failed: %v", urlErr.URL, urlErr.Err)
		} else {
			// For example, the upstream returned a non-2xx HTTP status.
			log.Printf("upstream error: %v", err)
		}
		http.Error(w, "remote resource error", http.StatusBadGateway)
		return
	}

	fmt.Fprint(w, "all is OK")
}

func getRemoteResource() error {
	url := "127.0.0.1:9999"
	//url := "http://example.com/asdfasdfasdf"
	resp, err := client.Get(url)

	if err != nil {
		return fmt.Errorf("resource error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("resource error: unexpected status %s", resp.Status)
	}

	return nil
}
