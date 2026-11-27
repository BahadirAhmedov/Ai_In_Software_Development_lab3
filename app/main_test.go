package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListNotesIgnoresUnknownSortColumn(t *testing.T) {
	var err error
	if db, err = sql.Open("sqlite", ":memory:"); err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	listNotes(rec, httptest.NewRequest(http.MethodGet, "/notes?sort=id;DROP%20TABLE%20notes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if _, err := db.Exec("SELECT 1 FROM notes"); err != nil {
		t.Fatalf("table dropped: %v", err)
	}
}
