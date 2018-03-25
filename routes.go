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
		"POST",
		"/login",
		Login,
	},
	Route{
		"PUT",
		"/files/{filename}",
		PutFile,
	},
	Route{
		"GET",
		"/files/{filename}",
		GetFile,
	},
	Route{
		"DELETE",
		"/files/{filename}",
		DeleteFile,
	},
	Route{
		"GET",
		"/files",
		ListFiles,
	},
}
