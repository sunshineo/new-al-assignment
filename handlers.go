package main

import (
	"bytes"
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

func LogAndSendError(err interface{}, status int, msg string, w http.ResponseWriter) {
	if err != nil {
		log.Print(err)
	}
	errorBody := models.ErrorResponse{Error: msg}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorBody); err != nil {
		panic(err)
	}
}

var (
	ValidUsername = regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString
	connStr = "postgres://storage-user:storage-password@postgres:5432/postgres?sslmode=disable"
	db, _ = sql.Open("postgres", connStr)
)

func Register(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1024)) // This will limit the whole thing down to 1MB. Should be enough
	if err != nil {
		LogAndSendError(err, 400, "Post body too large.", w)
		return
	}
	if err := r.Body.Close(); err != nil {
		LogAndSendError(err, 400, "Cannot close the request body", w)
		return
	}

	var user models.User
	if err := json.Unmarshal(body, &user); err != nil {
		LogAndSendError(err, 400, "Failed to parse the post body as JSON.", w)
		return
	}

	username := user.Username
	if usernameLength := len(username); usernameLength < 3 || usernameLength > 20 {
		LogAndSendError(err, 400, "Usernames must be at least 3 characters and no more than 20", w)
		return
	}

	if !ValidUsername(username) {
		LogAndSendError(err, 400, "Usernames may only contain alphanumeric characters", w)
		return
	}

	password := user.Password
	if passwordLength := len(user.Password); passwordLength < 8 {
		LogAndSendError(err, 400, "Password must be at least 8 characters", w)
		return
	}

	rows, err := db.Query("SELECT count(*) FROM account WHERE username = $1", username)
	if err != nil {
		LogAndSendError(err, 400, "Failed to query the database", w)
		return
	}
	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			LogAndSendError(err, 400, "Failed to get count from the query result", w)
		}
	}
	if count > 0 {
		LogAndSendError(err, 400, "This username already exists", w)
		return
	}

	// Hash the password before save. bcrypt has salt already
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		LogAndSendError(err, 400, "Failed to hash the password", w)
	}
	passwordHash := string(bytes)

	_, err = db.Query("INSERT INTO account VALUES($1, $2)", username, passwordHash)
	if err != nil {
		LogAndSendError(err, 400, "Failed to insert the user to database", w)
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
		LogAndSendError(err, 400, "Post body too large.", w)
		return
	}
	if err := r.Body.Close(); err != nil {
		LogAndSendError(err, 400, "Cannot close the request body", w)
		return
	}

	var user models.User
	if err := json.Unmarshal(body, &user); err != nil {
		LogAndSendError(err, 400, "Failed to parse the post body as JSON.", w)
		return
	}

	username := user.Username
	password := user.Password

	rows, err := db.Query("SELECT password FROM account WHERE username = $1", username)
	if err != nil {
		LogAndSendError(err, 400, "Failed to query the database", w)
		return
	}

	var passwordHash string
	for rows.Next() {
		err := rows.Scan(&passwordHash)
		if err != nil {
			LogAndSendError(err, 400, "Failed to get passwordHash from the query result", w)
		}
	}

	if len(passwordHash) == 0 {
		LogAndSendError(err, 403, "Could not find the user in the database", w)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		LogAndSendError(err, 403, "Incorrect password", w)
	}

	session, err := store.Get(r, "Session")
	if err != nil {
		LogAndSendError(err, 400, "Failed to create a session", w)
	}

	// Set user as authenticated
	session.Values["username"] = username
	err = session.Save(r, w)
	if err != nil {
		LogAndSendError(err, 400, "Failed to save the session", w)
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
	xSession := r.Header.Get("X-Session")
	if xSession != "" {
		r.Header.Set("Cookie", xSession)
	}
	session, err := store.Get(r, "Session")
	if err != nil {
		return ""
	}

	// Retrieve our struct and type-assert it
	username := session.Values["username"]
	if username == nil {
		return ""
	}
	return username.(string)
}

