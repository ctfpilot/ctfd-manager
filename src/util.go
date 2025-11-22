package main

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	log.Printf("Error response: Path: %s - Status: %d - Message: %s", r.URL.Path, status, message)

	response := ErrorResponse{Message: message}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling error response: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(jsonResponse); err != nil {
		log.Printf("Error writing response: %s", err)
	}
}

func stringStrip(str string) string {
	// Strip leading and trailing whitespace from a string
	return strings.TrimSpace(str)
}

func validatePassword(password string) bool {
	// Load data from env
	valid_password := os.Getenv("PASSWORD")

	password = stringStrip(password)
	valid_password = stringStrip(valid_password)

	// Compare the provided password with the valid password
	equal := subtle.ConstantTimeCompare([]byte(password), []byte(valid_password)) == 1

	return equal
}

// Convert string formatted json to map
func stringToMap(str string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(str), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
