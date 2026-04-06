package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppID          string
	PrivateKey     *rsa.PrivateKey
	InstallationID string // optional; empty means auto-discover
}

func Load() (*Config, error) {
	appID := os.Getenv("GH_APP_ID")
	if appID == "" {
		return nil, fmt.Errorf("GH_APP_ID is required (the GitHub App's numeric ID)")
	}
	if _, err := strconv.ParseInt(appID, 10, 64); err != nil {
		return nil, fmt.Errorf("GH_APP_ID must be a numeric ID, got %q", appID)
	}

	key, err := loadPrivateKey()
	if err != nil {
		return nil, err
	}

	return &Config{
		AppID:          appID,
		PrivateKey:     key,
		InstallationID: os.Getenv("GH_APP_INSTALLATION_ID"),
	}, nil
}

func loadPrivateKey() (*rsa.PrivateKey, error) {
	pemData := os.Getenv("GH_APP_PRIVATE_KEY")
	if pemData == "" {
		path := os.Getenv("GH_APP_PRIVATE_KEY_FILE")
		if path == "" {
			return nil, fmt.Errorf("GH_APP_PRIVATE_KEY or GH_APP_PRIVATE_KEY_FILE is required")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading private key file: %w", err)
		}
		pemData = string(data)
	}

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key: no PEM block found")
	}

	// Try PKCS#1 first (GitHub's default), fall back to PKCS#8
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, pkcs8Err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if pkcs8Err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("failed to parse private key: not an RSA key")
		}
		return rsaKey, nil
	}

	return key, nil
}
