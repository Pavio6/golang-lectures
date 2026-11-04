package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func withAdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("adminAuthMiddleware", r.URL.Path)

		_, err := r.Cookie("session_id")
		// Учебный пример! это не проверка авторизации!
		if err != nil {
			fmt.Println("no auth at", r.URL.Path)
			http.Redirect(w, r, "/", http.StatusFound)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("accessLogMiddleware", r.URL.Path)
		start := time.Now()

		next.ServeHTTP(w, r)

		fmt.Printf("[%s] %s, %s %s\n\n\n", r.Method, r.RemoteAddr, r.URL.Path, time.Since(start))
	})
}

func withPanicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("panicMiddleware", r.URL.Path)

		defer func() {
			if err := recover(); err != nil {
				fmt.Println("recovered", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func main() {
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /", adminIndex)
	adminMux.HandleFunc("GET /panic", adminPanicPage)

	// Set admin middleware:
	adminHandler := withAdminAuth(http.StripPrefix("/admin", adminMux))

	mux := http.NewServeMux()
	mux.Handle("GET /admin/", adminHandler)
	mux.HandleFunc("POST /login", loginPage)
	mux.HandleFunc("POST /logout", logoutPage)
	mux.HandleFunc("GET /", mainPage)

	// Set common middleware:
	siteHandler := withPanicRecovery(withLogging(mux))

	fmt.Println("starting server at http://127.0.0.1:8080")
	log.Fatal(http.ListenAndServe(":8080", siteHandler))
}

func adminIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `<a href="/">site index</a>`)
	fmt.Fprintln(w, "Admin main page")
}

func adminPanicPage(w http.ResponseWriter, r *http.Request) {
	panic("this must me recovered")
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	expiration := time.Now().Add(10 * time.Hour)
	cookie := http.Cookie{
		Name:     "session_id",
		Value:    "Sweetie",
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutPage(w http.ResponseWriter, r *http.Request) {
	session, err := r.Cookie("session_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	session.Expires = time.Now().AddDate(0, 0, -1)
	session.MaxAge = -1

	http.SetCookie(w, session)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	session, err := r.Cookie("session_id")
	// Учебный пример! это не проверка авторизации!
	loggedOut := errors.Is(err, http.ErrNoCookie)

	if loggedOut {
		fmt.Fprintln(w, `<html><form action="/login" method="post"><button>login</button></form>`)
		fmt.Fprintln(w, "You need to login")
	} else {
		fmt.Fprintln(w, `<html><form action="/logout" method="post"><button>logout</button></form>`)
		fmt.Fprintln(w, "Welcome, "+session.Value)
	}
}
