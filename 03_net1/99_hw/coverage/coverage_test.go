package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFindUsersValidation(t *testing.T) {
	client := &SearchClient{AccessToken: "token", URL: "http://127.0.0.1"}

	_, err := client.FindUsers(SearchRequest{Limit: -1})
	if err == nil || !strings.Contains(err.Error(), "limit must be > 0") {
		t.Fatalf("limit: got %v", err)
	}

	_, err = client.FindUsers(SearchRequest{Offset: -1})
	if err == nil || !strings.Contains(err.Error(), "offset must be > 0") {
		t.Fatalf("offset: got %v", err)
	}
}

func TestFindUsersSearch(t *testing.T) {
	client := newSearchClient(t, SearchServer)

	boyd, err := client.FindUsers(SearchRequest{
		Limit:      5,
		Query:      "Boyd",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(boyd.Users) != 1 || boyd.NextPage {
		t.Fatalf("Boyd result: %+v", boyd)
	}
	if boyd.Users[0].ID != 0 || boyd.Users[0].Name != "Boyd Wolf" || boyd.Users[0].Age != 22 || boyd.Users[0].Gender != "male" {
		t.Fatalf("unexpected user: %+v", boyd.Users[0])
	}
	if !strings.Contains(boyd.Users[0].About, "Nulla cillum") {
		t.Fatalf("about was not loaded: %q", boyd.Users[0].About)
	}

	empty, err := client.FindUsers(SearchRequest{
		Limit:   1,
		Query:   "this-name-does-not-exist",
		OrderBy: OrderByAsIs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Users) != 0 || empty.NextPage {
		t.Fatalf("empty query should miss everyone: %+v", empty)
	}

	byID, err := client.FindUsers(SearchRequest{
		Limit:      1,
		Offset:     1,
		OrderField: "Id",
		OrderBy:    OrderByAsc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !byID.NextPage || len(byID.Users) != 1 || byID.Users[0].ID != 1 || byID.Users[0].Name != "Hilda Mayer" {
		t.Fatalf("offset/limit: %+v", byID)
	}

	all := collectUsers(t, client, SearchRequest{Limit: 25, OrderBy: OrderByAsIs})
	if len(all) != 35 {
		t.Fatalf("dataset size: got %d", len(all))
	}
	if all[0].ID != 0 || all[0].Name != "Boyd Wolf" {
		t.Fatalf("file order starts with %+v", all[0])
	}

	byName := collectUsers(t, client, SearchRequest{Limit: 25, OrderField: "Name", OrderBy: OrderByAsc})
	for i := 1; i < len(byName); i++ {
		if byName[i].Name < byName[i-1].Name {
			t.Fatalf("name order broken at %s < %s", byName[i].Name, byName[i-1].Name)
		}
	}

	byAge := collectUsers(t, client, SearchRequest{
		Limit:      25,
		Query:      "on",
		OrderField: "age",
		OrderBy:    OrderByDesc,
	})
	if len(byAge) == 0 {
		t.Fatal("query on returned nothing")
	}
	for _, user := range byAge {
		if !strings.Contains(user.Name, "on") && !strings.Contains(user.About, "on") {
			t.Fatalf("%s does not match query", user.Name)
		}
	}
	maxAge := byAge[0].Age
	for _, user := range byAge[1:] {
		if user.Age > maxAge {
			t.Fatalf("age %d is above first user %d", user.Age, maxAge)
		}
	}

	capped, err := client.FindUsers(SearchRequest{Limit: 100, OrderField: "Id", OrderBy: OrderByDesc})
	if err != nil {
		t.Fatal(err)
	}
	if len(capped.Users) != 25 || !capped.NextPage || capped.Users[0].ID != 34 {
		t.Fatalf("limit cap: %+v first=%+v", capped.NextPage, capped.Users[0])
	}

	_, err = client.FindUsers(SearchRequest{Limit: 1, OrderField: "Gender", OrderBy: OrderByAsc})
	if err == nil || !strings.Contains(err.Error(), "OrderFeld Gender invalid") {
		t.Fatalf("bad order field: %v", err)
	}

	_, err = client.FindUsers(SearchRequest{Limit: 1, OrderBy: 2})
	if err == nil || !strings.Contains(err.Error(), "unknown bad request error") {
		t.Fatalf("bad order_by: %v", err)
	}
}

func TestFindUsersServerErrors(t *testing.T) {
	client := newSearchClient(t, SearchServer)

	client.AccessToken = ""
	_, err := client.FindUsers(SearchRequest{Limit: 1})
	if err == nil || !strings.Contains(err.Error(), "bad AccessToken") {
		t.Fatalf("token: %v", err)
	}
	client.AccessToken = "token"

	originalPath := datasetPath
	datasetPath = filepath.Join(t.TempDir(), "missing.xml")
	t.Cleanup(func() { datasetPath = originalPath })

	_, err = client.FindUsers(SearchRequest{Limit: 1})
	if err == nil || !strings.Contains(err.Error(), "SearchServer fatal error") {
		t.Fatalf("missing file: %v", err)
	}
}

func TestFindUsersBrokenPayloads(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{
			name: "bad error json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("not-json"))
			},
			want: "cant unpack error json",
		},
		{
			name: "unknown bad request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(SearchErrorResponse{Error: "boom"})
			},
			want: "unknown bad request error: boom",
		},
		{
			name: "bad result json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{"))
			},
			want: "cant unpack result json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newSearchClient(t, tc.handler)
			_, err := client.FindUsers(SearchRequest{Limit: 1})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestFindUsersTimeoutAndNetwork(t *testing.T) {
	requestDelay = 1500 * time.Millisecond
	t.Cleanup(func() { requestDelay = 0 })

	client := newSearchClient(t, SearchServer)
	_, err := client.FindUsers(SearchRequest{Limit: 1, OrderBy: OrderByAsIs})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("timeout: %v", err)
	}

	closed := httptest.NewServer(http.HandlerFunc(SearchServer))
	closed.Close()
	client = &SearchClient{AccessToken: "token", URL: closed.URL}
	_, err = client.FindUsers(SearchRequest{Limit: 1})
	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Fatalf("network: %v", err)
	}
}

