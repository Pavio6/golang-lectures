package main

import (
	"fmt"
	"log"
	"net/http"
)

// Начиная с Go 1.22 ServeMux поддерживает методы и шаблоны путей.
// Более специфичные шаблоны имеют приоритет.

// curl -v -X GET -H "Content-Type: application/json" http://localhost:8080
// curl -v -X PUT -H "Content-Type: application/json" http://localhost:8080/unknown_path
func DefaultPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "\n\tYou see default page\n\n")
}

// curl -v -X GET -H "Content-Type: application/json" http://localhost:8080/users
func UserList(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou see user list\n\n")
}

// curl -v -X GET -H "Content-Type: application/json" http://127.0.0.1:8080/users
func UserListForHost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou see user list for host %q\n\n", r.Host)
}

// curl -v -X GET -H "Content-Type: application/json" http://localhost:8080/users/
func ExactUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou see exact \"users\" list\n\n")
}

// curl -v -X GET -H "Content-Type: application/json" http://127.0.0.1:8080/users/bob
func GetUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou try to get user %q\n\n", r.PathValue("id"))
}

// curl -v -X POST -H "Content-Type: application/json" http://127.0.0.1:8080/users/bob
func CreateUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou try to create new user %q\n\n", r.PathValue("login"))
}

// curl -v -X PUT -H "Content-Type: application/json" http://localhost:8080/users/bob
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou try to update user %q\n\n", r.PathValue("login"))
}

// curl -v -X GET -H "Content-Type: application/json" http://localhost:8080/users/bob/check
func LongUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou see long %q list\n\n", r.PathValue("path"))
}

// curl -v -X GET -H "Content-Type: application/json" http://localhost:8080/users/bob/id
func ID(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "\n\tYou try to get id for %q\n\n", r.PathValue("path"))
}

func main() {
	r := http.NewServeMux()

	r.HandleFunc("/", DefaultPage)

	r.HandleFunc("GET /users", UserList)
	r.HandleFunc("GET 127.0.0.1/users", UserListForHost) // Наличие хоста повышает приоритет шаблона
	// r.HandleFunc("/users/{path...}", LongUsers)

	// "/users/{id}" matches exactly one path segment after "/users/".
	r.HandleFunc("GET /users/{id}", GetUser)
	r.HandleFunc("PUT /users/{login}", UpdateUser)
	r.HandleFunc("POST /users/{login}", CreateUser)

	r.HandleFunc("GET /users/{$}", ExactUsers) // Матчит только запросы на "/users/"

	r.HandleFunc("GET /users/{path...}", LongUsers) // Матчит запросы на "/users/..."

	r.HandleFunc("GET /users/{path}/id", ID)
	//r.HandleFunc("GET /{path}/id", ID) // Ambiguous with the other patterns.

	fmt.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// func main() {
// 	r := http.NewServeMux()

// 	r.HandleFunc("GET /users", UserList) // 405 для других методов (кроме HEAD)

// 	fmt.Println("starting server at :8080")
// 	log.Fatal(http.ListenAndServe(":8080", r))
// }
