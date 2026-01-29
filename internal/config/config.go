package config

import (
	"os"
	"path/filepath"
)

type Config struct {
  DataDir string
	DBPath string
}

func Load() (*Config, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    c := &Config{}
		c.DataDir = filepath.Join(homeDir, ".local", "share", "noted")
		c.DBPath = filepath.Join(c.DataDir, "noted.db")
    if err := os.MkdirAll(c.DataDir, os.ModePerm); err != nil {
        return nil, err
    }

    return c, nil
}
