package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type Store struct {
	mu   sync.RWMutex
	path string
	data storeData
}

type storeData struct {
	Entries    []Entry           `json:"entries"`
	PageHashes map[string]string `json:"page_hashes"`
}

type Entry struct {
	ID       string    `json:"id"`
	WikiPath string    `json:"wiki_path"`
	ChunkIdx int       `json:"chunk_idx"`
	Text     string    `json:"text"`
	Vector   []float32 `json:"vector"`
}

type SearchResult struct {
	WikiPath string
	Text     string
	Score    float32
}

func New(storePath string) (*Store, error) {
	s := &Store{
		path: storePath,
		data: storeData{
			Entries:    []Entry{},
			PageHashes: map[string]string{},
		},
	}

	if _, err := os.Stat(storePath); err == nil {
		data, err := os.ReadFile(storePath)
		if err != nil {
			return nil, fmt.Errorf("reading store: %w", err)
		}
		if err := json.Unmarshal(data, &s.data); err != nil {
			return nil, fmt.Errorf("parsing store: %w", err)
		}
	}

	return s, nil
}

func (s *Store) PageHash(wikiPath string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.PageHashes[wikiPath]
}

// Upsert replaces all chunks for a wiki page with new ones.
func (s *Store) Upsert(wikiPath, contentHash string, chunks []string, vectors [][]float32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old entries for this page
	filtered := s.data.Entries[:0]
	for _, e := range s.data.Entries {
		if e.WikiPath != wikiPath {
			filtered = append(filtered, e)
		}
	}
	s.data.Entries = filtered

	// Add new chunks
	for i, text := range chunks {
		raw := fmt.Sprintf("%s#%d", wikiPath, i)
		id := fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))[:12]
		s.data.Entries = append(s.data.Entries, Entry{
			ID:       id,
			WikiPath: wikiPath,
			ChunkIdx: i,
			Text:     text,
			Vector:   vectors[i],
		})
	}

	s.data.PageHashes[wikiPath] = contentHash
}

func (s *Store) DeletePage(wikiPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := s.data.Entries[:0]
	for _, e := range s.data.Entries {
		if e.WikiPath != wikiPath {
			filtered = append(filtered, e)
		}
	}
	s.data.Entries = filtered
	delete(s.data.PageHashes, wikiPath)
}

func (s *Store) Search(vector []float32, topK int) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		entry Entry
		score float32
	}

	results := make([]scored, 0, len(s.data.Entries))
	for _, e := range s.data.Entries {
		results = append(results, scored{e, cosine(vector, e.Vector)})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if topK > len(results) {
		topK = len(results)
	}

	out := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		out[i] = SearchResult{
			WikiPath: results[i].entry.WikiPath,
			Text:     results[i].entry.Text,
			Score:    results[i].score,
		}
	}
	return out
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) Stats() (pages, chunks int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.PageHashes), len(s.data.Entries)
}

func cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
