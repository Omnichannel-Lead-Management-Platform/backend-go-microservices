package api

import (
	"encoding/json"
	"net/http"
)

// ApiResponse defines a standard JSON format for all microservice responses.
type ApiResponse struct {
	Status  string      `json:"status"`            // "success" or "fail"
	Message string      `json:"message,omitempty"` // Optional success or error message
	Data    interface{} `json:"data,omitempty"`    // Optional payload
}

// Success writes a standard HTTP 200 JSON success response.
func Success(w http.ResponseWriter, data interface{}, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := ApiResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}

	json.NewEncoder(w).Encode(response)
}

// Error writes a standard JSON error response with the provided HTTP status code.
func Error(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ApiResponse{
		Status:  "fail",
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}
