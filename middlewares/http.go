package middlewares

import "net/http"

// RecoverHTTP ...
func RecoverHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recover()
		next.ServeHTTP(w, r)
	})
}
