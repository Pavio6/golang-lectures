package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	datasetPath  = "dataset.xml"
	requestDelay time.Duration
)

type xmlUser struct {
	ID        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Gender    string `xml:"gender"`
	About     string `xml:"about"`
}

type xmlRoot struct {
	Users []xmlUser `xml:"row"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	if requestDelay > 0 {
		time.Sleep(requestDelay)
	}

	if r.Header.Get("AccessToken") == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil || limit < 0 {
		sendError(w, http.StatusBadRequest, "limit invalid")
		return
	}
	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil || offset < 0 {
		sendError(w, http.StatusBadRequest, "offset invalid")
		return
	}

	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil || (orderBy != OrderByAsc && orderBy != OrderByDesc && orderBy != OrderByAsIs) {
		sendError(w, http.StatusBadRequest, "order_by invalid")
		return
	}

	orderField, ok := normalizeOrderField(r.FormValue("order_field"))
	if !ok {
		sendError(w, http.StatusBadRequest, ErrorBadOrderField)
		return
	}

	users, err := loadUsers(datasetPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	query := r.FormValue("query")
	found := make([]User, 0)
	for _, user := range users {
		if query == "" || strings.Contains(user.Name, query) || strings.Contains(user.About, query) {
			found = append(found, user)
		}
	}

	sortUsers(found, orderField, orderBy)

	if offset > len(found) {
		offset = len(found)
	}
	end := offset + limit
	if end > len(found) {
		end = len(found)
	}
	sendJSON(w, http.StatusOK, found[offset:end])
}

func normalizeOrderField(field string) (string, bool) {
	switch strings.ToLower(field) {
	case "", "name":
		return "Name", true
	case "id":
		return "Id", true
	case "age":
		return "Age", true
	default:
		return "", false
	}
}

func sortUsers(users []User, field string, orderBy int) {
	if orderBy == OrderByAsIs {
		return
	}
	sort.SliceStable(users, func(i, j int) bool {
		left, right := users[i], users[j]
		var less bool
		switch field {
		case "Id":
			less = left.ID < right.ID
		case "Age":
			less = left.Age < right.Age
		default:
			less = left.Name < right.Name
		}
		if orderBy == OrderByDesc {
			return !less && !sameOrderKey(left, right, field)
		}
		return less
	})
}

func sameOrderKey(left, right User, field string) bool {
	switch field {
	case "Id":
		return left.ID == right.ID
	case "Age":
		return left.Age == right.Age
	default:
		return left.Name == right.Name
	}
}

func loadUsers(path string) ([]User, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var root xmlRoot
	if err := xml.NewDecoder(file).Decode(&root); err != nil {
		return nil, err
	}

	users := make([]User, 0, len(root.Users))
	for _, row := range root.Users {
		users = append(users, User{
			ID:     row.ID,
			Name:   row.FirstName + " " + row.LastName,
			Age:    row.Age,
			About:  row.About,
			Gender: row.Gender,
		})
	}
	return users, nil
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, SearchErrorResponse{Error: message})
}

func sendJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