func PutFile(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromSession(r)
	if len(username) == 0 {
		LogAndSendError(nil, 403, "You are not logged in", w)
		return
	}
	// log.Print(username)

	// Now we can use our person object
	vars := mux.Vars(r)
	filename := vars["filename"]
	// log.Print(filename)

	rows, err := db.Query("SELECT count(*) FROM file WHERE username = $1 AND filename = $2", username, filename)
	if err != nil {
		LogAndSendError(err, 400, "Failed to query the database", w)
		return
	}
	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			LogAndSendError(err, 400, "Failed to get count from the query result", w)
		}
	}
	if count > 0 {
		LogAndSendError(err, 400, "This username + filename already exists", w)
		return
	}

	contentType := r.Header.Get("Content-Type")
	contentLength := r.Header.Get("Content-Length")

	_, err = db.Query("INSERT INTO file VALUES($1, $2, $3, $4)", username, filename, contentType, contentLength)
	if err != nil {
		LogAndSendError(err, 400, "Failed to insert the user's file record to database", w)
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
		LogAndSendError(nil, 403, "You are not logged in", w)
		return
	}
	// log.Print(username)

	vars := mux.Vars(r)
	filename := vars["filename"]
	// log.Print(filename)

	rows, err := db.Query("SELECT content_type, content_length FROM file WHERE username = $1 AND filename = $2", username, filename)
	if err != nil {
		LogAndSendError(err, 400, "Failed to query the database", w)
		return
	}

	var contentType string
	var contentLength string
	for rows.Next() {
		err := rows.Scan(&contentType, &contentLength)
		if err != nil {
			LogAndSendError(err, 400, "Failed to get filename from the query result", w)
		}
	}

	if len(contentType) == 0 {
		LogAndSendError(err, 404, "No such file was uploaded before", w)
		return
	}

	path := "files/" + username + "/" + filename
	streamBytes, err := ioutil.ReadFile(path)
	if err != nil {
		LogAndSendError(err, 404, "Failed to read file from the disk.", w)
		return
	}

	buffer := bytes.NewBuffer(streamBytes)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", contentLength)
	w.WriteHeader(200)

	if _, err := buffer.WriteTo(w); err != nil {
		panic(err)
	}
}

func DeleteFile(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromSession(r)
	if len(username) == 0 {
		LogAndSendError(nil, 400, "You are not logged in", w)
		return
	}
	// log.Print(username)

	vars := mux.Vars(r)
	filename := vars["filename"]
	// log.Print(filename)

	rows, err := db.Query("SELECT count(*) FROM file WHERE username = $1 AND filename = $2", username, filename)
	if err != nil {
		LogAndSendError(err, 400, "Failed to query the database", w)
		return
	}
	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			LogAndSendError(err, 400, "Failed to get count from the query result", w)
		}
	}
	if count == 0 {
		LogAndSendError(err, 404, "File does not exists", w)
		return
	}

	_, err = db.Query("DELETE FROM file WHERE username = $1 AND filename = $2", username, filename)
	if err != nil {
		LogAndSendError(err, 400, "Failed to query the database", w)
		return
	}

	path := "files/" + username + "/" + filename
	os.Remove(path)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(204)
}

func ListFiles(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromSession(r)
	if len(username) == 0 {
		LogAndSendError(nil, 403, "You are not logged in", w)
		return
	}
	// log.Print(username)

	rows, err := db.Query("SELECT filename FROM file WHERE username = $1", username)
	if err != nil {
		LogAndSendError(err, 400, "Failed to query the database", w)
		return
	}

	var files []string
	var filename string
	for rows.Next() {
		err := rows.Scan(&filename)
		if err != nil {
			LogAndSendError(err, 400, "Failed to get filename from the query result", w)
		}
		files = append(files, filename)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(200)
	if err := json.NewEncoder(w).Encode(files); err != nil {
		panic(err)
	}
}
