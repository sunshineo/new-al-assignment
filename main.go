package main

import (
	"log"
	"net/http"
)

func main() {

	router := InitializeRouter()
	log.Fatal(http.ListenAndServe(":8080", router))
}
