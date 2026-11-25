package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "modernc.org/sqlite"
)

var db *sql.DB

var password = "admin123"

func main() {
	db, _ = sql.Open("sqlite", "data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT, body TEXT)")

	http.HandleFunc("/create", create)
	http.HandleFunc("/list", list)
	http.HandleFunc("/get", get)
	http.HandleFunc("/update", update)
	http.HandleFunc("/delete", delete)

	fmt.Println("server on :8080, admin password: " + password)
	http.ListenAndServe(":8080", nil)
}

func create(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title")
	body := r.URL.Query().Get("body")
	q := "INSERT INTO notes (title, body) VALUES (?, ?)"
	fmt.Println(q)
	_, err := db.Exec(q, title, body)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("ok"))
}

func list(w http.ResponseWriter, r *http.Request) {
	q := "SELECT id, title, body FROM notes"
	search := r.URL.Query().Get("search")
	if search != "" {
		q = q + " WHERE title LIKE '%" + search + "%'"
	}
	fmt.Println(q)
	rows, err := db.Query(q)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	for rows.Next() {
		var id int
		var title, body string
		rows.Scan(&id, &title, &body)
		w.Write([]byte(fmt.Sprint(id) + " | " + title + " | " + body + "\n"))
	}
}

func get(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	q := "SELECT id, title, body FROM notes WHERE id = " + id
	fmt.Println(q)
	rows, err := db.Query(q)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	for rows.Next() {
		var nid int
		var title, body string
		rows.Scan(&nid, &title, &body)
		w.Write([]byte("<h1>" + title + "</h1><p>" + body + "</p>"))
	}
}

func update(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	title := r.URL.Query().Get("title")
	body := r.URL.Query().Get("body")
	q := "UPDATE notes SET title = '" + title + "', body = '" + body + "' WHERE id = " + id
	fmt.Println(q)
	_, err := db.Exec(q)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("ok"))
}

func delete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	q := "DELETE FROM notes WHERE id = " + id
	fmt.Println(q)
	db.Exec(q)
	w.Write([]byte("deleted"))
}
