// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

// respond writes v as a JSON body with the given status code.
func respond(w http.ResponseWriter, r *http.Request, statusCode int, v any) {
	render.Status(r, statusCode)
	render.JSON(w, r, v)
}

// msg writes a JSON body of the form {"message": "..."}, which is how the real
// API reports most errors.
func msg(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	respond(w, r, statusCode, struct {
		Message string `json:"message"`
	}{
		Message: message,
	})
}

// badRequest writes message as a 400 and returns true when err is non-nil, so
// callers can `if badRequest(...) { return }`. It returns false when err is nil.
func badRequest(w http.ResponseWriter, r *http.Request, message string, err error) bool {
	if err == nil {
		return false
	}
	slog.Warn(err.Error())
	msg(w, r, http.StatusBadRequest, message)
	return true
}

type listResponse[T any] struct {
	NextPageToken *string `json:"next_page_token"`
	Items         []T     `json:"items"`
}

func newListResponse[T any](items []T) listResponse[T] {
	return listResponse[T]{Items: items}
}
