package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

const (
	// one year
	defaultExpirationTime = 24 * 365 * time.Hour
)

func pkiCmd() *cobra.Command {
	var pkiPath string
	var orgName, commonName string

	pkiCmd := cobra.Command{
		Use:   "pki",
		Short: "Manages the PKI stuff",
	}

	pkiInitCmd := cobra.Command{
		Use:   "init",
		Short: "Initializes the PKI by crating a new CA",
		RunE: func(_ *cobra.Command, _ []string) error {
			log.Infof("Initializing PKI...")
			if fileInfo, err := os.Stat(pkiPath); err != nil {
				return err
			} else if !fileInfo.IsDir() {
				return fmt.Errorf("%s: is not a directory", pkiPath)
			}

			caCertPath := filepath.Join(pkiPath, "ca.pem")
			caKeyPath := filepath.Join(pkiPath, "ca.key")
			// TODO verify if the files already exist

			caCert, caKey, err := initCA(orgName, commonName)
			if err != nil {
				return err
			}

			if err = os.WriteFile(caCertPath, caCert, 0644); err != nil {
				return err
			}
			if err = os.WriteFile(caKeyPath, caKey, 0600); err != nil {
				return err
			}

			return nil
		},
	}

	pkiAddCmd := cobra.Command{
		Use:   "add",
		Short: "Creates a new certificate",
	}

	pkiAddClientCmd := cobra.Command{
		Use:   "client",
		Short: "Creates a new client certificate",
		RunE: func(_ *cobra.Command, _ []string) error {
			log.Info("not implemented")
			return nil
		},
	}

	pkiAddServerCmd := cobra.Command{
		Use:   "server",
		Short: "Creates a new server certificate",
		RunE: func(_ *cobra.Command, _ []string) error {
			log.Info("not implemented")
			return nil
		},
	}

	pkiCmd.
		PersistentFlags().
		StringVarP(&pkiPath, "pki-path", "p", "", "Base path where PKI certificates are located")
	pkiCmd.MarkPersistentFlagRequired("pki-path")

	pkiInitCmd.
		Flags().
		StringVarP(&orgName, "org", "o", "Gotas inc.", "Organization Name to assign to the CA")
	pkiInitCmd.
		Flags().
		StringVarP(&commonName, "cn", "c", "Gotas inc. CA", "Common Name to assign to the CA")

	pkiAddCmd.AddCommand(&pkiAddClientCmd, &pkiAddServerCmd)
	pkiCmd.AddCommand(&pkiInitCmd, &pkiAddCmd)

	return &pkiCmd
}

// initPKI creates a CA and a server cert
func initPKI() error {

	caCert, caKey, err := initCA("Gotas inc.", "Gotas inc CA")
	if err != nil {
		return err
	}

	if err = os.WriteFile("ca.pem", caCert, 0644); err != nil {
		return err
	}
	if err = os.WriteFile("ca.key", caKey, 0600); err != nil {
		return err
	}

	caKeyPair, err := tls.LoadX509KeyPair("ca.pem", "ca.key")
	if err != nil {
		return err
	}

	serverSubject := pkix.Name{
		Organization: []string{"Gotas inc Task Server"},
		Country:      []string{"AR"},
		Locality:     []string{"Mataderos"},
	}
	serverCert, serverKey, err := newCert(
		serverSubject, []string{"localhost"},
		[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		caKeyPair)
	if err != nil {
		return err
	}
	if err = os.WriteFile("server.pem", serverCert, 0644); err != nil {
		return err
	}
	if err = os.WriteFile("server.key", serverKey, 0600); err != nil {
		return err
	}

	clientSubject := pkix.Name{
		Organization: []string{"John Doe"},
		Country:      []string{"AR"},
		Locality:     []string{"Mataderos"},
	}
	clientCert, clientKey, err := newCert(clientSubject,
		[]string{"localhost"},
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		caKeyPair)
	if err != nil {
		return err
	}
	if err = os.WriteFile("client.pem", clientCert, 0644); err != nil {
		return err
	}
	if err = os.WriteFile("client.key", clientKey, 0600); err != nil {
		return err
	}

	return nil
}

// newCerts creates a new X509 certificate signed with the provided CA certificate
func newCert(subject pkix.Name,
	dnsNames []string,
	extensions []x509.ExtKeyUsage,
	caKeyPair tls.Certificate) ([]byte, []byte, error) {

	// take the first block
	caCert, err := x509.ParseCertificate(caKeyPair.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := serialNumber()
	if err != nil {
		return nil, nil, err
	}
	certTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(defaultExpirationTime),
		DNSNames:     dnsNames,

		ExtKeyUsage:           extensions,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	certRaw, err := x509.CreateCertificate(rand.Reader, certTemplate, caCert, &privateKey.PublicKey, caKeyPair.PrivateKey)
	if err != nil {
		return nil, nil, err
	}

	return encode(certRaw, privateKey)
}

// initCA creates a self signed CA.  The key pair uses P-256 elliptic curve algorithm.
// See https://pkg.go.dev/crypto/ecdsa for further information.
func initCA(org string, cn string) ([]byte, []byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := serialNumber()
	if err != nil {
		return nil, nil, err
	}

	ca := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
			CommonName:   cn,
		},

		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(defaultExpirationTime),

		BasicConstraintsValid: true,
		IsCA:                  true,

		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	caCertRaw, err := x509.CreateCertificate(rand.Reader, &ca, &ca, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	return encode(caCertRaw, privateKey)
}

// encode marshals a certificate to byte arrays
func encode(certRaw []byte, privateKey *ecdsa.PrivateKey) ([]byte, []byte, error) {
	cert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certRaw,
	})
	if cert == nil {
		return nil, nil, errors.New("error encoding certificate: nil")
	}

	keyRaw, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyRaw})
	if key == nil {
		return nil, nil, errors.New("error encoding key: nil")
	}

	return cert, key, nil

}

// serialNumber generates a random number up to 2^128
func serialNumber() (*big.Int, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	return serialNumber, err
}
