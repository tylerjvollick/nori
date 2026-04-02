package config

import (
	"errors"
	"os"
)

// Config holds all configuration values for the Nori server
type Config struct {
	JWTSecret     string
	AdminEmail    string
	AdminPassword string
	AccountName   string
}

// Load reads configuration from environment variables and validates them
func Load() (*Config, error) {
	cfg := &Config{
		JWTSecret:     os.Getenv("NORI_JWT_SECRET"),
		AdminEmail:    os.Getenv("NORI_ADMIN_EMAIL"),
		AdminPassword: os.Getenv("NORI_ADMIN_PASSWORD"),
		AccountName:   os.Getenv("NORI_ACCOUNT_NAME"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ensures all required configuration values are present
func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return errors.New("NORI_JWT_SECRET is required")
	}

	if c.AdminEmail == "" {
		return errors.New("NORI_ADMIN_EMAIL is required")
	}

	if c.AdminPassword == "" {
		return errors.New("NORI_ADMIN_PASSWORD is required")
	}

	if c.AccountName == "" {
		return errors.New("NORI_ACCOUNT_NAME is required")
	}

	return nil
}
