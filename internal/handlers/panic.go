package handlers

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

func PanicHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				http.Error(w, fmt.Sprintf("%v", r), http.StatusInternalServerError)

				log.Println("Error:", r, "\nStack:", string(debug.Stack()))
			}
		}()

		h.ServeHTTP(w, r)
	})
}
