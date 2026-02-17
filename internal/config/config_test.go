package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_DefaultPaths(t *testing.T) {
	// Unset env vars that might interfere
	os.Unsetenv("NOTED_VECLITE_PATH")
	os.Unsetenv("NOTED_EMBEDDING_MODEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expectedDataDir := filepath.Join(homeDir, ".local", "share", "noted")

	if cfg.DataDir != expectedDataDir {
		t.Errorf("expected DataDir=%q, got %q", expectedDataDir, cfg.DataDir)
	}

	expectedDBPath := filepath.Join(expectedDataDir, "noted.db")
	if cfg.DBPath != expectedDBPath {
		t.Errorf("expected DBPath=%q, got %q", expectedDBPath, cfg.DBPath)
	}

	if !strings.HasSuffix(cfg.VeclitePath, "vectors.veclite") {
		t.Errorf("expected VeclitePath to end with vectors.veclite, got %q", cfg.VeclitePath)
	}
}

func TestLoad_CustomVeclitePath(t *testing.T) {
	customPath := "/tmp/test-vectors.veclite"
	t.Setenv("NOTED_VECLITE_PATH", customPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.VeclitePath != customPath {
		t.Errorf("expected VeclitePath=%q, got %q", customPath, cfg.VeclitePath)
	}
}

func TestLoad_CustomEmbeddingModel(t *testing.T) {
	t.Setenv("NOTED_EMBEDDING_MODEL", "all-minilm")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.EmbeddingModel != "all-minilm" {
		t.Errorf("expected EmbeddingModel=all-minilm, got %q", cfg.EmbeddingModel)
	}
}

func TestLoad_CreatesDataDir(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	info, err := os.Stat(cfg.DataDir)
	if os.IsNotExist(err) {
		t.Error("expected DataDir to be created")
	}
	if err == nil && !info.IsDir() {
		t.Error("expected DataDir to be a directory")
	}
}
