package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteJSON writes a JSON response with the specified status code.
// It sets the Content-Type header to application/json and encodes the data as JSON.
// Returns an error if encoding fails.
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("failed to encode response to json: %w", err)
	}

	return nil
}
