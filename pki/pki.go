package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"
)

const (
	// one year
	defaultExpirationTime = 24 * 365 * time.Hour
)

// CreateCA creates a self signed CA.  The key pair uses P-256 elliptic curve algorithm.
// See https://pkg.go.dev/crypto/ecdsa for further information.
func CreateCA(org string, cn string) ([]byte, []byte, error) {
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

// CreateClientCert creates a new client certificate
func CreateClientCert(name, cn string, caKeyPair tls.Certificate) ([]byte, []byte, error) {
	clientSubject := pkix.Name{
		Organization: []string{name},
		Country:      []string{"AR"},
		Locality:     []string{"Mataderos"},
	}
	return newCert(clientSubject, []string{cn}, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, caKeyPair)
}

// CreateServerCert creates a new server certificate
func CreateServerCert(org, cn string, caKeyPair tls.Certificate) ([]byte, []byte, error) {
	serverSubject := pkix.Name{
		Organization: []string{org},
		Country:      []string{"AR"},
		Locality:     []string{"Mataderos"},
	}

	return newCert(serverSubject, []string{cn}, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, caKeyPair)
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
