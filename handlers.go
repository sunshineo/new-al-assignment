package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"./models"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"github.com/gorilla/sessions"
)

func sendError(msg string, w http.ResponseWriter) {
	errorBody := models.ErrorResponse{Error: msg}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(400)
	if err := json.NewEncoder(w).Encode(errorBody); err != nil {
		panic(err)
	}
}

var (
	ValidUsername = regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString
	connStr = "postgres://storage-user:storage-password@localhost:5432/postgres?sslmode=disable"
	db, _ = sql.Open("postgres", connStr)
)

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

	var user models.User
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

	// Hash the password before save. bcrypt has salt already
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		sendError("Failed to hash the password", w)
		log.Print(err)
	}
	passwordHash := string(bytes)

	_, err = db.Query("INSERT INTO account VALUES($1, $2)", username, passwordHash)
	if err != nil {
		sendError("Failed to insert the user to database", w)
		log.Print(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(204)
}

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key = []byte("super-secret-key")
	store = sessions.NewCookieStore(key)
)

func Login(w http.ResponseWriter, r *http.Request) {
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

	var user models.User
	if err := json.Unmarshal(body, &user); err != nil {
		sendError("Failed to parse the post body as JSON.", w)
		log.Print(err)
		return
	}

	username := user.Username
	password := user.Password

	rows, err := db.Query("SELECT password FROM account WHERE username = $1", username)
	if err != nil {
		sendError("Failed to query the database", w)
		log.Print(err)
		return
	}

	var passwordHash string
	for rows.Next() {
		err := rows.Scan(&passwordHash)
		if err != nil {
			sendError("Failed to get passwordHash from the query result", w)
			log.Print(err)
		}
	}

	if len(passwordHash) == 0 {
		sendError("Could not find the user in the database", w)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		sendError("Incorrect password", w)
		log.Print(err)
	}

	session, err := store.Get(r, "storage-service-session")
	if err != nil {
		sendError("Failed to create a session", w)
		log.Print(err)
	}

	// Set user as authenticated
	session.Values["username"] = username
	err = session.Save(r, w)
	if err != nil {
		sendError("Failed to save the session", w)
		log.Print(err)
	}

	// log.Print(w.Header().Get("Set-Cookie"))
	responseBody := models.TokenResponse{Token: w.Header().Get("Set-Cookie")}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		panic(err)
	}
}

func GetUsernameFromSession(r *http.Request) string {
	session, err := store.Get(r, "storage-service-session")
	if err != nil {
		log.Print(err)
		return ""
	}

	// Retrieve our struct and type-assert it
	username := session.Values["username"].(string)
	return username
}

func PutFile(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromSession(r)
	if len(username) == 0 {
		sendError("You are not logged in", w)
		return
	}
	// log.Print(username)

	// Now we can use our person object
	vars := mux.Vars(r)
	filename := vars["filename"]
	// log.Print(filename)

	rows, err := db.Query("SELECT count(*) FROM file WHERE username = $1 AND filename =  $2", username, filename)
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
		sendError("This username + filename already exists", w)
		return
	}

	_, err = db.Query("INSERT INTO file VALUES($1, $2)", username, filename)
	if err != nil {
		sendError("Failed to insert the user's file record to database", w)
		log.Print(err)
	}

	path := "./files/" + username
	if _, err := os.Stat(path); os.IsNotExist(err) {
	    os.MkdirAll(path, os.ModePerm)
	}
	file, err := os.Create(path + "/" + filename)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(file, r.Body)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Location", filename)
	w.WriteHeader(201)
}

func GetFile(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromSession(r)
	if len(username) == 0 {
		sendError("You are not logged in", w)
		return
	}
	// log.Print(username)

	vars := mux.Vars(r)
	filename := vars["filename"]
	// log.Print(filename)

	rows, err := db.Query("SELECT filename FROM file WHERE username = $1 AND filename = $2", username, filename)
	if err != nil {
		sendError("Failed to query the database", w)
		log.Print(err)
		return
	}

	var filenameFromDB string
	for rows.Next() {
		err := rows.Scan(&filenameFromDB)
		if err != nil {
			sendError("Failed to get filename from the query result", w)
			log.Print(err)
		}
	}

	if len(filenameFromDB) == 0 {
		sendError("Could not find the file", w)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(200)
}

func DeleteFile(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromSession(r)
	if len(username) == 0 {
		sendError("You are not logged in", w)
		return
	}
	// log.Print(username)

	vars := mux.Vars(r)
	filename := vars["filename"]
	// log.Print(filename)

	_, err := db.Query("DELETE FROM file WHERE username = $1 AND filename = $2", username, filename)
	if err != nil {
		sendError("Failed to query the database", w)
		log.Print(err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(200)
}

func ListFiles(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromSession(r)
	if len(username) == 0 {
		sendError("You are not logged in", w)
		return
	}
	// log.Print(username)

	rows, err := db.Query("SELECT filename FROM file WHERE username = $1", username)
	if err != nil {
		sendError("Failed to query the database", w)
		log.Print(err)
		return
	}

	var files []string
	var filename string
	for rows.Next() {
		err := rows.Scan(&filename)
		if err != nil {
			sendError("Failed to get filename from the query result", w)
			log.Print(err)
		}
		files = append(files, filename)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(files); err != nil {
		panic(err)
	}
}
