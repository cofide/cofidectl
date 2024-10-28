package config

import (
	"reflect"
	"testing"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

func TestFileLoaderImplementsLoader(t *testing.T) {
	loader := NewFileLoader("fake.yaml")
	var _ Loader = loader
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

	got, err := loader.Read()
	if err != nil {
		t.Fatalf("MemoryLoader.Read() returned error: %v", err)
	}

	want := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MemoryLoader.Read() = %v, want %v", got, want)
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
		{
			Name:          "tz1",
			TrustProvider: &trust_provider_proto.TrustProvider{},
			// FIXME: The zero value for these slices is nil, but the YAML unmarshaller creates empty slices
			Federations:         []*federation_proto.Federation{},
			AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		},
		{
			Name:                "tz2",
			TrustProvider:       &trust_provider_proto.TrustProvider{},
			Federations:         []*federation_proto.Federation{},
			AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		},
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		{Name: "ap1"},
		{Name: "ap2"},
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
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MemoryLoader.Read() = %v, want %v", got, want)
	}
}

func TestMemoryLoaderInitialConfig(t *testing.T) {
	// Creating a MemoryLoader with an initial Config should return an identical Config on Read.
	config := NewConfig()
	config.Plugins = []string{"plugin1", "plugin2"}
	config.TrustZones = []*trust_zone_proto.TrustZone{
		{
			Name:          "tz1",
			TrustProvider: &trust_provider_proto.TrustProvider{},
			// FIXME: The zero value for these slices is nil, but the YAML unmarshaller creates empty slices
			Federations:         []*federation_proto.Federation{},
			AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		},
		{
			Name:                "tz2",
			TrustProvider:       &trust_provider_proto.TrustProvider{},
			Federations:         []*federation_proto.Federation{},
			AttestationPolicies: []*attestation_policy_proto.AttestationPolicy{},
		},
	}
	config.AttestationPolicies = []*attestation_policy_proto.AttestationPolicy{
		{Name: "ap1"},
		{Name: "ap2"},
	}

	loader, err := NewMemoryLoader(config)
	if err != nil {
		t.Fatalf("NewMemoryLoader() returned error: %v", err)
	}

	got, err := loader.Read()
	if err != nil {
		t.Fatalf("MemoryLoader.Read() returned error: %v", err)
	}

	want := config
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MemoryLoader.Read() = %v, want %v", got, want)
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
