package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func InitializeRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = AccessLogger(handler)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Handler(handler)

	}

	return router
}
