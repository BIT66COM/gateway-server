package handlers

import (
	"net/http"
)

// Credit goes to https://github.com/stellar/horizon
func StripTrailingSlashMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			l := len(path)

			// if the path is longer than 1 char (i.e., not '/')
			// and has a trailing slash, remove it.
			if l > 1 && path[l-1] == '/' {
				r.URL.Path = path[0 : l-1]
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func HeadersMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func ApiKeyMiddleware(apiKey string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			k := r.PostFormValue("apiKey")
			if k != apiKey {
				errorForbidden(w, errorResponseString("forbidden", "Access denied."))
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
