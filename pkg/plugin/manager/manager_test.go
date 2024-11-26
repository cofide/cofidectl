// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"errors"
	"testing"

	"github.com/cofide/cofidectl/internal/pkg/config"
	cofidectl_plugin "github.com/cofide/cofidectl/pkg/plugin"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGrpcDataSource struct {
	local.LocalDataSource
}

func newFakeGrpcDataSource(t *testing.T, configLoader config.Loader) *fakeGrpcDataSource {
	lds, err := local.NewLocalDataSource(configLoader)
	assert.Nil(t, err)
	return &fakeGrpcDataSource{LocalDataSource: *lds}
}

func TestManager_Init_success(t *testing.T) {
	tests := []struct {
		name       string
		config     *config.Config
		pluginName string
		want       func(config.Loader) cofidectl_plugin.DataSource
	}{
		{
			name:       "local",
			config:     nil,
			pluginName: LocalPluginName,
			want: func(cl config.Loader) cofidectl_plugin.DataSource {
				lds, err := local.NewLocalDataSource(cl)
				assert.Nil(t, err)
				return lds
			},
		},
		{
			name:       "gRPC",
			config:     nil,
			pluginName: "fake-plugin",
			want: func(cl config.Loader) cofidectl_plugin.DataSource {
				fcds := newFakeGrpcDataSource(t, cl)
				return fcds
			},
		},
		{
			name:       "existing local",
			config:     &config.Config{DataSource: LocalPluginName},
			pluginName: LocalPluginName,
			want: func(cl config.Loader) cofidectl_plugin.DataSource {
				lds, err := local.NewLocalDataSource(cl)
				assert.Nil(t, err)
				return lds
			},
		},
		{
			name:       "existing gRPC",
			config:     &config.Config{DataSource: "fake-plugin"},
			pluginName: "fake-plugin",
			want: func(cl config.Loader) cofidectl_plugin.DataSource {
				fcds := newFakeGrpcDataSource(t, cl)
				return fcds
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			// Mock out the Connect plugin loader function.
			m.loadGrpcPlugin = func(logger hclog.Logger, _ string) (*go_plugin.Client, cofidectl_plugin.DataSource, error) {
				return nil, newFakeGrpcDataSource(t, configLoader), nil
			}

			got, err := m.Init(tt.pluginName)
			require.Nil(t, err)

			want := tt.want(configLoader)
			assert.Equal(t, want, got)

			config, err := configLoader.Read()
			assert.Nil(t, err)
			assert.Equal(t, config.DataSource, tt.pluginName)

			got2, err := m.GetDataSource()
			require.Nil(t, err)
			assert.Same(t, got, got2, "GetDataSource() should return a cached copy")
		})
	}
}

func TestManager_Init_failure(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		pluginName     string
		want           func(config.Loader) cofidectl_plugin.DataSource
		wantErrMessage string
	}{
		{
			name:           "existing different plugin",
			config:         &config.Config{DataSource: "fake-plugin"},
			pluginName:     LocalPluginName,
			wantErrMessage: "existing config file uses a different plugin: fake-plugin vs local",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)

			_, err = m.Init(tt.pluginName)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErrMessage)

			config, err := configLoader.Read()
			assert.Nil(t, err)
			assert.Equal(t, config.DataSource, tt.config.DataSource, "config should not be updated")

			assert.Nil(t, m.source, "cached data source should be nil")
			assert.Nil(t, m.client, "cached client should be nil")
		})
	}
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
				assert.Nil(t, err)
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
			require.Nil(t, err)

			m := NewManager(configLoader)

			// Mock out the Connect plugin loader function.
			var client *go_plugin.Client
			if tt.config.DataSource != LocalPluginName {
				client = &go_plugin.Client{}
				m.loadGrpcPlugin = func(logger hclog.Logger, _ string) (*go_plugin.Client, cofidectl_plugin.DataSource, error) {
					ds := newFakeGrpcDataSource(t, configLoader)
					return client, ds, nil
				}
			}

			got, err := m.GetDataSource()
			require.Nil(t, err)

			want := tt.want(configLoader)
			assert.Equal(t, want, got)
			assert.Same(t, client, m.client)

			got2, err := m.GetDataSource()
			require.Nil(t, err)
			assert.Same(t, got, got2, "second GetDataSource() should return a cached copy")
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
			require.Nil(t, err)

			m := NewManager(configLoader)
			// Mock out the Connect plugin loader function, and inject a load failure.
			m.loadGrpcPlugin = func(logger hclog.Logger, _ string) (*go_plugin.Client, cofidectl_plugin.DataSource, error) {
				return nil, nil, errors.New("failed to create connect plugin")
			}

			_, err = m.GetDataSource()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, m.source, "failed GetDataSource should not cache")
			assert.Nil(t, m.client, "failed GetDataSource should not cache")
		})
	}
}

func TestManager_Shutdown(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   func(config.Loader) cofidectl_plugin.DataSource
	}{
		{
			name:   "local",
			config: config.Config{DataSource: LocalPluginName},
		},
		{
			name:   "gRPC",
			config: config.Config{DataSource: "fake-plugin"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(&tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			// Mock out the Connect plugin loader function.
			client := &go_plugin.Client{}
			m.loadGrpcPlugin = func(logger hclog.Logger, _ string) (*go_plugin.Client, cofidectl_plugin.DataSource, error) {
				ds := newFakeGrpcDataSource(t, configLoader)
				return client, ds, nil
			}

			_, err = m.GetDataSource()
			require.Nil(t, err)

			m.Shutdown()
			assert.Nil(t, m.source)
			assert.Nil(t, m.client)
		})
	}
}

func TestManager_GetPluginConfig(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   func(config.Loader) cofidectl_plugin.DataSource
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO
		})
	}
}

func TestManager_SetPluginConfig(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   func(config.Loader) cofidectl_plugin.DataSource
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO
		})
	}
}
