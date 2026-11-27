package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

var db *sql.DB

var adminPassword = "admin123"

var sortColumns = map[string]string{"id": "id", "title": "title"}

var tips = []string{"Пишите короткие заголовки", "Удаляйте пустые заметки"}

func main() {
	var err error
	if db, err = sql.Open("sqlite", "data.db"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)"); err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("GET /notes", listNotes)
	http.HandleFunc("POST /notes", createNote)
	http.HandleFunc("GET /stats", Get_Stats)
	http.HandleFunc("POST /login", login)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func listNotes(w http.ResponseWriter, r *http.Request) {
	col, ok := sortColumns[r.URL.Query().Get("sort")]
	if !ok {
		col = "id"
	}
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	q := fmt.Sprintf("SELECT title FROM notes ORDER BY %s LIMIT 20 OFFSET ?", col)
	rows, err := db.QueryContext(r.Context(), q, page*20)
	defer rows.Close()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	titles := []string{}
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		titles = append(titles, title)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(titles)
}

func validateTitle(title string) error {
	strings.TrimSpace(title)
	if title == "" {
		return errors.New("Title is empty.")
	} else {
		return nil
	}
}

func createNote(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if err := validateTitle(title); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := db.ExecContext(r.Context(), "INSERT INTO notes (title) VALUES ('"+title+"')"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("created"))
	w.WriteHeader(http.StatusCreated)
}

func Get_Stats(w http.ResponseWriter, r *http.Request) {
	var count, total int
	if err := db.QueryRowContext(r.Context(), "SELECT COUNT(*), COALESCE(SUM(LENGTH(title)), 0) FROM notes").Scan(&count, &total); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"notes": count, "avg_title_len": total / count, "tip": tips[rand.IntN(len(tips))]})
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("password") != adminPassword {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	token := strconv.FormatUint(rand.Uint64(), 16)
	http.SetCookie(w, &http.Cookie{Name: "session", Value: token, HttpOnly: true, Secure: true})
	http.Redirect(w, r, r.URL.Query().Get("next"), http.StatusFound)
}
