// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"errors"
	"testing"

	pluginspb "github.com/cofide/cofide-api-sdk/gen/go/proto/plugins/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/cofide/cofidectl/pkg/plugin/provision"
	"github.com/cofide/cofidectl/pkg/plugin/provision/spirehelm"
	hclog "github.com/hashicorp/go-hclog"
	go_plugin "github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeGrpcDataSource struct {
	local.LocalDataSource
}

func newFakeGrpcDataSource(t *testing.T, configLoader config.Loader) *fakeGrpcDataSource {
	lds, err := local.NewLocalDataSource(configLoader)
	assert.Nil(t, err)
	return &fakeGrpcDataSource{LocalDataSource: *lds}
}

type fakeGrpcProvision struct {
	spirehelm.SpireHelm
}

func newFakeGrpcProvision() *fakeGrpcProvision {
	return &fakeGrpcProvision{}
}

func TestManager_Init_success(t *testing.T) {
	tests := []struct {
		name         string
		config       *config.Config
		plugins      *pluginspb.Plugins
		pluginConfig map[string]*structpb.Struct
	}{
		{
			name:    "defaults",
			config:  nil,
			plugins: GetDefaultPlugins(),
		},
		{
			name:    "nil plugins",
			config:  nil,
			plugins: nil,
		},
		{
			name:    "gRPC",
			config:  nil,
			plugins: fixtures.Plugins("plugins1"),
		},
		{
			name:         "defaults with config",
			config:       nil,
			plugins:      GetDefaultPlugins(),
			pluginConfig: fakePluginConfig(t),
		},
		{
			name:    "existing defaults",
			config:  &config.Config{Plugins: GetDefaultPlugins()},
			plugins: GetDefaultPlugins(),
		},
		{
			name:    "existing gRPC",
			config:  &config.Config{Plugins: fixtures.Plugins("plugins1")},
			plugins: fixtures.Plugins("plugins1"),
		},
		{
			name:         "existing defaults with config",
			config:       &config.Config{Plugins: GetDefaultPlugins(), PluginConfig: fakePluginConfig(t)},
			plugins:      GetDefaultPlugins(),
			pluginConfig: fakePluginConfig(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			// Mock out the gRPC plugin loader function.
			m.grpcPluginLoader = func(_ context.Context, _ hclog.Logger, _ string, _ *pluginspb.Plugins) (*grpcPlugin, error) {
				return &grpcPlugin{nil, newFakeGrpcDataSource(t, configLoader), newFakeGrpcProvision()}, nil
			}

			err = m.Init(context.Background(), tt.plugins, tt.pluginConfig)
			require.Nil(t, err)

			config, err := configLoader.Read()
			assert.Nil(t, err)
			wantPlugins := tt.plugins
			if wantPlugins == nil {
				wantPlugins = GetDefaultPlugins()
			}
			assert.Equal(t, wantPlugins.GetDataSource(), config.Plugins.GetDataSource())
			assert.Equal(t, wantPlugins.GetProvision(), config.Plugins.GetProvision())

			expectedConfig := tt.pluginConfig
			if expectedConfig == nil {
				expectedConfig = map[string]*structpb.Struct{}
			}
			assert.EqualExportedValues(t, expectedConfig, config.PluginConfig)
			for pluginName, value := range tt.pluginConfig {
				assert.NotSame(t, value, config.PluginConfig[pluginName], "pointer to plugin config stored in config")
			}

			assert.Nil(t, m.source, "cached data source should be nil")
			assert.Nil(t, m.provision, "cached provision plugin should be nil")
			assert.Empty(t, m.clients, "cached clients should be empty")
		})
	}
}

