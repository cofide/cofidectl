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

type fakeConnectDataSource struct {
	local.LocalDataSource
}

func newFakeConnectDataSource(t *testing.T, configLoader config.Loader) *fakeConnectDataSource {
	lds, err := local.NewLocalDataSource(configLoader)
	if err != nil {
		t.Fatalf("NewLocalDataSource() error = %v", err)
	}
	return &fakeConnectDataSource{LocalDataSource: *lds}
}

func TestPluginManager_GetPlugin_success(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
		want   func(config.Loader) []cofidectl_plugin.DataSource
	}{
		{
			name:   "empty",
			config: config.Config{Plugins: []string{}},
			want: func(cl config.Loader) []cofidectl_plugin.DataSource {
				lds, err := local.NewLocalDataSource(cl)
				if err != nil {
					t.Fatalf("NewLocalDataSource() error = %v", err)
				}
				return []cofidectl_plugin.DataSource{lds}
			},
		},
		{
			name:   "connect",
			config: config.Config{Plugins: []string{"cofidectl-connect-plugin"}},
			want: func(cl config.Loader) []cofidectl_plugin.DataSource {
				fcds := newFakeConnectDataSource(t, cl)
				return []cofidectl_plugin.DataSource{fcds}
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
			l.loadConnectPlugin = func(logger hclog.Logger) (cofidectl_plugin.DataSource, error) {
				return newFakeConnectDataSource(t, configLoader), nil
			}

			got, err := l.GetPlugin()
			if err != nil {
				t.Fatalf("Loader.GetPlugins() error = %v", err)
			}

			want := tt.want(configLoader)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Loader.GetPlugins() = %v, want %v", got, want)
			}
		})
	}
}

func TestLoader_GetPlugins_failure(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr string
	}{
		{
			name:    "invalid plugin",
			config:  config.Config{Plugins: []string{"invalid"}},
			wantErr: "only the cofidectl-connect-plugin is currently supported",
		},
		{
			name:    "connect plugin load failure",
			config:  config.Config{Plugins: []string{"cofidectl-connect-plugin"}},
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
			l.loadConnectPlugin = func(logger hclog.Logger) (cofidectl_plugin.DataSource, error) {
				return nil, errors.New("failed to create connect plugin")
			}

			_, err = l.GetPlugin()
			if err == nil {
				t.Fatalf("Loader.GetPlugins() did not return error")
			}

			if err.Error() != tt.wantErr {
				t.Fatalf("Loader.GetPlugins() error message = %s, wantErrString %s", err.Error(), tt.wantErr)
			}
		})
	}
}
