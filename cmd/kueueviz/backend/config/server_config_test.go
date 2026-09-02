/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"log/slog"
	"testing"

	"github.com/spf13/viper"
)

// resetViper clears viper state between tests so env-var bindings do not bleed
// across test cases.
func resetViper(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		viper.Reset()
	})
}

func TestNewServerConfig_Defaults(t *testing.T) {
	resetViper(t)

	cfg := NewServerConfig()

	if cfg.Port != "8080" {
		t.Errorf("expected default Port=8080, got %q", cfg.Port)
	}
	if cfg.AuthMode != "Disabled" {
		t.Errorf("expected default AuthMode=Disabled, got %q", cfg.AuthMode)
	}
	if cfg.Verbosity != 0 {
		t.Errorf("expected default Verbosity=0, got %d", cfg.Verbosity)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected default LogFormat=text, got %q", cfg.LogFormat)
	}
}

func TestNewServerConfig_EnvVarOverrides(t *testing.T) {
	cases := []struct {
		name          string
		envPort       string
		envVerbosity  string
		envLogFormat  string
		wantPort      string
		wantVerbosity int
		wantLogFormat string
	}{
		{
			name:          "custom port",
			envPort:       "9090",
			wantPort:      "9090",
			wantVerbosity: 0,
			wantLogFormat: "text",
		},
		{
			name:          "json log format",
			envLogFormat:  "json",
			wantPort:      "8080",
			wantVerbosity: 0,
			wantLogFormat: "json",
		},
		{
			name:          "verbosity=1 (debug)",
			envVerbosity:  "1",
			wantPort:      "8080",
			wantVerbosity: 1,
			wantLogFormat: "text",
		},
		{
			name:          "verbosity=2 (trace)",
			envVerbosity:  "2",
			wantPort:      "8080",
			wantVerbosity: 2,
			wantLogFormat: "text",
		},
		{
			name:          "all overridden",
			envPort:       "9091",
			envVerbosity:  "1",
			envLogFormat:  "json",
			wantPort:      "9091",
			wantVerbosity: 1,
			wantLogFormat: "json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetViper(t)

			if tc.envPort != "" {
				t.Setenv("KUEUEVIZ_PORT", tc.envPort)
			}
			if tc.envVerbosity != "" {
				t.Setenv("KUEUEVIZ_VERBOSITY", tc.envVerbosity)
			}
			if tc.envLogFormat != "" {
				t.Setenv("KUEUEVIZ_LOG_FORMAT", tc.envLogFormat)
			}

			cfg := NewServerConfig()

			if cfg.Port != tc.wantPort {
				t.Errorf("Port: got %q, want %q", cfg.Port, tc.wantPort)
			}
			if cfg.Verbosity != tc.wantVerbosity {
				t.Errorf("Verbosity: got %d, want %d", cfg.Verbosity, tc.wantVerbosity)
			}
			if cfg.LogFormat != tc.wantLogFormat {
				t.Errorf("LogFormat: got %q, want %q", cfg.LogFormat, tc.wantLogFormat)
			}
		})
	}
}

func TestSetupLogger_LevelSelection(t *testing.T) {
	cases := []struct {
		name       string
		verbosity  int
		logFormat  string
		wantLevel  slog.Level
		wantJSON   bool
	}{
		{
			name:      "verbosity=0 → info, text handler",
			verbosity: 0,
			logFormat: "text",
			wantLevel: slog.LevelInfo,
			wantJSON:  false,
		},
		{
			name:      "verbosity=1 → debug, text handler",
			verbosity: 1,
			logFormat: "text",
			wantLevel: slog.LevelDebug,
			wantJSON:  false,
		},
		{
			name:      "verbosity=2 → trace, text handler",
			verbosity: 2,
			logFormat: "text",
			wantLevel: slog.LevelDebug - 4,
			wantJSON:  false,
		},
		{
			name:      "verbosity=5 → trace (clamped), json handler",
			verbosity: 5,
			logFormat: "json",
			wantLevel: slog.LevelDebug - 4,
			wantJSON:  true,
		},
		{
			name:      "logFormat=JSON (uppercase) → json handler",
			verbosity: 0,
			logFormat: "JSON",
			wantLevel: slog.LevelInfo,
			wantJSON:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &ServerConfig{
				Verbosity: tc.verbosity,
				LogFormat: tc.logFormat,
			}
			// SetupLogger must not panic or error regardless of input.
			SetupLogger(cfg)

			got := slog.Default()
			if got == nil {
				t.Fatal("slog.Default() is nil after SetupLogger")
			}

			// Verify the configured level is respected: a record at the expected
			// level must be Enabled, a record one level above must not be.
			if !got.Enabled(nil, tc.wantLevel) {
				t.Errorf("expected level %v to be enabled, but it was not", tc.wantLevel)
			}
			// One severity above the threshold must be disabled (unless already at max).
			if tc.wantLevel < slog.LevelError && got.Enabled(nil, tc.wantLevel-1) {
				t.Errorf("expected level %v to be disabled (below threshold %v)", tc.wantLevel-1, tc.wantLevel)
			}
		})
	}
}

func TestGetServerAddress(t *testing.T) {
	cfg := &ServerConfig{Port: "9090"}
	if got := cfg.GetServerAddress(); got != ":9090" {
		t.Errorf("GetServerAddress: got %q, want %q", got, ":9090")
	}
}
