package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExplorerValueValidation(t *testing.T) {
	cases := []struct {
		name, kind string
		nullable   bool
		value      any
		valid      bool
	}{
		{"integer", "int", false, json.Number("42"), true},
		{"fractional integer", "int", false, json.Number("1.5"), false},
		{"integer overflow", "int", false, json.Number("18446744073709551616"), false},
		{"negative integer", "int", false, json.Number("-2"), true},
		{"floating point", "float", false, json.Number("1.25"), true},
		{"numeric string", "int", false, "42", false},
		{"nullable", "string", true, nil, true},
		{"required", "string", false, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := validateExplorerValue(explorerColumn{kind: tc.kind, nullable: tc.nullable}, tc.value)
			if ok != tc.valid {
				t.Fatalf("validation = %v, want %v", ok, tc.valid)
			}
		})
	}
}

// This table deliberately differs from both tables in the supplied tests.
func TestExplorerDynamicSchema(t *testing.T) {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE hw06_products (product_key BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, quantity INT NOT NULL, price DECIMAL(10,2) NOT NULL, note TEXT NULL, label VARCHAR(32) NOT NULL DEFAULT 'untitled')")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = db.Exec("DROP TABLE hw06_products") }()
	handler, err := NewDBExplorer(db)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	runCases(t, server, db, []Case{
		{Method: http.MethodPut, Path: "/hw06_products", Body: CR{"quantity": 3, "price": 2.5, "extra": "ignored", "product_key": 99}, Result: CR{"response": CR{"product_key": 1}}},
		{Path: "/hw06_products/1", Result: CR{"response": CR{"record": CR{"product_key": 1, "quantity": 3, "price": 2.5, "note": nil, "label": "untitled"}}}},
		{Method: http.MethodPost, Path: "/hw06_products/1", Body: CR{"quantity": 1.5}, Status: 400, Result: CR{"error": "field quantity have invalid type"}},
		{Method: http.MethodPost, Path: "/hw06_products/1", Body: CR{"price": 3.75, "note": "quote' and slash\\"}, Result: CR{"response": CR{"updated": 1}}},
		{Path: "/hw06_products/1", Result: CR{"response": CR{"record": CR{"product_key": 1, "quantity": 3, "price": 3.75, "note": "quote' and slash\\", "label": "untitled"}}}},
		{Path: "/hw06_products", Query: "limit=0", Result: CR{"response": CR{"records": []CR{}}}},
		{Method: http.MethodDelete, Path: "/hw06_products/1", Result: CR{"response": CR{"deleted": 1}}},
		{Path: "/hw06_products/1", Status: 404, Result: CR{"error": "record not found"}},
	})
}

func TestExplorerManualPrimaryKey(t *testing.T) {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("CREATE TABLE hw06_manual_keys (code INT NOT NULL DEFAULT 17 PRIMARY KEY, title VARCHAR(32) NOT NULL)")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = db.Exec("DROP TABLE hw06_manual_keys") }()
	handler, err := NewDBExplorer(db)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	runCases(t, server, db, []Case{
		{Method: http.MethodPut, Path: "/hw06_manual_keys", Body: CR{"code": 42, "title": "explicit"}, Result: CR{"response": CR{"code": 42}}},
		{Path: "/hw06_manual_keys/42", Result: CR{"response": CR{"record": CR{"code": 42, "title": "explicit"}}}},
		{Method: http.MethodPut, Path: "/hw06_manual_keys", Body: CR{"code": 0, "title": "zero"}, Result: CR{"response": CR{"code": 0}}},
		{Path: "/hw06_manual_keys/0", Result: CR{"response": CR{"record": CR{"code": 0, "title": "zero"}}}},
		{Method: http.MethodPut, Path: "/hw06_manual_keys", Body: CR{"code": -7, "title": "negative"}, Result: CR{"response": CR{"code": -7}}},
		{Path: "/hw06_manual_keys/-7", Result: CR{"response": CR{"record": CR{"code": -7, "title": "negative"}}}},
		{Method: http.MethodPut, Path: "/hw06_manual_keys", Body: CR{"title": "default"}, Result: CR{"response": CR{"code": 17}}},
		{Path: "/hw06_manual_keys/17", Result: CR{"response": CR{"record": CR{"code": 17, "title": "default"}}}},
	})
}
