package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/fsperm"
	"github.com/svetlyopet/mindpalace/internal/vault"
	"gopkg.in/yaml.v3"
)

const (
	defaultVaultDir = ".mindpalace"
	envVault        = "MINDPALACE_VAULT"
)

// Config is vault-local settings from config.yaml at the vault root (or legacy .mindpalace/config.yaml).
type Config struct {
	LLM     LLMConfig     `yaml:"llm"`
	Capture CaptureConfig `yaml:"capture"`
	Serve   ServeConfig   `yaml:"serve"`
	Vault   VaultConfig   `yaml:"vault"`
	Editor  string        `yaml:"editor"`
}

type LLMConfig struct {
	Backend   string `yaml:"backend"`
	Model     string `yaml:"model"`
	BaseURL   string `yaml:"base_url"`
	APIKeyEnv string `yaml:"api_key_env"`
}

func (c LLMConfig) Enabled() bool {
	return c.Backend != "" && c.Backend != "none"
}

func (c LLMConfig) APIKey() string {
	if c.APIKeyEnv == "" {
		return ""
	}
	return os.Getenv(c.APIKeyEnv)
}

type CaptureConfig struct {
	AutoTag      bool   `yaml:"auto_tag"`
	FullHTML     bool   `yaml:"full_html"`
	OCR          string `yaml:"ocr"`
	SocialOEmbed bool   `yaml:"social_oembed"`
}

func Default() *Config {
	return &Config{
		LLM: LLMConfig{
			Backend:   "none",
			Model:     "llama3.1",
			BaseURL:   "http://localhost:11434",
			APIKeyEnv: "MINDPALACE_API_KEY",
		},
		Capture: CaptureConfig{
			AutoTag:      true,
			FullHTML:     false,
			OCR:          "auto",
			SocialOEmbed: true,
		},
		Serve: ServeConfig{
			Addr: defaultServeAddr,
		},
		Editor: "",
	}
}

func Load(vaultRoot string) (*Config, error) {
	cfg := Default()
	path := vault.ConfigPath(vaultRoot)
	data, err := vault.ReadFileBytes(path, nil)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ResolveVaultRoot picks the vault path: flag > MINDPALACE_VAULT > ~/.mindpalace.
func ResolveVaultRoot(flagValue string) (string, error) {
	if flagValue != "" {
		return filepath.Clean(flagValue), nil
	}
	if v := os.Getenv(envVault); v != "" {
		return filepath.Clean(v), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, defaultVaultDir), nil
}

// WriteDefault saves default config under the vault config.yaml (flat layout).
func WriteDefault(vaultRoot string) error {
	path := vault.ConfigPath(vaultRoot)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fsperm.DirMode); err != nil {
		return err
	}
	cfg := Default()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	header := "# Mindpalace vault config\n"
	return os.WriteFile(path, append([]byte(header), data...), fsperm.PrivateFileMode)
}

// EditorCommand returns configured editor, $EDITOR, or vim.
func (c *Config) EditorCommand() (string, error) {
	ed := strings.TrimSpace(c.Editor)
	if ed == "" {
		ed = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if ed == "" {
		ed = "vim"
	}
	if _, err := exec.LookPath(ed); err != nil {
		return "", fmt.Errorf("editor %q not found on PATH", ed)
	}
	return ed, nil
}
