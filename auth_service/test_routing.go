package main

import (
	"fmt"
	"net/http/httptest"
	"os"

	"github.com/aarondl/authboss/v3"
	"github.com/aarondl/authboss/v3/defaults"
	_ "github.com/aarondl/authboss/v3/auth"
)

func main() {
	ab := authboss.New()
	defaults.SetCore(&ab.Config, true, false)
	ab.Config.Paths.Mount = "/api/auth"
	ab.Config.Core.ViewRenderer = defaults.JSONRenderer{}
	ab.Config.Core.MailRenderer = defaults.JSONRenderer{}
	ab.Config.Core.Mailer = defaults.NewLogMailer(os.Stdout)
	
	ab.Init()
	
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	rr := httptest.NewRecorder()
	
	ab.Config.Core.Router.ServeHTTP(rr, req)
	
	fmt.Println("Status Code:", rr.Code)
}
