// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
)

func TestFileLoaderImplementsLoader(t *testing.T) {
	loader := NewFileLoader("fake.yaml")
	var _ Loader = loader
}

func TestFileLoaderNonExistentConfig(t *testing.T) {
	// Reading a non-existent file should return an error.
	tempDir := t.TempDir()
	loader := NewFileLoader(filepath.Join(tempDir, "non-existent.yaml"))

	gotExists, err := loader.Exists()
	require.NoError(t, err, err)
	assert.False(t, gotExists, "FileLoader.Exists() returned true")

	_, gotErr := loader.Read()
	require.Error(t, gotErr, "FileLoader.Read() did not return error")

	wantErr := "non-existent.yaml: no such file or directory"
	assert.ErrorContains(t, gotErr, wantErr)
}

func TestFileLoaderWriteEmptyConfig(t *testing.T) {
	// Reading after writing an empty Config should return an identical empty Config.
	tempDir := t.TempDir()
	loader := NewFileLoader(filepath.Join(tempDir, "config.yaml"))

	config := NewConfig()
	err := loader.Write(config)
	require.NoError(t, err, err)

	gotExists, err := loader.Exists()
	require.NoError(t, err, err)
	assert.True(t, gotExists, "FileLoader.Exists() returned false")

	got, err := loader.Read()
	require.NoError(t, err, err)

	want := NewConfig()
	assert.EqualExportedValues(t, want, got)
}

func TestFileLoaderNonEmptyConfig(t *testing.T) {
	// Reading after writing a non-empty Config should return an identical non-empty Config.
	tempDir := t.TempDir()
	loader := NewFileLoader(filepath.Join(tempDir, "config.yaml"))

	config := NewConfig()
	config.TrustZones = []*trust_zone_proto.TrustZone{
		fixtures.TrustZone("tz1"),
		fixtures.TrustZone("tz2"),
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		fixtures.AttestationPolicy("ap1"),
		fixtures.AttestationPolicy("ap2"),
	}
	config.Plugins = fixtures.Plugins("plugins1")

	err := loader.Write(config)
	require.NoError(t, err, err)

	got, err := loader.Read()
	require.NoError(t, err, err)

	want := config
	assert.EqualExportedValues(t, want, got)
}

func TestFileLoaderReadInvalid(t *testing.T) {
	// Reading from data that does not pass validation returns an error
	tempFile := filepath.Join(t.TempDir(), "config.yaml")
	loader := NewFileLoader(tempFile)

	err := os.WriteFile(tempFile, []byte(`plugins: 123`), 0o600)
	require.NoError(t, err, err)

	_, gotErr := loader.Read()
	require.Error(t, gotErr, "FileLoader.Read() did not return error")

	wantErr := `error validating configuration YAML: plugins: conflicting values 123 and`
	assert.ErrorContains(t, gotErr, wantErr)
}

func TestMemoryLoaderImplementsLoader(t *testing.T) {
	loader, _ := NewMemoryLoader(nil)
	var _ Loader = loader
}

func TestMemoryLoaderReadEmptyConfig(t *testing.T) {
	// First read without a write should return an error.
	loader, err := NewMemoryLoader(nil)
	require.NoError(t, err, err)

	gotExists, err := loader.Exists()
	require.NoError(t, err, err)
	assert.False(t, gotExists, "MemoryLoader.Exists() returned true")

	_, gotErr := loader.Read()
	require.Error(t, gotErr, gotErr)

	wantErr := "in-memory configuration does not exist"
	assert.ErrorContains(t, gotErr, wantErr)
}

func TestMemoryLoaderWriteEmptyConfig(t *testing.T) {
	// Reading after writing an empty Config should return an identical empty Config.
	loader, err := NewMemoryLoader(nil)
	require.NoError(t, err, err)

	config := NewConfig()
	err = loader.Write(config)
	require.NoError(t, err, err)

	gotExists, err := loader.Exists()
	require.NoError(t, err, err)
	assert.True(t, gotExists, "MemoryLoader.Exists() returned false")

	got, err := loader.Read()
	require.NoError(t, err, err)

	want := NewConfig()
	assert.EqualExportedValues(t, want, got)
}

func TestMemoryLoaderNonEmptyConfig(t *testing.T) {
	// Reading after writing a non-empty Config should return an identical non-empty Config.
	loader, err := NewMemoryLoader(nil)
	require.NoError(t, err, err)

	config := NewConfig()
	config.TrustZones = []*trust_zone_proto.TrustZone{
		fixtures.TrustZone("tz1"),
		fixtures.TrustZone("tz2"),
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		fixtures.AttestationPolicy("ap1"),
		fixtures.AttestationPolicy("ap2"),
	}
	config.Plugins = fixtures.Plugins("plugins1")

	err = loader.Write(config)
	require.NoError(t, err, err)

	got, err := loader.Read()
	require.NoError(t, err, err)

	want := config
	assert.EqualExportedValues(t, want, got)
}

func TestMemoryLoaderInitialConfig(t *testing.T) {
	// Creating a MemoryLoader with an initial Config should return an identical Config on Read.
	config := NewConfig()
	config.TrustZones = []*trust_zone_proto.TrustZone{
		fixtures.TrustZone("tz1"),
		fixtures.TrustZone("tz2"),
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		fixtures.AttestationPolicy("ap1"),
		fixtures.AttestationPolicy("ap2"),
	}
	config.Plugins = fixtures.Plugins("plugins1")

	loader, err := NewMemoryLoader(config)
	if err != nil {
		t.Fatalf("NewMemoryLoader() returned error: %v", err)
	}

	gotExists, err := loader.Exists()
	if err != nil {
		t.Errorf("MemoryLoader.Exists() error = %v", err)
	} else if !gotExists {
		t.Errorf("MemoryLoader.Exists() returned false")
	}

	got, err := loader.Read()
	if err != nil {
		t.Fatalf("MemoryLoader.Read() returned error: %v", err)
	}

	want := config
	assert.EqualExportedValues(t, want, got)
}

func TestMemoryLoaderReadInvalid(t *testing.T) {
	// Reading from data that does not pass validation returns an error
	loader, err := NewMemoryLoader(nil)
	if err != nil {
		t.Fatalf("NewMemoryLoader() returned error: %v", err)
	}

	loader.data = []byte(`plugins: 123`)
	loader.exists = true

	_, gotErr := loader.Read()
	if gotErr == nil {
		t.Fatalf("MemoryLoader.Read() did not return error")
	}

	wantErr := `error validating configuration YAML: plugins: conflicting values 123 and`
	assert.ErrorContains(t, gotErr, wantErr)
}
