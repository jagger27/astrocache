package transport

import (
	"encoding/json"
	"net/http"

	"github.com/astromechio/astrocache/logger"
	"github.com/pkg/errors"
)

// ReplyWithJSON replies with json and 200
func ReplyWithJSON(w http.ResponseWriter, value interface{}) {
	response, err := json.Marshal(value)
	if err != nil {
		logger.LogError(errors.Wrap(err, "ReplyWithJSON failed to Marshal"))
		InternalServerError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

// Ok replies with 200
func Ok(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Created replies with 201
func Created(w http.ResponseWriter) {
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Created"))
}

// Conflict replies with 409
func Conflict(w http.ResponseWriter) {
	http.Error(w, "Conflict", http.StatusConflict)
}

// InternalServerError respoonds with 500
func InternalServerError(w http.ResponseWriter) {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// NotFound responds with 404
func NotFound(w http.ResponseWriter) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

// BadRequest responds with 400
func BadRequest(w http.ResponseWriter) {
	http.Error(w, "Bad Request", http.StatusBadRequest)
}

// Forbidden responds with 403
func Forbidden(w http.ResponseWriter) {
	http.Error(w, "Forbidden", http.StatusForbidden)
}
