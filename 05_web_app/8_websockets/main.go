package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/icrowley/fake"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	tmpl := template.Must(template.ParseFiles("index.html"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("render page: %v", err)
		}
	})

	mux.HandleFunc("GET /notifications", notificationsHandler)

	fmt.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func notificationsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade websocket: %v", err)
		return
	}

	go sendNewMsgNotifications(ws)
}

func sendNewMsgNotifications(client *websocket.Conn) {
	defer client.Close()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		w, err := client.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}

		msg := newMessage()
		if _, err := w.Write(msg); err != nil {
			w.Close()
			return
		}
		if err := w.Close(); err != nil {
			return
		}
	}
}

func newMessage() []byte {
	data, _ := json.Marshal(map[string]string{
		"email":   fake.EmailAddress(),
		"name":    fake.FirstName() + " " + fake.LastName(),
		"subject": fake.Product() + " " + fake.Model(),
	})

	return data
}