func TestSearchServerRequestErrors(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		token  string
		status int
		error  string
	}{
		{name: "no token", query: "limit=1&offset=0&order_by=0", status: http.StatusUnauthorized},
		{name: "bad limit", query: "limit=no&offset=0&order_by=0", token: "token", status: http.StatusBadRequest, error: "limit invalid"},
		{name: "negative offset", query: "limit=1&offset=-1&order_by=0", token: "token", status: http.StatusBadRequest, error: "offset invalid"},
		{name: "bad order field", query: "limit=1&offset=0&order_by=1&order_field=About", token: "token", status: http.StatusBadRequest, error: ErrorBadOrderField},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := doSearch(t, tc.query, tc.token)
			if status != tc.status {
				t.Fatalf("status %d body %s", status, body)
			}
			if tc.error != "" && !strings.Contains(body, tc.error) {
				t.Fatalf("body %s", body)
			}
		})
	}

	broken := filepath.Join(t.TempDir(), "broken.xml")
	if err := os.WriteFile(broken, []byte("<root>"), 0o644); err != nil {
		t.Fatal(err)
	}
	originalPath := datasetPath
	datasetPath = broken
	t.Cleanup(func() { datasetPath = originalPath })

	status, _ := doSearch(t, "limit=1&offset=0&order_by=0", "token")
	if status != http.StatusInternalServerError {
		t.Fatalf("broken xml status %d", status)
	}
}

func newSearchClient(t *testing.T, handler http.HandlerFunc) *SearchClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &SearchClient{AccessToken: "token", URL: server.URL}
}

func collectUsers(t *testing.T, client *SearchClient, req SearchRequest) []User {
	t.Helper()
	var users []User
	for {
		resp, err := client.FindUsers(req)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, resp.Users...)
		if !resp.NextPage {
			return users
		}
		req.Offset += len(resp.Users)
	}
}

func doSearch(t *testing.T, rawQuery, token string) (int, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"?"+rawQuery, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("AccessToken", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body := make([]byte, 0)
	buf := make([]byte, 512)
	for {
		n, readErr := resp.Body.Read(buf)
		body = append(body, buf[:n]...)
		if readErr != nil {
			break
		}
	}
	return resp.StatusCode, string(body)
}
