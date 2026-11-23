// Copyright (c) CircleCI
// SPDX-License-Identifier: MPL-2.0

package fakecircle

import (
	"net/http"
	"strings"
)

// auth rejects any request that does not carry the fake's token, either as a
// Circle-Token header or as a bearer token.
func (s *Service) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("Circle-Token")
		if tok == "" {
			tok = bearerToken(r)
		}
		switch tok {
		case "":
			msg(w, r, http.StatusUnauthorized, "You must log in first.")
		case s.tok:
			next.ServeHTTP(w, r)
		default:
			msg(w, r, http.StatusUnauthorized, "Invalid token provided.")
		}
	})
}

// bearerOnly rejects requests that authenticate with Circle-Token, as the real
// V3 API does.
func (s *Service) bearerOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bearerToken(r) != s.tok {
			v3Error(w, r, http.StatusUnauthorized, "missing, invalid or expired token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return tok
}
