package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Loader provides an interface to read and write a `Config`.
type Loader interface {
	Exists() (bool, error)
	Read() (*Config, error)
	Write(*Config) error
}

// FileLoader implements the `Loader` interface by reading and writing to a file.
type FileLoader struct {
	filePath  string
	validator *Validator
}

func NewFileLoader(filePath string) *FileLoader {
	return &FileLoader{filePath: filePath, validator: NewValidator()}
}

func (fl *FileLoader) Exists() (bool, error) {
	if _, err := os.Stat(fl.filePath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (fl *FileLoader) Read() (*Config, error) {
	data, err := os.ReadFile(fl.filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	return validatedRead(data, fl.validator)
}

func (fl *FileLoader) Write(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling configuration to YAML: %w", err)
	}

	err = os.WriteFile(fl.filePath, data, 0600)
	if err != nil {
		return fmt.Errorf("error writing configuration file: %w", err)
	}

	return nil
}

// MemoryLoader implements the `Loader` interface by reading and writing to bytes in memory.
type MemoryLoader struct {
	exists    bool
	data      []byte
	validator *Validator
}

func NewMemoryLoader(config *Config) (*MemoryLoader, error) {
	ml := &MemoryLoader{
		exists:    config != nil,
		data:      []byte{},
		validator: NewValidator(),
	}

	if config != nil {
		if err := ml.Write(config); err != nil {
			return nil, err
		}
	}

	return ml, nil
}

func (ml *MemoryLoader) Exists() (bool, error) {
	return ml.exists, nil
}

func (ml *MemoryLoader) Read() (*Config, error) {
	if !ml.exists {
		return nil, fmt.Errorf("in-memory configuration does not exist")
	}

	return validatedRead(ml.data, ml.validator)
}

func (ml *MemoryLoader) Write(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling configuration to YAML: %w", err)
	}
	ml.data = data
	ml.exists = true
	return nil
}

func validatedRead(data []byte, validator *Validator) (*Config, error) {
	if err := validator.Validate(data); err != nil {
		return nil, err
	}

	config := NewConfig()
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling configuration from YAML: %s", err)
	}
	return config, nil
}
