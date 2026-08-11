package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

func main() {
	router := chi.NewRouter()
	
	router.Mount("/api/auth", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Mounted Path received:", r.URL.Path)
	}))
	
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	rr := httptest.NewRecorder()
	
	router.ServeHTTP(rr, req)
}
