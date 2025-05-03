package config_test

import (
	"log"
	"os"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/config"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// removeTmpFile удаляет временный файл и логирует ошибку, если удаление не удалось.
func removeTmpFile(tmpFile *os.File) {
	if err := os.Remove(tmpFile.Name()); err != nil {
		log.Printf("failed to remove temporary file %s: %v", tmpFile.Name(), err)
	}
}

// createTempConfigFile создаёт временный YAML-файл с заданным содержимым.
func createTempConfigFile(t *testing.T, content string) *os.File {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test_config_*.yml")
	require.NoError(t, err)

	if content != "" {
		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
	}
	require.NoError(t, tmpFile.Close())

	return tmpFile
}

func TestLoadConfig(t *testing.T) {
	// Сбрасываем viper перед каждым тестом
	viper.Reset()

	t.Run("successful load", func(t *testing.T) {
		tmpFile := createTempConfigFile(t, `
environment: test
nats:
  url: nats://test:4222
  user: test_user
  pass: test_pass
auth:
  issuer_seed: SAAGTESTSEED
  xkey_seed: SXAKTESTSEED
  users_file: /tmp/users.json
`)
		defer removeTmpFile(tmpFile)

		cfg, err := config.Load(tmpFile.Name())
		require.NoError(t, err)

		assert.Equal(t, "test", cfg.Environment)
		assert.Equal(t, "nats://test:4222", cfg.Nats.URL)
		assert.Equal(t, "test_user", cfg.Nats.User)
		assert.Equal(t, "test_pass", cfg.Nats.Pass)
		assert.Equal(t, "SAAGTESTSEED", cfg.Auth.IssuerSeed)
		assert.Equal(t, "SXAKTESTSEED", cfg.Auth.XKeySeed)
		assert.Equal(t, "/tmp/users.json", cfg.Auth.UsersFile)
	})

	t.Run("successful load with environment variables", func(t *testing.T) {
		tmpFile := createTempConfigFile(t, `
environment: test
nats:
  url: nats://test:4222
auth:
  issuer_seed: SAAGTESTSEED
  xkey_seed: SXAKTESTSEED
`)
		defer removeTmpFile(tmpFile)

		// Устанавливаем переменные окружения с проверкой ошибок
		if err := os.Setenv("NATS_URL", "nats://env:4222"); err != nil {
			t.Fatalf("failed to set NATS_URL: %v", err)
		}
		if err := os.Setenv("ENVIRONMENT", "production"); err != nil {
			t.Fatalf("failed to set ENVIRONMENT: %v", err)
		}
		// Отменяем переменные окружения с проверкой ошибок
		defer func() {
			if err := os.Unsetenv("NATS_URL"); err != nil {
				t.Errorf("failed to unset NATS_URL: %v", err)
			}
			if err := os.Unsetenv("ENVIRONMENT"); err != nil {
				t.Errorf("failed to unset ENVIRONMENT: %v", err)
			}
		}()

		cfg, err := config.Load(tmpFile.Name())
		require.NoError(t, err)

		assert.Equal(t, "production", cfg.Environment)
		assert.Equal(t, "nats://env:4222", cfg.Nats.URL)
		assert.Equal(t, "SAAGTESTSEED", cfg.Auth.IssuerSeed)
		assert.Equal(t, "SXAKTESTSEED", cfg.Auth.XKeySeed)
	})

	t.Run("validation failures", func(t *testing.T) {
		tests := []struct {
			name      string
			config    string
			expectErr string
		}{
			{
				"missing issuer seed",
				`auth:
  xkey_seed: "SXAK..."
environment: test`,
				"auth.issuer_seed is required",
			},
			{
				"missing xkey seed",
				`auth:
  issuer_seed: "SAAG..."
environment: test`,
				"auth.xkey_seed is required",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpFile := createTempConfigFile(t, tt.config)
				defer removeTmpFile(tmpFile)

				_, err := config.Load(tmpFile.Name())
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
				"auth.issuer_seed is required",
			},
			{
				"invalid yaml",
				"invalid: [",
				"failed to read config file",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tmpFile := createTempConfigFile(t, tt.config)
				defer removeTmpFile(tmpFile)

				_, err := config.Load(tmpFile.Name())
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			})
		}
	})

	t.Run("default environment", func(t *testing.T) {
		tmpFile := createTempConfigFile(t, `
nats:
  url: nats://default:4222
auth:
  issuer_seed: SAAGDEFAULT
  xkey_seed: SXAKDEFAULT
`)
		defer removeTmpFile(tmpFile)

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
		tmpFile := createTempConfigFile(t, `
nats:
  url: nats://localhost:4222
  user: test_user
  pass: test_pass
auth:
  issuer_seed: SAAGVALID
  xkey_seed: SXAKVALID
  users_file: /tmp/users.json
`)
		defer removeTmpFile(tmpFile)

		assert.NotPanics(t, func() {
			cfg := config.MustLoad(tmpFile.Name())
			assert.NotNil(t, cfg)
			assert.Equal(t, "nats://localhost:4222", cfg.Nats.URL)
			assert.Equal(t, "test_user", cfg.Nats.User)
			assert.Equal(t, "test_pass", cfg.Nats.Pass)
			assert.Equal(t, "SAAGVALID", cfg.Auth.IssuerSeed)
			assert.Equal(t, "SXAKVALID", cfg.Auth.XKeySeed)
			assert.Equal(t, "/tmp/users.json", cfg.Auth.UsersFile)
		}, "MustLoad should return valid config without panicking")
	})
}
