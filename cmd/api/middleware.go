package main

import (
	"fmt"
	"net/http"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// If there was a panic, set a connection close header on the respone, this acts as a trigger
				// to automatically close the connection after the response has been sent
				w.Header().Set("Connection", "close")
				// The value returned by the recover() has the type any, so can use fmt.Errorf()
				// to normalize it into an error and call our serverErrorReponse method on it.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
