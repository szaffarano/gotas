package cmd

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pki"
)

func pkiCmd() *cobra.Command {
	var pkiPath string
	var orgName, caCommonName string
	var serverCommonName, clientCommonName string

	pkiCmd := cobra.Command{
		Use:   "pki",
		Short: "Manages the PKI stuff",
	}

	pkiInitCmd := cobra.Command{
		Use:   "init",
		Short: "Initializes the PKI by crating a new CA",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := createIfNotExists(pkiPath); err != nil {
				return err
			}

			certPath, keyPath, err := pairPath("ca", pkiPath)
			if err != nil {
				return err
			}

			caCert, caKey, err := pki.CreateCA(orgName, caCommonName)
			if err != nil {
				return err
			}

			return writePair(certPath, keyPath, caCert, caKey)
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
			caCert, err := loadCakeyPair(pkiPath)
			if err != nil {
				return nil
			}

			certFile, keyFile, err := pairPath(clientCommonName, pkiPath)
			if err != nil {
				return err
			}

			cert, key, err := pki.CreateClientCert(orgName, clientCommonName, caCert)
			if err != nil {
				return err
			}

			return writePair(certFile, keyFile, cert, key)
		},
	}

	pkiAddServerCmd := cobra.Command{
		Use:   "server",
		Short: "Creates a new server certificate",
		RunE: func(_ *cobra.Command, _ []string) error {
			caCert, err := loadCakeyPair(pkiPath)
			if err != nil {
				return err
			}

			certFile, keyFile, err := pairPath(serverCommonName, pkiPath)
			if err != nil {
				return err
			}

			cert, key, err := pki.CreateServerCert(orgName, serverCommonName, caCert)
			if err != nil {
				return err
			}

			return writePair(certFile, keyFile, cert, key)
		},
	}

	pkiCmd.
		PersistentFlags().
		StringVarP(&pkiPath, "pki-path", "p", "", "Base path where PKI certificates are located")
	pkiCmd.
		PersistentFlags().
		StringVarP(&orgName, "org", "o", "Gotas inc.", "Organization Name to assign to the CA")

	if err := pkiCmd.MarkPersistentFlagRequired("pki-path"); err != nil {
		// should never happens
		panic(err)
	}

	pkiInitCmd.
		Flags().
		StringVarP(&caCommonName, "cn", "c", "Gotas inc. server", "Common Name to assign to the CA")

	pkiAddServerCmd.
		Flags().
		StringVarP(&serverCommonName, "cn", "c", "localhost", "Common Name to assign to the server")

	pkiAddClientCmd.
		Flags().
		StringVarP(&clientCommonName, "cn", "c", "user", "Common Name to assign to the client")

	pkiAddCmd.AddCommand(&pkiAddClientCmd, &pkiAddServerCmd)
	pkiCmd.AddCommand(&pkiInitCmd, &pkiAddCmd)

	return &pkiCmd
}

func pairPath(prefix, pkiPath string) (string, string, error) {
	caCertPath := filepath.Join(pkiPath, fmt.Sprintf("%s.pem", prefix))
	caKeyPath := filepath.Join(pkiPath, fmt.Sprintf("%s.key", prefix))

	if fileInfo, err := os.Stat(pkiPath); err != nil {
		return caCertPath, caKeyPath, err
	} else if !fileInfo.IsDir() {
		return caCertPath, caKeyPath, fmt.Errorf("%s: is not a directory", pkiPath)
	}

	if err := exists(caCertPath); err != nil {
		return caCertPath, caKeyPath, err
	}

	if err := exists(caKeyPath); err != nil {
		return caCertPath, caKeyPath, err
	}

	return caCertPath, caKeyPath, nil
}

func writePair(certPath, keyPath string, cert, key []byte) error {
	if err := os.WriteFile(certPath, cert, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return err
	}
	log.Infof("%v: crated successfully", certPath)
	log.Infof("%v: crated successfully", keyPath)
	return nil
}

func exists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s: file exists", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func createIfNotExists(pkiPath string) error {
	if fileInfo, err := os.Stat(pkiPath); os.IsNotExist(err) {
		return os.Mkdir(pkiPath, 0700)
	} else if !fileInfo.IsDir() {
		return fmt.Errorf("%s: is not a directory", pkiPath)
	}
	return nil
}

func loadCakeyPair(pkiPath string) (tls.Certificate, error) {
	caCertPath, caKeyPath, err := pairPath("ca", pkiPath)
	if err == nil {
		// error nil means ca doesn't exists
		// TODO improve!
		return tls.Certificate{}, fmt.Errorf("not initialized pki at %q", pkiPath)
	}

	return tls.LoadX509KeyPair(caCertPath, caKeyPath)
}
