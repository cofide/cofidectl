package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
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
	if err != nil {
		t.Errorf("FileLoader.Exists() error = %v", err)
	} else if gotExists {
		t.Errorf("FileLoader.Exists() returned true")
	}

	_, gotErr := loader.Read()
	if gotErr == nil {
		t.Fatal("FileLoader.Read() did not return error")
	}

	wantErr := "non-existent.yaml: no such file or directory"
	if !strings.Contains(gotErr.Error(), wantErr) {
		t.Fatalf("FileLoader.Read() err = %v, want %v", gotErr.Error(), wantErr)
	}
}

func TestFileLoaderWriteEmptyConfig(t *testing.T) {
	// Reading after writing an empty Config should return an identical empty Config.
	tempDir := t.TempDir()
	loader := NewFileLoader(filepath.Join(tempDir, "config.yaml"))

	config := NewConfig()
	err := loader.Write(config)
	if err != nil {
		t.Fatalf("FileLoader.Write() returned error: %v", err)
	}

	gotExists, err := loader.Exists()
	if err != nil {
		t.Errorf("FileLoader.Exists() error = %v", err)
	} else if !gotExists {
		t.Errorf("FileLoader.Exists() returned false")
	}

	got, err := loader.Read()
	if err != nil {
		t.Fatalf("FileLoader.Read() returned error: %v", err)
	}

	want := NewConfig()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FileLoader.Read() mismatch (-want,+got):\n%s", diff)
	}
}

func TestFileLoaderNonEmptyConfig(t *testing.T) {
	// Reading after writing a non-empty Config should return an identical non-empty Config.
	tempDir := t.TempDir()
	loader := NewFileLoader(filepath.Join(tempDir, "config.yaml"))

	config := NewConfig()
	config.Plugins = []string{"plugin1", "plugin2"}
	config.TrustZones = []*trust_zone_proto.TrustZone{
		fixtures.TrustZone("tz1"),
		fixtures.TrustZone("tz2"),
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		fixtures.AttestationPolicy("ap1"),
		fixtures.AttestationPolicy("ap2"),
	}

	err := loader.Write(config)
	if err != nil {
		t.Fatalf("FileLoader.Write() returned error: %v", err)
	}

	got, err := loader.Read()
	if err != nil {
		t.Fatalf("FileLoader.Read() returned error: %v", err)
	}

	want := config
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("FileLoader.Read() mismatch (-want,+got):\n%s", diff)
	}
}

func TestFileLoaderReadInvalid(t *testing.T) {
	// Reading from data that does not pass validation returns an error
	tempFile := filepath.Join(t.TempDir(), "config.yaml")
	loader := NewFileLoader(tempFile)

	if err := os.WriteFile(tempFile, []byte(`plugins: "not-a-list"`), 0o600); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	_, gotErr := loader.Read()
	if gotErr == nil {
		t.Fatalf("FileLoader.Read() did not return error")
	}

	wantErr := `error validating configuration YAML: plugins: conflicting values "not-a-list" and [...#Plugin] (mismatched types string and list)`

	if gotErr.Error() != wantErr {
		t.Fatalf("FileLoader.Read() err = %v, want %v", gotErr.Error(), wantErr)
	}
}

func TestMemoryLoaderImplementsLoader(t *testing.T) {
	loader, _ := NewMemoryLoader(nil)
	var _ Loader = loader
}

func TestMemoryLoaderReadEmptyConfig(t *testing.T) {
	// First read without a write should return an error.
	loader, err := NewMemoryLoader(nil)
	if err != nil {
		t.Fatalf("NewMemoryLoader() returned error: %v", err)
	}

	gotExists, err := loader.Exists()
	if err != nil {
		t.Errorf("MemoryLoader.Exists() error = %v", err)
	} else if gotExists {
		t.Errorf("MemoryLoader.Exists() returned true")
	}

	_, gotErr := loader.Read()
	if gotErr == nil {
		t.Fatal("MemoryLoader.Read() did not return error")
	}

	wantErr := "in-memory configuration does not exist"
	if gotErr.Error() != wantErr {
		t.Fatalf("MemoryLoader.Read() err = %v, want %v", gotErr.Error(), wantErr)
	}
}

func TestMemoryLoaderWriteEmptyConfig(t *testing.T) {
	// Reading after writing an empty Config should return an identical empty Config.
	loader, err := NewMemoryLoader(nil)
	if err != nil {
		t.Fatalf("NewMemoryLoader() returned error: %v", err)
	}

	config := NewConfig()
	err = loader.Write(config)
	if err != nil {
		t.Fatalf("MemoryLoader.Write() returned error: %v", err)
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

	want := NewConfig()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("MemoryLoader.Read() mismatch (-want,+got):\n%s", diff)
	}
}

func TestMemoryLoaderNonEmptyConfig(t *testing.T) {
	// Reading after writing a non-empty Config should return an identical non-empty Config.
	loader, err := NewMemoryLoader(nil)
	if err != nil {
		t.Fatalf("NewMemoryLoader() returned error: %v", err)
	}

	config := NewConfig()
	config.Plugins = []string{"plugin1", "plugin2"}
	config.TrustZones = []*trust_zone_proto.TrustZone{
		fixtures.TrustZone("tz1"),
		fixtures.TrustZone("tz2"),
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		fixtures.AttestationPolicy("ap1"),
		fixtures.AttestationPolicy("ap2"),
	}

	err = loader.Write(config)
	if err != nil {
		t.Fatalf("MemoryLoader.Write() returned error: %v", err)
	}

	got, err := loader.Read()
	if err != nil {
		t.Fatalf("MemoryLoader.Read() returned error: %v", err)
	}

	want := config
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("MemoryLoader.Read() mismatch (-want,+got):\n%s", diff)
	}
}

func TestMemoryLoaderInitialConfig(t *testing.T) {
	// Creating a MemoryLoader with an initial Config should return an identical Config on Read.
	config := NewConfig()
	config.Plugins = []string{"plugin1", "plugin2"}
	config.TrustZones = []*trust_zone_proto.TrustZone{
		fixtures.TrustZone("tz1"),
		fixtures.TrustZone("tz2"),
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		fixtures.AttestationPolicy("ap1"),
		fixtures.AttestationPolicy("ap2"),
	}

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
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("MemoryLoader.Read() mismatch (-want,+got):\n%s", diff)
	}
}

func TestMemoryLoaderReadInvalid(t *testing.T) {
	// Reading from data that does not pass validation returns an error
	loader, err := NewMemoryLoader(nil)
	if err != nil {
		t.Fatalf("NewMemoryLoader() returned error: %v", err)
	}

	loader.data = []byte(`plugins: "not-a-list"`)
	loader.exists = true

	_, gotErr := loader.Read()
	if gotErr == nil {
		t.Fatalf("MemoryLoader.Read() did not return error")
	}

	wantErr := `error validating configuration YAML: plugins: conflicting values "not-a-list" and [...#Plugin] (mismatched types string and list)`

	if gotErr.Error() != wantErr {
		t.Fatalf("MemoryLoader.Read() err = %v, want %v", gotErr.Error(), wantErr)
	}
}
