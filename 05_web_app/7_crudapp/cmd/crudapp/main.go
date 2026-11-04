package main

import (
	"html/template"
	"log"
	"net/http"

	"crudapp/internal/handlers"
	"crudapp/internal/items"
	"crudapp/internal/middleware"
	"crudapp/internal/session"
	"crudapp/internal/user"

	"go.uber.org/zap"
)

func main() {
	templates := template.Must(template.ParseGlob("./templates/*"))

	sm := session.NewSessionsManager()
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer zapLogger.Sync() // flushes buffer, if any
	logger := zapLogger.Sugar()

	userRepo := user.NewMemoryRepo()
	itemsRepo := items.NewMemoryRepo()

	userHandler := &handlers.UserHandler{
		Tmpl:     templates,
		UserRepo: userRepo,
		Logger:   logger,
		Sessions: sm,
	}

	handlers := &handlers.ItemsHandler{
		Tmpl:      templates,
		Logger:    logger,
		ItemsRepo: itemsRepo,
	}

	r := http.NewServeMux()
	r.HandleFunc("GET /", userHandler.Index)
	r.HandleFunc("POST /login", userHandler.Login)
	r.HandleFunc("POST /logout", userHandler.Logout)

	r.HandleFunc("GET /items", handlers.List)
	r.HandleFunc("GET /items/new", handlers.AddForm)
	r.HandleFunc("POST /items/new", handlers.Add)
	r.HandleFunc("GET /items/{id}", handlers.Edit)
	r.HandleFunc("POST /items/{id}", handlers.Update)
	r.HandleFunc("DELETE /items/{id}", handlers.Delete)

	mux := middleware.Auth(sm, r)
	mux = middleware.AccessLog(logger, mux)
	mux = middleware.Panic(mux)

	addr := ":8080"
	logger.Infow("starting server",
		"type", "START",
		"addr", addr,
	)
	if err := http.ListenAndServe(addr, mux); err != nil {
		zapLogger.Sync()
		log.Fatal(err)
	}
}
