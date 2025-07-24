// ************************************************************************************************
// Package mcp provides TLS certificate generation utilities for HTTPS support.
package mcp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ************************************************************************************************
// GenerateSelfSignedCert generates a self-signed certificate and private key.
// It creates both certificate and key files at the specified paths.
//
// Parameters:
//   - certPath: Path where the certificate file will be saved
//   - keyPath: Path where the private key file will be saved
//   - hosts: List of hostnames/IPs the certificate should be valid for
//
// Returns:
//   - error: An error if certificate generation fails
func GenerateSelfSignedCert(certPath, keyPath string, hosts []string) error {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Repomix-MCP"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add hosts to certificate
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, host)
		}
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key to file
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// ************************************************************************************************
// LoadTLSConfig loads a TLS configuration from certificate and key files.
// If the files don't exist and autoGenCert is true, it generates new ones.
//
// Parameters:
//   - certPath: Path to the certificate file
//   - keyPath: Path to the private key file
//   - autoGenCert: Whether to auto-generate certificates if they don't exist
//   - hosts: List of hostnames/IPs for certificate generation
//
// Returns:
//   - *tls.Config: The TLS configuration
//   - error: An error if loading or generation fails
func LoadTLSConfig(certPath, keyPath string, autoGenCert bool, hosts []string) (*tls.Config, error) {
	// Check if certificate and key files exist
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	// If files don't exist and auto-generation is enabled, create them
	if (os.IsNotExist(certErr) || os.IsNotExist(keyErr)) && autoGenCert {
		if len(hosts) == 0 {
			hosts = []string{"localhost", "127.0.0.1", "::1"}
		}
		
		if err := GenerateSelfSignedCert(certPath, keyPath, hosts); err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
	}

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate and key: %w", err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "localhost",
	}

	return tlsConfig, nil
}