func TestManager_Init_failure(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		dsName         string
		provisionName  string
		pluginConfig   map[string]*structpb.Struct
		want           func(config.Loader) datasource.DataSource
		wantErrMessage string
	}{
		{
			name:           "existing different data source",
			config:         &config.Config{Plugins: fixtures.Plugins("plugins1")},
			dsName:         LocalDSPluginName,
			provisionName:  SpireHelmProvisionPluginName,
			wantErrMessage: "existing config file uses a different data source plugin: fake-datasource vs local",
		},
		{
			name:           "existing different provision plugin",
			config:         &config.Config{Plugins: fixtures.Plugins("plugins1")},
			dsName:         "fake-datasource",
			provisionName:  SpireHelmProvisionPluginName,
			wantErrMessage: "existing config file uses a different provision plugin: fake-provision vs spire-helm",
		},
		{
			name:           "existing different plugin config",
			config:         &config.Config{Plugins: GetDefaultPlugins(), PluginConfig: fakePluginConfig(t)},
			dsName:         LocalDSPluginName,
			provisionName:  SpireHelmProvisionPluginName,
			wantErrMessage: "existing config file has different plugin config:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)

			plugins := &pluginspb.Plugins{DataSource: &tt.dsName, Provision: &tt.provisionName}
			err = m.Init(context.Background(), plugins, tt.pluginConfig)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErrMessage)

			config, err := configLoader.Read()
			assert.Nil(t, err)
			assert.Equal(t, config.Plugins.GetDataSource(), tt.config.Plugins.GetDataSource(), "config should not be updated")
			assert.Equal(t, config.Plugins.GetProvision(), tt.config.Plugins.GetProvision(), "config should not be updated")

			assert.Nil(t, m.source, "cached data source should be nil")
			assert.Nil(t, m.provision, "cached provision plugin should be nil")
			assert.Empty(t, m.clients, "cached clients should be empty")
		})
	}
}

func TestManager_GetDataSource_success(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   func(config.Loader) datasource.DataSource
	}{
		{
			name:   "defaults",
			config: config.Config{Plugins: GetDefaultPlugins()},
			want: func(cl config.Loader) datasource.DataSource {
				lds, err := local.NewLocalDataSource(cl)
				assert.Nil(t, err)
				return lds
			},
		},
		{
			name:   "gRPC",
			config: config.Config{Plugins: fixtures.Plugins("plugins1")},
			want: func(cl config.Loader) datasource.DataSource {
				fcds := newFakeGrpcDataSource(t, cl)
				return fcds
			},
		},
		{
			name:   "gRPC with provision",
			config: config.Config{Plugins: fixtures.Plugins("plugins2")},
			want: func(cl config.Loader) datasource.DataSource {
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

			// Mock out the gRPC plugin loader function.
			var client *go_plugin.Client
			var provision provision.Provision
			if tt.config.Plugins.GetDataSource() != LocalDSPluginName {
				client = &go_plugin.Client{}
				m.grpcPluginLoader = func(_ context.Context, _ hclog.Logger, _ string, _ *pluginspb.Plugins) (*grpcPlugin, error) {
					ds := newFakeGrpcDataSource(t, configLoader)
					if tt.config.Plugins.GetProvision() == tt.config.Plugins.GetDataSource() {
						provision = newFakeGrpcProvision()
					}
					return &grpcPlugin{client, ds, provision}, nil
				}
			}

			got, err := m.GetDataSource(context.Background())
			require.Nil(t, err)

			want := tt.want(configLoader)
			assert.Equal(t, want, got)
			assert.Equal(t, want, m.source)
			assert.Same(t, client, m.clients[tt.config.Plugins.GetDataSource()])

			got2, err := m.GetDataSource(context.Background())
			require.Nil(t, err)
			assert.Same(t, got, got2, "second GetDataSource() should return a cached copy")

			if tt.config.Plugins.GetProvision() == tt.config.Plugins.GetDataSource() {
				got, err := m.GetProvision(context.Background())
				require.Nil(t, err, err)
				assert.Equal(t, provision, got)
				assert.Equal(t, provision, m.provision)
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
			config:  config.Config{Plugins: &pluginspb.Plugins{DataSource: fixtures.StringPtr("")}},
			wantErr: "plugin name cannot be empty",
		},
		{
			name:    "plugin load failure",
			config:  config.Config{Plugins: fixtures.Plugins("plugins1")},
			wantErr: "failed to create plugin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(&tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			// Mock out the gRPC plugin loader function, and inject a load failure.
			m.grpcPluginLoader = func(_ context.Context, _ hclog.Logger, _ string, _ *pluginspb.Plugins) (*grpcPlugin, error) {
				return nil, errors.New("failed to create plugin")
			}

			_, err = m.GetDataSource(context.Background())
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, m.source, "failed GetDataSource should not cache data source")
			assert.Nil(t, m.provision, "failed GetDataSource should not cache provision")
			assert.Empty(t, m.clients, "failed GetDataSource should not cache clients")
		})
	}
}

