package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type ctxKey string

const requestIDKey ctxKey = "requestID"

// middleware that injects a value into the request context
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// in real apps this might come from a header or be generated (uuid, etc.)
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = "generated-123"
		}

		// store it in the context
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)

		// pass the new context down the chain
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handler(w http.ResponseWriter, r *http.Request) {
	// retrieve the value later in the request lifecycle
	reqID, _ := r.Context().Value(requestIDKey).(string)

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "hello! requestID=%s\n", reqID)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	// wrap mux with middleware
	h := withRequestID(mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", h))
}
