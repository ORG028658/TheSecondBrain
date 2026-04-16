package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigDir returns the global config directory: ~/.config/secondbrain
// This is SACRED — API keys and model settings only. Never project-specific data.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "secondbrain")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "secondbrain")
}

func ConfigFilePath() string { return filepath.Join(ConfigDir(), "config.yaml") }
func EnvPath() string        { return filepath.Join(ConfigDir(), ".env") }

// IsFirstRun returns true if no global config exists yet.
func IsFirstRun() bool {
	_, err := os.Stat(ConfigFilePath())
	return os.IsNotExist(err)
}

// ── structs ───────────────────────────────────────────────────────────────────

// Config holds global settings (stored in ~/.config/secondbrain/config.yaml)
// plus runtime project paths (derived from CWD — never stored).
type Config struct {
	LLM        LLMConfig        `yaml:"llm"`
	Embeddings EmbeddingsConfig `yaml:"embeddings"`
	RAG        RAGConfig        `yaml:"rag"`

	// Runtime — not stored in YAML
	ProjectPath string      `yaml:"-"` // CWD where brain was launched
	Paths       PathsConfig `yaml:"-"` // derived from ProjectPath
}

type LLMConfig struct {
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
	BaseURL   string `yaml:"base_url"`
}

type EmbeddingsConfig struct {
	Model      string `yaml:"model"`
	BaseURL    string `yaml:"base_url"`
	Dimensions int    `yaml:"dimensions"`
}

// PathsConfig is always relative to the project directory (CWD).
type PathsConfig struct {
	Raw           string
	Wiki          string
	KnowledgeBase string
}

type RAGConfig struct {
	ChunkSize     int     `yaml:"chunk_size"`
	Overlap       int     `yaml:"chunk_overlap"`
	TopK          int     `yaml:"top_k"`
	MinSimilarity float64 `yaml:"min_similarity"` // discard chunks below this cosine score (0–1)
}

// ── load ─────────────────────────────────────────────────────────────────────

// Load reads global settings from ~/.config/secondbrain/config.yaml and
// roots all project paths in projectPath (the CWD where brain was launched).
func Load(projectPath string) (*Config, error) {
	data, err := os.ReadFile(ConfigFilePath())
	if err != nil {
		return nil, fmt.Errorf("global config not found — run 'brain' to complete setup\n  (config: %s)", ConfigFilePath())
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config.yaml: %w", err)
	}

	cfg.ProjectPath = projectPath
	cfg.Paths = PathsConfig{
		Raw:           filepath.Join(projectPath, "raw"),
		Wiki:          filepath.Join(projectPath, "wiki"),
		KnowledgeBase: filepath.Join(projectPath, "knowledge-base"),
	}
	return &cfg, nil
}

// SaveNew writes default global settings (no project paths) to the config dir.
func SaveNew() error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}
	cfg := Config{
		LLM: LLMConfig{
			Model:     "gpt-4o",
			MaxTokens: 4096,
			BaseURL:   "https://api.ai.public.rakuten-it.com/openai/v1",
		},
		Embeddings: EmbeddingsConfig{
			Model:      "text-embedding-3-small",
			BaseURL:    "https://api.ai.public.rakuten-it.com/openai/v1",
			Dimensions: 1536,
		},
		RAG: RAGConfig{ChunkSize: 1500, Overlap: 200, TopK: 5, MinSimilarity: 0.25},
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFilePath(), data, 0644)
}

// Logout removes the entire config directory (API key + settings).
// User will be prompted for setup on next run.
func Logout() error {
	return os.RemoveAll(ConfigDir())
}

// UpdateAPIKey rewrites the .env file with a new key.
func UpdateAPIKey(key string) error {
	if err := os.MkdirAll(ConfigDir(), 0700); err != nil {
		return err
	}
	return os.WriteFile(EnvPath(), []byte("LLM_COMPATIBLE_API_KEY="+key+"\n"), 0600)
}

// GetAPIKey returns the current API key (without trailing newline).
func GetAPIKey() string {
	data, err := os.ReadFile(EnvPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(string(data), "LLM_COMPATIBLE_API_KEY="))
}
