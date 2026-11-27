package main

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
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
	db, err = sql.Open("sqlite", "data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)"); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("GET /notes", listNotes)
	http.HandleFunc("POST /notes", createNote)
	http.HandleFunc("DELETE /notes/{id}", deleteNote)
	http.HandleFunc("GET /stats", Get_Stats)
	http.HandleFunc("POST /login", login)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
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
	q := fmt.Sprintf("SELECT id, title FROM notes ORDER BY %s LIMIT 20 OFFSET ?", col)
	rows, err := db.QueryContext(r.Context(), q, page*20)
	defer rows.Close()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	notes := map[int]string{}
	for rows.Next() {
		var id int
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		notes[id] = title
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, notes)
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

func deleteNote(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("ADMIN_TOKEN")
	if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Admin-Token")), []byte(token)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := db.ExecContext(r.Context(), "DELETE FROM notes WHERE id = ?", r.PathValue("id")); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func Get_Stats(w http.ResponseWriter, r *http.Request) {
	var count, total int
	err := db.QueryRowContext(r.Context(), "SELECT COUNT(*), COALESCE(SUM(LENGTH(title)), 0) FROM notes").Scan(&count, &total)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"notes": count, "avg_title_len": total / count, "tip": tips[rand.IntN(len(tips))]})
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
