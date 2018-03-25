package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"regexp"
	"database/sql"
	"log"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
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
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1024)) // This will limit the whole thing down to 1MB. Should be enough
	if err != nil {
		sendError("Post body too large.", w)
		log.Print(err)
		return
	}
	if err := r.Body.Close(); err != nil {
		sendError("Cannot close the request body", w)
		log.Print(err)
		return
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		sendError("Failed to parse the post body as JSON.", w)
		log.Print(err)
		return
	}

	username := user.Username
	if usernameLength := len(username); usernameLength < 3 || usernameLength > 20 {
		sendError("Usernames must be at least 3 characters and no more than 20", w)
		return
	}

	if !ValidUsername(username) {
		sendError("Usernames may only contain alphanumeric characters", w)
		return
	}

	password := user.Password
	if passwordLength := len(user.Password); passwordLength < 8 {
		sendError("Password must be at least 8 characters", w)
		return
	}

	connStr := "postgres://storage-user:storage-password@localhost:5432/postgres?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		sendError("Failed to connect to database", w)
		log.Print(err)
		return
	}

	rows, err := db.Query("SELECT count(*) FROM account WHERE username = $1", username)
	if err != nil {
		sendError("Failed to query the database", w)
		log.Print(err)
		return
	}
	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			sendError("Failed to get count from the query result", w)
			log.Print(err)
		}
	}
	if count > 0 {
		sendError("This username already exists", w)
		return
	}

	_, err = db.Query("INSERT INTO account VALUES($1, $2)", username, password)
	if err != nil {
		sendError("Failed to insert the user to database", w)
		log.Print(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(204)
}

