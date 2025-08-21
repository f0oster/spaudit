package spauth

import (
	"fmt"
	"os"

	"github.com/koltyakov/gosip"
	"github.com/koltyakov/gosip/auth/azurecert"
)

type Config struct {
	SiteURL      string
	TenantID     string
	ClientID     string
	CertPath     string
	CertPassword string
}

func FromEnv() (Config, error) {
	// Environment should already be loaded by main.go
	cfg := Config{
		SiteURL:      os.Getenv("SP_SITE_URL"),
		TenantID:     os.Getenv("SP_TENANT_ID"),
		ClientID:     os.Getenv("SP_CLIENT_ID"),
		CertPath:     os.Getenv("SP_CERT_PATH"),
		CertPassword: os.Getenv("SP_CERT_PASSWORD"),
	}

	if cfg.SiteURL == "" || cfg.TenantID == "" || cfg.ClientID == "" || cfg.CertPath == "" {
		return cfg, fmt.Errorf("missing required configuration: SP_SITE_URL, SP_TENANT_ID, SP_CLIENT_ID, SP_CERT_PATH")
	}
	return cfg, nil
}

func NewClient(cfg Config) (*gosip.SPClient, error) {
	ac := &azurecert.AuthCnfg{
		SiteURL:  cfg.SiteURL,
		TenantID: cfg.TenantID,
		ClientID: cfg.ClientID,
		CertPath: cfg.CertPath,
		CertPass: cfg.CertPassword,
	}
	client := &gosip.SPClient{AuthCnfg: ac}
	return client, nil
}
