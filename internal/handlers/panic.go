package handlers

import (
	"fmt"
	"net/http"

	"github.com/pojntfx/stfs/internal/logging"
)

func PanicHandler(h http.Handler, log *logging.JSONLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)

				log.Error("Error during HTTP request", map[string]interface{}{
					"err": err,
				})
			}
		}()

		h.ServeHTTP(w, r)
	})
}
