package main

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
)

type User struct {
	ID       int
	Username string
	phone    string
}

var jsonStr = `{"ID": 42, "username": "rvasily", "phone": "123", "ID": 100500}`

func main() {
	data := []byte(jsonStr)

	vasya := User{
		ID:       12345,
		Username: "Вася <head>",
		phone:    "987654321",
	}

	u1 := &User{}
	err := json.Unmarshal(data, u1)
	if err != nil {
		fmt.Printf("unmarshal error: %v\n", err)
	}
	fmt.Printf("struct:\n\t%#v\n\n", u1)

	result, err := json.Marshal(vasya)
	if err != nil {
		fmt.Printf("marshal error: %v\n", err)
	}
	fmt.Printf("json string:\n\t%s\n", string(result))

	fmt.Printf("\n\n---\n\n")

	u2 := &User{}
	err = jsonv2.Unmarshal(data, u2) // jsonv2.MatchCaseInsensitiveNames(true)
	if err != nil {
		fmt.Printf("unmarshal error: %v\n", err)
	}
	fmt.Printf("struct:\n\t%#v\n\n", u2)

	result, err = jsonv2.Marshal(vasya, jsontext.EscapeForHTML(false))
	if err != nil {
		fmt.Printf("marshal error: %v\n", err)
	}
	fmt.Printf("json string:\n\t%s\n", string(result))
}
