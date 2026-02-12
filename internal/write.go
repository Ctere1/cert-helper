package internal

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

func WriteCertificatePEM(filename string, certDER []byte) error {
	certFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	encoded := pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	err = certFile.Close()
	if err != nil {
		return err
	}
	return encoded
}

func WritePrivateKeyPEM(filename string, privateKey crypto.PrivateKey) error {
	keyFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	block, err := pemBlockForPrivateKey(privateKey)
	if err != nil {
		return closeWithError(keyFile, err)
	}
	encoded := pem.Encode(keyFile, block)
	err = keyFile.Close()
	if err != nil {
		return err
	}
	return encoded
}

func closeWithError(file *os.File, err error) error {
	closeErr := file.Close()
	if closeErr != nil {
		return errors.Join(err, closeErr)
	}
	return err
}

func pemBlockForPrivateKey(privateKey crypto.PrivateKey) (*pem.Block, error) {
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}, nil
	case *ecdsa.PrivateKey:
		der, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		return &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: der,
		}, nil
	default:
		der, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		return &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		}, nil
	}
}

func WritePFX(filename string, privateKey crypto.PrivateKey, cert *x509.Certificate, password string) error {
	var pfxData []byte
	var err error

	if password != "" {
		pfxData, err = pkcs12.Legacy.Encode(privateKey, cert, nil, password)
	} else {
		pfxData, err = pkcs12.Legacy.Encode(privateKey, cert, nil, "")
	}

	if err != nil {
		return err
	}

	return os.WriteFile(filename, pfxData, 0644)
}