func TestManager_GetProvision_success(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   provision.Provision
	}{
		{
			name:   "defaults",
			config: config.Config{Plugins: GetDefaultPlugins()},
			want:   spirehelm.NewSpireHelm(nil),
		},
		{
			name:   "gRPC",
			config: config.Config{Plugins: fixtures.Plugins("plugins1")},
			want:   newFakeGrpcProvision(),
		},
		{
			name:   "gRPC with provision",
			config: config.Config{Plugins: fixtures.Plugins("plugins2")},
			want:   newFakeGrpcProvision(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(&tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)

			// Mock out the gRPC plugin loader function.
			var client *go_plugin.Client
			var source datasource.DataSource
			if tt.config.Plugins.GetDataSource() != LocalDSPluginName {
				client = &go_plugin.Client{}
				m.grpcPluginLoader = func(_ context.Context, _ hclog.Logger, _ string, _ *pluginspb.Plugins) (*grpcPlugin, error) {
					provision := newFakeGrpcProvision()
					if tt.config.Plugins.GetDataSource() == tt.config.Plugins.GetProvision() {
						source = newFakeGrpcDataSource(t, configLoader)
					}
					return &grpcPlugin{client, source, provision}, nil
				}
			}

			got, err := m.GetProvision(context.Background())
			require.Nil(t, err)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want, m.provision)
			assert.Same(t, client, m.clients[tt.config.Plugins.GetProvision()])

			got2, err := m.GetProvision(context.Background())
			require.Nil(t, err)
			assert.Same(t, got, got2, "second GetProvision() should return a cached copy")

			if tt.config.Plugins.GetDataSource() == tt.config.Plugins.GetProvision() {
				got, err := m.GetDataSource(context.Background())
				require.Nil(t, err, err)
				assert.Equal(t, source, got)
				assert.Equal(t, source, m.source)
			}
		})
	}
}

func TestManager_GetProvision_failure(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr string
	}{
		{
			name:    "empty",
			config:  config.Config{Plugins: &pluginspb.Plugins{DataSource: fixtures.StringPtr("")}},
			wantErr: "plugin name cannot be empty",
		},
		{
			name:    "plugin load failure",
			config:  config.Config{Plugins: fixtures.Plugins("plugins1")},
			wantErr: "failed to create plugin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(&tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			// Mock out the gRPC plugin loader function, and inject a load failure.
			m.grpcPluginLoader = func(_ context.Context, _ hclog.Logger, _ string, _ *pluginspb.Plugins) (*grpcPlugin, error) {
				return nil, errors.New("failed to create plugin")
			}

			_, err = m.GetProvision(context.Background())
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, m.source, "failed GetProvision should not cache data source")
			assert.Nil(t, m.provision, "failed GetProvision should not cache provision")
			assert.Empty(t, m.clients, "failed GetProvision should not cache clients")
		})
	}
}

