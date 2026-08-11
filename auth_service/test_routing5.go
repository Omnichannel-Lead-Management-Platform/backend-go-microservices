package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"github.com/go-chi/chi/v5"
	"github.com/aarondl/authboss/v3/defaults"
)

func main() {
	r := defaults.NewRouter()
	r.Post("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("HIT /login")
	}))
	
	router := chi.NewRouter()
	router.Mount("/api/auth", http.StripPrefix("/api/auth", r))
	
	req1 := httptest.NewRequest("POST", "/api/auth/login", nil)
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)
	fmt.Println("Status Code /api/auth/login:", rr1.Code)
}
