package main

import (
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
)

func main() {
	router := RegisterRoutes()
	handler := http.Handler(router)

	if os.Getenv("ENV") != "PROD" {
		// Enable CORS
		c := cors.New(cors.Options{
			AllowedOrigins:   []string{"http://localhost:5173"},
			AllowCredentials: true,
			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		})

		handler = c.Handler(router)
	}

	err := http.ListenAndServe("localhost:8080", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
