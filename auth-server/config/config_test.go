package config_test

import (
	"log"
	"os"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func removeTmpFile(tmpFile *os.File) {
	if err := os.Remove(tmpFile.Name()); err != nil {
		log.Printf("failed to remove temporary file %s: %v", tmpFile.Name(), err)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test_config_*.yml")
		require.NoError(t, err)

		defer removeTmpFile(tmpFile)

		_, err = tmpFile.WriteString(`
environment: test
nats:
  url: nats://test:4222
auth:
  issuer_seed: SAAGTESTSEED
  xkey_seed: SXAKTESTSEED
`)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		cfg, err := config.Load(tmpFile.Name())
		require.NoError(t, err)

		assert.Equal(t, "test", cfg.Environment)
		assert.Equal(t, "nats://test:4222", cfg.Nats.URL)
		assert.Equal(t, "SAAGTESTSEED", cfg.Auth.IssuerSeed)
	})

	t.Run("validation failures", func(t *testing.T) {
		tests := []struct {
			name      string
			config    string
			expectErr string
		}{
			{
				"missing issuer seed",
				`auth: { xkey_seed: "SXAK..." }`,
				"auth.issuer_seed is required",
			},
			{
				"missing xkey seed",
				`auth: { issuer_seed: "SAAG..." }`,
				"auth.xkey_seed is required",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpFile, err := os.CreateTemp("", "bad_config_*.yml")
				require.NoError(t, err)

				defer removeTmpFile(tmpFile)

				_, err = tmpFile.WriteString(tt.config)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				_, err = config.Load(tmpFile.Name())
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			})
		}
	})

	t.Run("parse failures", func(t *testing.T) {
		tests := []struct {
			name      string
			config    string
			expectErr string
		}{
			{
				"empty file",
				"",
				"config file is empty", // YAML lib returns EOF for empty files
			},
			{
				"invalid yaml",
				"invalid: [",
				"failed to parse config file",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpFile, err := os.CreateTemp("", "parse_fail_*.yml")
				require.NoError(t, err)
				defer removeTmpFile(tmpFile)

				_, err = tmpFile.WriteString(tt.config)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				_, err = config.Load(tmpFile.Name())
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			})
		}
	})

	t.Run("default environment", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "default_env_*.yml")
		require.NoError(t, err)

		defer removeTmpFile(tmpFile)

		_, err = tmpFile.WriteString(`
nats:
  url: nats://default:4222
auth:
  issuer_seed: SAAGDEFAULT
  xkey_seed: SXAKDEFAULT
`)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		cfg, err := config.Load(tmpFile.Name())
		require.NoError(t, err)
		assert.Equal(t, "development", cfg.Environment)
	})
}

func TestMustLoad(t *testing.T) {
	t.Run("panics on error", func(t *testing.T) {
		assert.PanicsWithValue(t,
			"Failed to load config: failed to read config file: open nonexistent_file.yml: no such file or directory",
			func() { config.MustLoad("nonexistent_file.yml") },
			"MustLoad should panic with error message when config cannot be loaded")
	})

	t.Run("returns config on success", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "valid_config_*.yml")
		require.NoError(t, err)

		defer removeTmpFile(tmpFile)

		_, err = tmpFile.WriteString(`
nats:
  url: nats://localhost:4222
auth:
  issuer_seed: SAAGVALID
  xkey_seed: SXAKVALID
`)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		assert.NotPanics(t, func() {
			cfg := config.MustLoad(tmpFile.Name())
			assert.NotNil(t, cfg)
			assert.Equal(t, "nats://localhost:4222", cfg.Nats.URL)
		}, "MustLoad should return valid config without panicking")
	})
}
