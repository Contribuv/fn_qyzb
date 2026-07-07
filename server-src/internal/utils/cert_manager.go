package utils

import (
	"encoding/json"
	"os"
)

type CertManager struct {
	certConfPath string
}

type CertInfo struct {
	Domain   string `json:"domain"`
	CertPath string `json:"certPath"`
	KeyPath  string `json:"keyPath"`
}

type trimCertEntry struct {
	Domain      string `json:"domain"`
	Certificate string `json:"certificate"`
	Fullchain   string `json:"fullchain"`
	PrivateKey  string `json:"privateKey"`
	Used        bool   `json:"used"`
}

var certManager *CertManager

func GetCertManager() *CertManager {
	if certManager == nil {
		certManager = &CertManager{
			certConfPath: getCertConfPath(),
		}
	}
	return certManager
}

func getCertConfPath() string {
	if p := os.Getenv("TRIM_CERT_CONF"); p != "" {
		return p
	}
	return "/usr/trim/etc/network_cert_all.conf"
}

func (m *CertManager) ListCerts() []CertInfo {
	data, err := os.ReadFile(m.certConfPath)
	if err != nil {
		return nil
	}

	var entries []trimCertEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}

	var certs []CertInfo
	for _, e := range entries {
		if !e.Used {
			continue
		}
		certPath := e.Fullchain
		if certPath == "" {
			certPath = e.Certificate
		}
		if certPath == "" || e.PrivateKey == "" {
			continue
		}
		certs = append(certs, CertInfo{
			Domain:   e.Domain,
			CertPath: certPath,
			KeyPath:  e.PrivateKey,
		})
	}
	return certs
}

func (m *CertManager) GetCertByDomain(domain string) *CertInfo {
	certs := m.ListCerts()
	for i := range certs {
		if certs[i].Domain == domain {
			return &certs[i]
		}
	}
	return nil
}
