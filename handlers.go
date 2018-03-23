package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"regexp"

	"github.com/gorilla/mux"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome!\n")
}

func TodoIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(todos); err != nil {
		panic(err)
	}
}

func TodoShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var todoId int
	var err error
	if todoId, err = strconv.Atoi(vars["todoId"]); err != nil {
		panic(err)
	}
	todo := RepoFindTodo(todoId)
	if todo.Id > 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(todo); err != nil {
			panic(err)
		}
		return
	}

	// If we didn't find it, 404
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
		panic(err)
	}

}

/*
Test with this curl command:

curl -H "Content-Type: application/json" -d '{"name":"New Todo"}' http://localhost:8080/todos

*/
func TodoCreate(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &todo); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	t := RepoCreateTodo(todo)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(t); err != nil {
		panic(err)
	}
}

func sendError(msg string, w http.ResponseWriter) {
	errorBody := errorJson{Error: msg}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(400)
	if err := json.NewEncoder(w).Encode(errorBody); err != nil {
		panic(err)
	}
}

var ValidUsername = regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString

func Register(w http.ResponseWriter, r *http.Request) {
	var user User
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1024)) // This will limit the whole thing down to 1MB. Should be enough
	if err != nil {
		sendError("Post body too large.", w)
		return
	}
	if err := r.Body.Close(); err != nil {
		sendError("Cannot close the request body", w)
		return
	}

	if err := json.Unmarshal(body, &user); err != nil {
		sendError("Failed to parse the post body as JSON.", w)
		return
	}

	if usernameLength := len(user.Username); usernameLength < 3 || usernameLength > 20 {
		sendError("Usernames must be at least 3 characters and no more than 20", w)
		return
	}

	if !ValidUsername(user.Username) {
		sendError("Usernames may only contain alphanumeric characters", w)
		return
	}

	if passwordLength := len(user.Password); passwordLength < 8 {
		sendError("Password must be at least 8 characters", w)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(204)
}

