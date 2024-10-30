package config

import (
	"os"
	"path/filepath"
	"testing"
)

func readTestConfig(t *testing.T, file string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", "config", file))
	if err != nil {
		t.Fatal("Failed to read test configuration file")
	}
	return data
}
