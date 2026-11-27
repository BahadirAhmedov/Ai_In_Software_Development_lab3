package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupDB(t *testing.T) {
	t.Helper()
	var err error
	if db, err = sql.Open("sqlite", ":memory:"); err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
}

func TestListNotesIgnoresUnknownSortColumn(t *testing.T) {
	setupDB(t)
	rec := httptest.NewRecorder()
	listNotes(rec, httptest.NewRequest(http.MethodGet, "/notes?sort=id;DROP%20TABLE%20notes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if _, err := db.Exec("SELECT 1 FROM notes"); err != nil {
		t.Fatalf("table dropped: %v", err)
	}
}

func TestDeleteNoteRequiresToken(t *testing.T) {
	setupDB(t)
	for _, tc := range []struct{ env, header string }{{"", ""}, {"secret", "wrong"}} {
		t.Setenv("ADMIN_TOKEN", tc.env)
		req := httptest.NewRequest(http.MethodDelete, "/notes/1", strings.NewReader(""))
		req.Header.Set("X-Admin-Token", tc.header)
		rec := httptest.NewRecorder()
		deleteNote(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("env=%q header=%q: status = %d, want 401", tc.env, tc.header, rec.Code)
		}
	}
}
