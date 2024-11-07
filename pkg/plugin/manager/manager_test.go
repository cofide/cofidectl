// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cofide/cofidectl/internal/pkg/config"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	hclog "github.com/hashicorp/go-hclog"
)

type fakeGrpcDataSource struct {
	local.LocalDataSource
}

func newFakeGrpcDataSource(t *testing.T, configLoader config.Loader) *fakeGrpcDataSource {
	lds, err := local.NewLocalDataSource(configLoader)
	if err != nil {
		t.Fatalf("NewLocalDataSource() error = %v", err)
	}
	return &fakeGrpcDataSource{LocalDataSource: *lds}
}

func TestManager_GetDataSource_success(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   func(config.Loader) cofidectl_plugin.DataSource
	}{
		{
			name:   "local",
			config: config.Config{DataSource: LocalPluginName},
			want: func(cl config.Loader) cofidectl_plugin.DataSource {
				lds, err := local.NewLocalDataSource(cl)
				if err != nil {
					t.Fatalf("NewLocalDataSource() error = %v", err)
				}
				return lds
			},
		},
		{
			name:   "gRPC",
			config: config.Config{DataSource: "fake-plugin"},
			want: func(cl config.Loader) cofidectl_plugin.DataSource {
				fcds := newFakeGrpcDataSource(t, cl)
				return fcds
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(&tt.config)
			if err != nil {
				t.Fatalf("NewMemoryLoader() error = %v", err)
			}

			l := NewManager(configLoader)
			// Mock out the Connect plugin loader function.
			l.loadGrpcPlugin = func(logger hclog.Logger, _ string) (cofidectl_plugin.DataSource, error) {
				return newFakeGrpcDataSource(t, configLoader), nil
			}

			got, err := l.GetDataSource()
			if err != nil {
				t.Fatalf("Manager.GetDataSource() error = %v", err)
			}

			want := tt.want(configLoader)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Manager.GetDataSource() = %v, want %v", got, want)
			}
		})
	}
}

func TestManager_GetDataSource_failure(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr string
	}{
		{
			name:    "empty",
			config:  config.Config{DataSource: ""},
			wantErr: "plugin name cannot be empty",
		},
		{
			name:    "connect plugin load failure",
			config:  config.Config{DataSource: "fake-plugin"},
			wantErr: "failed to create connect plugin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(&tt.config)
			if err != nil {
				t.Fatalf("NewMemoryLoader() error = %v", err)
			}

			l := NewManager(configLoader)
			// Mock out the Connect plugin loader function, and inject a load failure.
			l.loadGrpcPlugin = func(logger hclog.Logger, _ string) (cofidectl_plugin.DataSource, error) {
				return nil, errors.New("failed to create connect plugin")
			}

			_, err = l.GetDataSource()
			if err == nil {
				t.Fatalf("Manager.GetDataSource() did not return error")
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("Manager.GetDataSource() error message = %s, wantErrString %s", err.Error(), tt.wantErr)
			}
		})
	}
}
