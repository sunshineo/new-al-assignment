package main

import "net/http"

type Route struct {
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"POST",
		"/register",
		Register,
	},
	Route{
		"GET",
		"/",
		Index,
	},
	Route{
		"GET",
		"/todos",
		TodoIndex,
	},
	Route{
		"POST",
		"/todos",
		TodoCreate,
	},
	Route{
		"GET",
		"/todos/{todoId}",
		TodoShow,
	},
}
