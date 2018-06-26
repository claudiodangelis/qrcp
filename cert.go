package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

// This is copied from https://github.com/bettercap/bettercap/blob/master/tls/cert.go#L55 with a few modifications
// Generate generate cert on the given certPath
// Generate corresponding key will be in keyPath
func Generate(certPath string, keyPath string) error {
	keyfile, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyfile.Close()

	certfile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certfile.Close()

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	aYear := time.Duration(365*24) * time.Hour
	notAfter := notBefore.Add(aYear)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:            []string{"ZH"},
			Locality:           []string{"WUHAN"},
			Organization:       []string{"QR-FileTransfer"},
			OrganizationalUnit: []string{"Development"},
			CommonName:         "claudiodangelis/qr-filetransfer",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA: true,
	}

	cert_raw, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyfile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return err
	}

	return pem.Encode(certfile, &pem.Block{Type: "CERTIFICATE", Bytes: cert_raw})
}
