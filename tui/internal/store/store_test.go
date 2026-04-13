package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ── NewFresh ──────────────────────────────────────────────────────────────────

func TestNewFresh_NotNil(t *testing.T) {
	s := NewFresh("/some/path/store.json")
	if s == nil {
		t.Fatal("NewFresh returned nil")
	}
}

func TestNewFresh_EmptyStore(t *testing.T) {
	s := NewFresh("/some/path/store.json")
	pages, chunks := s.Stats()
	if pages != 0 || chunks != 0 {
		t.Errorf("NewFresh should be empty, got pages=%d chunks=%d", pages, chunks)
	}
}

func TestNewFresh_OperatesWithoutPanic(t *testing.T) {
	s := NewFresh("/some/path/store.json")
	// All read operations on a fresh store should work without panic
	_ = s.PageHash("wiki/sources/test.md")
	results := s.Search([]float32{0.1, 0.2, 0.3}, 5)
	if len(results) != 0 {
		t.Error("Search on empty store should return no results")
	}
}

// ── New with corrupted store ──────────────────────────────────────────────────

func TestNew_CorruptedStore_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.json")

	// Write invalid JSON
	if err := os.WriteFile(storePath, []byte("{not valid json!!}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(storePath)
	if err == nil {
		t.Error("New should return error for corrupted store file")
	}
}

func TestNew_MissingFile_ReturnsEmptyStore(t *testing.T) {
	s, err := New("/nonexistent/path/store.json")
	if err != nil {
		t.Fatalf("New with missing file should succeed with empty store, got: %v", err)
	}
	pages, chunks := s.Stats()
	if pages != 0 || chunks != 0 {
		t.Errorf("expected empty store, got pages=%d chunks=%d", pages, chunks)
	}
}

func TestNew_ValidStore_LoadsData(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.json")

	// Write a valid minimal store
	data := storeData{
		Entries:    []Entry{},
		PageHashes: map[string]string{"wiki/sources/test.md": "abc123"},
	}
	b, _ := json.Marshal(data)
	os.WriteFile(storePath, b, 0644)

	s, err := New(storePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.PageHash("wiki/sources/test.md") != "abc123" {
		t.Error("loaded store should contain persisted page hash")
	}
}

// ── Upsert + Search ───────────────────────────────────────────────────────────

func TestUpsertAndSearch(t *testing.T) {
	s := NewFresh("")
	vec := []float32{1.0, 0.0, 0.0}
	s.Upsert("wiki/concepts/test.md", "hash1", []string{"chunk text"}, [][]float32{vec})

	results := s.Search(vec, 1)
	if len(results) == 0 {
		t.Fatal("expected search result, got none")
	}
	if results[0].WikiPath != "wiki/concepts/test.md" {
		t.Errorf("wrong wiki path: %s", results[0].WikiPath)
	}
	if results[0].Score < 0.99 {
		t.Errorf("identical vectors should score ~1.0, got %.3f", results[0].Score)
	}
}

func TestDeletePage(t *testing.T) {
	s := NewFresh("")
	vec := []float32{1.0, 0.0, 0.0}
	s.Upsert("wiki/concepts/test.md", "hash1", []string{"chunk"}, [][]float32{vec})
	s.DeletePage("wiki/concepts/test.md")

	results := s.Search(vec, 5)
	for _, r := range results {
		if r.WikiPath == "wiki/concepts/test.md" {
			t.Error("deleted page should not appear in search results")
		}
	}
}
