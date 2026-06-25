package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DataDir        string
	DBPath         string
	VaultPath      string
	VeclitePath    string
	EmbeddingModel string
}

func Load() (*Config, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    c := &Config{}
	c.DataDir = filepath.Join(homeDir, ".local", "share", "noted")
	c.DBPath = filepath.Join(c.DataDir, "noted.db")

	// Markdown vault directory (source of truth for notes). Override with NOTED_VAULT.
	if vaultPath := os.Getenv("NOTED_VAULT"); vaultPath != "" {
		c.VaultPath = vaultPath
	} else {
		c.VaultPath = filepath.Join(c.DataDir, "vault")
	}

	// Optional: veclite path from environment
	if veclitePath := os.Getenv("NOTED_VECLITE_PATH"); veclitePath != "" {
		c.VeclitePath = veclitePath
	} else {
		// Default veclite path in data directory
		c.VeclitePath = filepath.Join(c.DataDir, "vectors.veclite")
	}

	// Optional: embedding model from environment
	c.EmbeddingModel = os.Getenv("NOTED_EMBEDDING_MODEL")

	if err := os.MkdirAll(c.DataDir, os.ModePerm); err != nil {
		return nil, err
	}

	return c, nil
}
