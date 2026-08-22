package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aarondl/authboss/v3"
	"github.com/omnichannel/common/api"
)

// APIResponder implements authboss.Responder to output standard JSON
type APIResponder struct{}

func NewAPIResponder() *APIResponder {
	return &APIResponder{}
}

func (a *APIResponder) Respond(w http.ResponseWriter, r *http.Request, code int, templateName string, data authboss.HTMLData) error {
	for k, v := range data {
		fmt.Printf("DEBUG: HTMLData key: %s, type: %T\n", k, v)
	}
	
	// Authboss stores validation errors in the "errors" map
	if errsRaw, ok := data["errors"]; ok {
		if errsMap, ok := errsRaw.(map[string][]string); ok && len(errsMap) > 0 {
			var errorMessages []string
			for field, msgs := range errsMap {
				for _, msg := range msgs {
					errorMessages = append(errorMessages, fmt.Sprintf("%s: %v", field, msg))
				}
			}
			
			finalErrorMessage := strings.Join(errorMessages, ", ")
			// We force a 400 Bad Request because validation failed
			api.Error(w, http.StatusBadRequest, finalErrorMessage)
			return nil
		}
	}

	// Check for a top-level error (e.g., login failed)
	if errMsg, ok := data[authboss.DataErr]; ok {
		api.Error(w, http.StatusUnauthorized, fmt.Sprintf("%v", errMsg))
		return nil
	}
	
	// If it's a success, check for a success message
	successMsg := "Success"
	if msg, ok := data[authboss.FlashSuccessKey]; ok {
		successMsg = fmt.Sprintf("%v", msg)
	}

	// Default to 200 OK for successful requests
	if code == 0 {
		code = http.StatusOK
	}

	// We can safely return a simple map
	responseData := map[string]interface{}{}
	
	// If the JWTReadWriter generated a token, extract it and include it in the JSON body
	if token := w.Header().Get("X-Access-Token"); token != "" {
		responseData["token"] = token
	}

	api.Success(w, responseData, successMsg)
	return nil
}

// Redirect intercepts Authboss successful actions (like successful login or registration)
// and returns a standardized JSON 200 OK instead of actually doing an HTTP 302/307 redirect.
func (a *APIResponder) Redirect(w http.ResponseWriter, r *http.Request, ro authboss.RedirectOptions) error {
	fmt.Println("CUSTOM REDIRECTOR CALLED!!!")
	successMsg := ro.Success
	if successMsg == "" {
		successMsg = "Success"
	}
	
	responseData := map[string]interface{}{}
	
	if token := w.Header().Get("X-Access-Token"); token != "" {
		responseData["token"] = token
	}

	api.Success(w, responseData, successMsg)
	return nil
}