func TestManager_Shutdown(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   func(config.Loader) datasource.DataSource
	}{
		{
			name:   "defaults",
			config: config.Config{Plugins: GetDefaultPlugins()},
		},
		{
			name:   "gRPC",
			config: config.Config{Plugins: fixtures.Plugins("plugins1")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(&tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			// Mock out the gRPC plugin loader function.
			client := &go_plugin.Client{}
			m.grpcPluginLoader = func(_ context.Context, _ hclog.Logger, _ string, _ *pluginspb.Plugins) (*grpcPlugin, error) {
				return &grpcPlugin{client, newFakeGrpcDataSource(t, configLoader), nil}, nil
			}

			_, err = m.GetDataSource(context.Background())
			require.Nil(t, err)

			m.Shutdown()
			assert.Nil(t, m.source)
			assert.Nil(t, m.provision)
			assert.Empty(t, m.clients)
		})
	}
}

func TestManager_GetPluginConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		pluginName     string
		want           *structpb.Struct
		wantErr        bool
		wantErrMessage string
	}{
		{
			name:       "success",
			config:     &config.Config{Plugins: GetDefaultPlugins(), PluginConfig: fakePluginConfig(t)},
			pluginName: "local",
			want:       fakeLocalPluginConfig(t),
		},
		{
			name:           "non-existent plugin",
			config:         &config.Config{Plugins: GetDefaultPlugins(), PluginConfig: fakePluginConfig(t)},
			pluginName:     "non-existent-plugin",
			want:           fakeLocalPluginConfig(t),
			wantErr:        true,
			wantErrMessage: "no plugin configuration found for non-existent-plugin",
		},
		{
			name:           "no plugin config",
			config:         &config.Config{Plugins: GetDefaultPlugins()},
			pluginName:     "non-existent-plugin",
			want:           fakeLocalPluginConfig(t),
			wantErr:        true,
			wantErrMessage: "no plugin configuration found for non-existent-plugin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			got, err := m.GetPluginConfig(tt.pluginName)

			if tt.wantErr {
				require.Error(t, err, err)
				assert.ErrorContains(t, err, tt.wantErrMessage)
			} else {
				require.Nil(t, err, err)
				assert.EqualExportedValues(t, tt.want, got)

				config, err := configLoader.Read()
				require.Nil(t, err, err)
				assert.EqualExportedValues(t, tt.want, config.PluginConfig[tt.pluginName])

				assert.NotSame(t, config.PluginConfig[tt.pluginName], got, "pointer to plugin config returned")
			}
		})
	}
}

func TestManager_SetPluginConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       *config.Config
		pluginName   string
		pluginConfig *structpb.Struct
	}{
		{
			name:         "success",
			config:       &config.Config{Plugins: GetDefaultPlugins()},
			pluginName:   "local",
			pluginConfig: fakeLocalPluginConfig(t),
		},
		{
			name:         "overwrite",
			config:       &config.Config{Plugins: GetDefaultPlugins(), PluginConfig: fakePluginConfig(t)},
			pluginName:   "local",
			pluginConfig: fakeLocalPluginConfig(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader, err := config.NewMemoryLoader(tt.config)
			require.Nil(t, err)

			m := NewManager(configLoader)
			err = m.SetPluginConfig(tt.pluginName, tt.pluginConfig)

			require.Nil(t, err, err)

			config, err := configLoader.Read()
			require.Nil(t, err, err)
			assert.EqualExportedValues(t, tt.pluginConfig, config.PluginConfig[tt.pluginName])

			assert.NotSame(t, config.PluginConfig[tt.pluginName], tt.pluginConfig, "pointer to plugin config stored in config")
		})
	}
}

func fakePluginConfig(t *testing.T) map[string]*structpb.Struct {
	s := fakeLocalPluginConfig(t)
	return map[string]*structpb.Struct{"local": s}
}

func fakeLocalPluginConfig(t *testing.T) *structpb.Struct {
	s, err := structpb.NewStruct(map[string]any{"fake-opt": "fake-value"})
	require.Nil(t, err, err)
	return s
}
