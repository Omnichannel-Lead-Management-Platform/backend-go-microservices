package main

import (
	"fmt"
	"net/http/httptest"
	"os"

	"github.com/aarondl/authboss/v3"
	"github.com/aarondl/authboss/v3/defaults"
	_ "github.com/aarondl/authboss/v3/auth"
	"github.com/go-chi/chi/v5"
)

func main() {
	ab := authboss.New()
	defaults.SetCore(&ab.Config, true, false)
	ab.Config.Paths.Mount = "/api/auth"
	ab.Config.Core.ViewRenderer = defaults.JSONRenderer{}
	ab.Config.Core.MailRenderer = defaults.JSONRenderer{}
	ab.Config.Core.Mailer = defaults.NewLogMailer(os.Stdout)
	ab.Init()
	
	router := chi.NewRouter()
	
	router.Mount("/api/auth", ab.Config.Core.Router)
	// OR router.Mount("/", ab.Config.Core.Router) 
	
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	rr := httptest.NewRecorder()
	
	router.ServeHTTP(rr, req)
	
	fmt.Println("Status Code:", rr.Code)
}
