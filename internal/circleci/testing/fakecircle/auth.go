// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import "net/http"

// auth rejects any request that does not carry the fake's Circle-Token.
func (s *Service) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Circle-Token") {
		case "":
			msg(w, r, http.StatusUnauthorized, "You must log in first.")
		case s.tok:
			next.ServeHTTP(w, r)
		default:
			msg(w, r, http.StatusUnauthorized, "Invalid token provided.")
		}
	})
}
