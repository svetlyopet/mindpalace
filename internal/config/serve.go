package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/svetlyopet/mindpalace/internal/vault"
	"gopkg.in/yaml.v3"
)

const defaultServeAddr = "127.0.0.1:7451"

type ServeConfig struct {
	Addr  string `yaml:"addr"`
	Token string `yaml:"token"`
}

func (c *Config) validate() error {
	if err := c.validateLLM(); err != nil {
		return err
	}
	if err := c.validateCapture(); err != nil {
		return err
	}
	if err := c.validateServe(false); err != nil {
		return err
	}
	return c.validateVault()
}

func (c *Config) validateLLM() error {
	switch c.LLM.Backend {
	case "", "none", "ollama", "openai":
	default:
		return fmt.Errorf("invalid llm.backend %q (want none, ollama, openai)", c.LLM.Backend)
	}
	return nil
}

func (c *Config) validateCapture() error {
	switch c.Capture.OCR {
	case "", "auto", "off":
	default:
		return fmt.Errorf("invalid capture.ocr %q (want auto, off)", c.Capture.OCR)
	}
	return nil
}

// ValidateServeAddr checks listen address; allowWildcard permits 0.0.0.0 and ::.
func ValidateServeAddr(addr string, allowWildcard bool) error {
	if strings.TrimSpace(addr) == "" {
		return fmt.Errorf("serve.addr is empty")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("serve.addr %q: %w", addr, err)
	}
	if !allowWildcard && (host == "0.0.0.0" || host == "::") {
		return fmt.Errorf("serve.addr %q binds all interfaces; use 127.0.0.1 or pass --allow-wildcard-bind", addr)
	}
	return nil
}

func (c *Config) validateServe(allowWildcard bool) error {
	if c.Serve.Addr == "" {
		c.Serve.Addr = defaultServeAddr
	}
	return ValidateServeAddr(c.Serve.Addr, allowWildcard)
}

// PrepareServe validates listen address before starting the server.
func (c *Config) PrepareServe(allowWildcard bool) error {
	return c.validateServe(allowWildcard)
}

// Save writes config to the vault config.yaml atomically.
func Save(vaultRoot string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	path := vault.ConfigPath(vaultRoot)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	header := "# Mindpalace vault config\n"
	payload := append([]byte(header), data...)
	tmp, err := os.CreateTemp(dir, "config-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// EnsureServeToken generates and persists a token when missing.
func EnsureServeToken(vaultRoot string, cfg *Config) (string, error) {
	if cfg.Serve.Token != "" {
		return cfg.Serve.Token, nil
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	cfg.Serve.Token = base64.RawURLEncoding.EncodeToString(b)
	if err := Save(vaultRoot, cfg); err != nil {
		return "", err
	}
	return cfg.Serve.Token, nil
}
