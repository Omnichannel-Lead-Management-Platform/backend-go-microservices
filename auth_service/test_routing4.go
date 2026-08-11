package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"github.com/aarondl/authboss/v3/defaults"
)

func main() {
	r := defaults.NewRouter()
	
	r.Post("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("HIT /login")
	}))
	
	r.Post("/api/auth/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("HIT /api/auth/login")
	}))
	
	req1 := httptest.NewRequest("POST", "/login", nil)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	fmt.Println("Status Code /login:", rr1.Code)

	req2 := httptest.NewRequest("POST", "/api/auth/login", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	fmt.Println("Status Code /api/auth/login:", rr2.Code)
}
