// Copyright 2025 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log/slog"
	"testing"

	cmdcontext "github.com/cofide/cofidectl/pkg/cmd/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_configFlag(t *testing.T) {
	cmdCtx := cmdcontext.NewCommandContext("cofide.yaml", nil)
	defer cmdCtx.Shutdown()

	rootCmd, err := NewRootCommand("cofidectl", "test", cmdCtx).GetRootCommand()
	require.NoError(t, err)

	flag := rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, flag, "--config flag should be registered")
	assert.Equal(t, "cofide.yaml", flag.DefValue)
	assert.Equal(t, "string", flag.Value.Type())
}

func Test_slogLevelFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		level   string
		want    slog.Level
		wantErr bool
	}{
		{name: "debug", level: "debug", want: slog.LevelDebug},
		{name: "warn", level: "warn", want: slog.LevelWarn},
		{name: "info", level: "info", want: slog.LevelInfo},
		{name: "error", level: "error", want: slog.LevelError},
		{name: "debug upper", level: "DEBUG", want: slog.LevelDebug},
		{name: "warn upper", level: "WARN", want: slog.LevelWarn},
		{name: "info upper", level: "INFO", want: slog.LevelInfo},
		{name: "error upper", level: "ERROR", want: slog.LevelError},
		{name: "invalid", level: "invalid level", wantErr: true},
		{name: "warning", level: "warning", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := slogLevelFromString(tt.level)

			if tt.wantErr {
				require.Error(t, err, err)
				assert.ErrorContains(t, err, "unexpected log level")
			} else {
				assert.Equal(t, got, tt.want)
			}
		})
	}
}
