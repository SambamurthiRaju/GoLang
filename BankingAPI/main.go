package main

import (
	"log"
	"net/http"

	"BankingAPI/docs"
	httpserver "BankingAPI/internal/httpserver"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Banking API
// @version 1.0
// @description Simple in-memory banking API with JWT authentication.
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	srv := httpserver.NewServer()
	docs.SwaggerInfo.BasePath = "/"

	// register swagger endpoint
	http.Handle("/swagger/", httpSwagger.WrapHandler)

	// attach mux from server (server includes router with middleware)
	http.Handle("/", srv.Router())

	addr := ":8080"
	log.Printf("listening on %s (swagger: http://localhost%s/swagger/index.html)\n", addr, addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
