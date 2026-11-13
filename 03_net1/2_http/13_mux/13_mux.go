package main

import (
	"fmt"
	"net/http"
	"time"
)

const adminPrefix = "admin"

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "URL:", r.URL.String())
}

func adminPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ADMIN URL:", r.URL.String())
}

func adminPage2(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ADMIN USER URL:", r.URL.String())
}

func main() {
	admin := http.NewServeMux()
	admin.HandleFunc("/user", adminPage2)
	admin.HandleFunc("/", adminPage)

	adminHandler := http.StripPrefix(
		"/"+adminPrefix,
		admin,
	)

	mux := http.NewServeMux()
	mux.Handle("/admin/", adminHandler)
	mux.HandleFunc("/", handler)

	server := http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Println("starting server at :8080")
	server.ListenAndServe()
}
