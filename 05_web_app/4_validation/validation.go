package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"
)

type sendMessageRequest struct {
	ID        int    `json:"id,omitempty"`
	Priority  string `json:"priority"`
	Recipient string `json:"to"`
	Subject   string `json:"subject"`
}

func (msg sendMessageRequest) validate() error {
	switch msg.Priority {
	case "low", "normal", "high":
	default:
		return errors.New("priority must be low, normal, or high")
	}

	address, err := mail.ParseAddress(msg.Recipient)
	if err != nil || address.Address != msg.Recipient {
		return errors.New("to must be a valid email address")
	}

	if n := utf8.RuneCountInString(strings.TrimSpace(msg.Subject)); n == 0 || n > 10 {
		return errors.New("subject must contain between 1 and 10 characters")
	}

	return nil
}

//	curl -X POST http://localhost:8080/ -H 'Content-Type: application/json' \
//	  -d '{"id":12,"priority":"low","to":"v.romanov@example.com","subject":"Hello!"}'
func handler(w http.ResponseWriter, r *http.Request) {
	// Limit the body size before decoding untrusted input.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var msg sendMessageRequest
	if err := decoder.Decode(&msg); err != nil {
		writeDecodeError(w, err)
		return
	}
	// Require exactly one JSON value in the request body.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = errors.New("request must contain a single JSON value")
		}
		writeDecodeError(w, err)
		return
	}

	if err := msg.validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		http.Error(w, "request body is too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", handler)

	log.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
