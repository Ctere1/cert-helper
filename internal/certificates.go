package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

const (
	rootCAFolder         = "ca/root"
	intermediateCAFolder = "ca/intermediate"
	certsFolder          = "certs"
	DefaultCAKeyUsage    = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign
)

type IntermediateCAInfo struct {
	RootName string
	Name     string
}

type CertificateOptions struct {
	KeyBits          int
	KeyUsage         x509.KeyUsage
	ExtKeyUsage      []x509.ExtKeyUsage
	KeyType          string
	ExportPrivateKey bool
}

func DefaultCertificateOptions() CertificateOptions {
	return CertificateOptions{
		KeyBits:          DefaultKeyBits,
		KeyType:          KeyTypeRSA,
		ExportPrivateKey: true,
	}
}

func DefaultExtKeyUsage() []x509.ExtKeyUsage {
	return []x509.ExtKeyUsage{
		x509.ExtKeyUsageClientAuth,
		x509.ExtKeyUsageServerAuth,
	}
}

func NormalizeName(value, fallback string) string {
	sanitized := sanitizeName(value)
	if sanitized == "" {
		return fallback
	}
	return sanitized
}

func sanitizeName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	invalid := regexp.MustCompile(`[^\w\-]+`)
	sanitized := invalid.ReplaceAllString(trimmed, "_")
	return strings.Trim(sanitized, "_")
}

func rootCAPaths(outputDir, name string) (string, string) {
	rootName := NormalizeName(name, "default")
	if rootName == "default" {
		return filepath.Join(outputDir, "ca.pem"), filepath.Join(outputDir, "ca.key")
	}
	caDir := filepath.Join(outputDir, rootCAFolder, rootName)
	return filepath.Join(caDir, "ca.pem"), filepath.Join(caDir, "ca.key")
}

func intermediateCAPaths(outputDir, rootName, name string) (string, string) {
	rootName = NormalizeName(rootName, "default")
	intermediateName := NormalizeName(name, "intermediate")
	caDir := filepath.Join(outputDir, intermediateCAFolder, rootName, intermediateName)
	return filepath.Join(caDir, "ca.pem"), filepath.Join(caDir, "ca.key")
}

func rootCertDir(outputDir, rootName string) string {
	return filepath.Join(outputDir, certsFolder, "root", NormalizeName(rootName, "default"))
}

func intermediateCertDir(outputDir, rootName, name string) string {
	rootName = NormalizeName(rootName, "default")
	intermediateName := NormalizeName(name, "intermediate")
	return filepath.Join(outputDir, certsFolder, "intermediate", rootName, intermediateName)
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

func GenerateRootCA(outputDir, name string, subject Subject, validityDays int) (string, string, error) {
	return GenerateRootCAWithOptions(outputDir, name, subject, validityDays, 0, 0)
}

func GenerateRootCAWithOptions(outputDir, name string, subject Subject, validityDays int, keyBits int, keyUsage x509.KeyUsage) (string, string, error) {
	if subject.CommonName == "" {
		return "", "", fmt.Errorf("common name is required")
	}
	certPath, keyPath := rootCAPaths(outputDir, name)
	if err := ensureParentDir(certPath); err != nil {
		return "", "", err
	}

	privateKey, err := GeneratePrivateKeyWithBits(keyBits)
	if err != nil {
		return "", "", err
	}

	if keyUsage == 0 {
		keyUsage = DefaultCAKeyUsage
	}

	template := &x509.Certificate{
		Subject:               subject.PKIXName(),
		Issuer:                subject.PKIXName(),
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(time.Duration(validityDays) * 24 * time.Hour),
		SerialNumber:          GenerateSerialNumber(),
		PublicKey:             &privateKey.PublicKey,
		IsCA:                  true,
		MaxPathLen:            -1,
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	if err := WriteCertificatePEM(certPath, certDER); err != nil {
		return "", "", err
	}
	if err := WritePrivateKeyPEM(keyPath, privateKey); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

func GenerateIntermediateCA(outputDir, rootName, name string, subject Subject, validityDays int) (string, string, error) {
	return GenerateIntermediateCAWithOptions(outputDir, rootName, name, subject, validityDays, 0, 0)
}

func GenerateIntermediateCAWithOptions(outputDir, rootName, name string, subject Subject, validityDays int, keyBits int, keyUsage x509.KeyUsage) (string, string, error) {
	if subject.CommonName == "" {
		return "", "", fmt.Errorf("common name is required")
	}

	rootCertPath, rootKeyPath := rootCAPaths(outputDir, rootName)
	rootKey, err := LoadCAPrivateKey(rootKeyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to load root CA private key: %w", err)
	}
	rootCert, err := LoadCACertificate(rootCertPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to load root CA certificate: %w", err)
	}

	certPath, keyPath := intermediateCAPaths(outputDir, rootName, name)
	if err := ensureParentDir(certPath); err != nil {
		return "", "", err
	}

	privateKey, err := GeneratePrivateKeyWithBits(keyBits)
	if err != nil {
		return "", "", err
	}

	if keyUsage == 0 {
		keyUsage = DefaultCAKeyUsage
	}

	template := &x509.Certificate{
		Subject:               subject.PKIXName(),
		Issuer:                rootCert.Subject,
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(time.Duration(validityDays) * 24 * time.Hour),
		SerialNumber:          GenerateSerialNumber(),
		PublicKey:             &privateKey.PublicKey,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert, &privateKey.PublicKey, rootKey)
	if err != nil {
		return "", "", err
	}

	if err := WriteCertificatePEM(certPath, certDER); err != nil {
		return "", "", err
	}
	if err := WritePrivateKeyPEM(keyPath, privateKey); err != nil {
		return "", "", err
	}
	return certPath, keyPath, nil
}

func GenerateCertificate(outputDir, issuerType, rootName, issuerName string, subject Subject, subjectAltNames []string, validityDays int, pfxPassword string) (string, string, string, error) {
	return GenerateCertificateWithOptions(outputDir, issuerType, rootName, issuerName, subject, subjectAltNames, validityDays, pfxPassword, DefaultCertificateOptions())
}

func GenerateCertificateWithOptions(outputDir, issuerType, rootName, issuerName string, subject Subject, subjectAltNames []string, validityDays int, pfxPassword string, options CertificateOptions) (string, string, string, error) {
	if subject.CommonName == "" {
		return "", "", "", fmt.Errorf("common name is required")
	}

	issuerType = strings.ToLower(strings.TrimSpace(issuerType))
	if issuerType == "" {
		issuerType = "root"
	}

	var (
		caCert   *x509.Certificate
		caKey    *rsa.PrivateKey
		certDir  string
		certPath string
		keyPath  string
		pfxPath  string
		err      error
	)

	switch issuerType {
	case "intermediate":
		rootName = NormalizeName(rootName, "default")
		if issuerName == "" {
			return "", "", "", fmt.Errorf("intermediate CA name is required")
		}
		caCertPath, caKeyPath := intermediateCAPaths(outputDir, rootName, issuerName)
		caKey, err = LoadCAPrivateKey(caKeyPath)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to load intermediate CA private key: %w", err)
		}
		caCert, err = LoadCACertificate(caCertPath)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to load intermediate CA certificate: %w", err)
		}
		certDir = intermediateCertDir(outputDir, rootName, issuerName)
	default:
		issuerName = NormalizeName(issuerName, "default")
		caCertPath, caKeyPath := rootCAPaths(outputDir, issuerName)
		caKey, err = LoadCAPrivateKey(caKeyPath)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to load root CA private key: %w", err)
		}
		caCert, err = LoadCACertificate(caCertPath)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to load root CA certificate: %w", err)
		}
		certDir = rootCertDir(outputDir, issuerName)
	}

	privateKey, publicKey, err := GenerateKeyPair(options.KeyType, options.KeyBits)
	if err != nil {
		return "", "", "", err
	}

	normalizedSANs, err := normalizeSANs(append([]string{subject.CommonName}, subjectAltNames...))
	if err != nil {
		return "", "", "", err
	}

	extKeyUsage := options.ExtKeyUsage
	if len(extKeyUsage) == 0 {
		extKeyUsage = DefaultExtKeyUsage()
	}

	template := &x509.Certificate{
		Subject:      subject.PKIXName(),
		Issuer:       caCert.Subject,
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     time.Now().Add(time.Duration(validityDays) * 24 * time.Hour),
		SerialNumber: GenerateSerialNumber(),
		PublicKey:    publicKey,
		ExtKeyUsage:  extKeyUsage,
		DNSNames:     normalizedSANs,
	}
	if options.KeyUsage != 0 {
		template.KeyUsage = options.KeyUsage
	}

	for _, san := range normalizedSANs {
		if ip := net.ParseIP(san); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
			template.DNSNames = RemoveFromSlice(template.DNSNames, san)
		}
	}

	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return "", "", "", err
	}
	safeCommonName := NormalizeName(subject.CommonName, "certificate")
	certPath = filepath.Join(certDir, fmt.Sprintf("cert_%s.pem", safeCommonName))
	keyPath = filepath.Join(certDir, fmt.Sprintf("cert_%s.key", safeCommonName))
	pfxPath = filepath.Join(certDir, fmt.Sprintf("cert_%s.pfx", safeCommonName))

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, publicKey, caKey)
	if err != nil {
		return "", "", "", err
	}

	if err := WriteCertificatePEM(certPath, certDER); err != nil {
		return "", "", "", err
	}
	if options.ExportPrivateKey {
		if err := WritePrivateKeyPEM(keyPath, privateKey); err != nil {
			return "", "", "", err
		}
		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			return "", "", "", err
		}
		if err := WritePFX(pfxPath, privateKey, cert, pfxPassword); err != nil {
			return "", "", "", err
		}
	} else {
		keyPath = ""
		pfxPath = ""
	}

	return certPath, keyPath, pfxPath, nil
}

func ListRootCAs(outputDir string) ([]string, error) {
	var roots []string
	defaultCert := filepath.Join(outputDir, "ca.pem")
	defaultKey := filepath.Join(outputDir, "ca.key")
	if fileExists(defaultCert) && fileExists(defaultKey) {
		roots = append(roots, "default")
	}

	rootDir := filepath.Join(outputDir, rootCAFolder)
	entries, err := os.ReadDir(rootDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			roots = append(roots, entry.Name())
		}
	}
	sort.Strings(roots)
	return roots, nil
}

func ListIntermediateCAs(outputDir, rootName string) ([]string, error) {
	rootName = NormalizeName(rootName, "default")
	var names []string
	intermediateDir := filepath.Join(outputDir, intermediateCAFolder, rootName)
	entries, err := os.ReadDir(intermediateDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func ListAllIntermediateCAs(outputDir string) ([]IntermediateCAInfo, error) {
	var all []IntermediateCAInfo
	rootDir := filepath.Join(outputDir, intermediateCAFolder)
	entries, err := os.ReadDir(rootDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, rootEntry := range entries {
		if !rootEntry.IsDir() {
			continue
		}
		intermediateDir := filepath.Join(rootDir, rootEntry.Name())
		intermediateEntries, err := os.ReadDir(intermediateDir)
		if err != nil {
			return nil, err
		}
		for _, intermediate := range intermediateEntries {
			if intermediate.IsDir() {
				all = append(all, IntermediateCAInfo{
					RootName: rootEntry.Name(),
					Name:     intermediate.Name(),
				})
			}
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].RootName != all[j].RootName {
			return all[i].RootName < all[j].RootName
		}
		return all[i].Name < all[j].Name
	})
	return all, nil
}

func normalizeSANs(sans []string) ([]string, error) {
	var normalized []string
	for _, san := range sans {
		trimmed := strings.TrimSpace(san)
		if trimmed == "" {
			continue
		}
		sanASCII, err := idna.ToASCII(trimmed)
		if err != nil {
			return nil, fmt.Errorf("failed to convert SAN %s to ASCII: %w", trimmed, err)
		}
		normalized = append(normalized, sanASCII)
	}
	return normalized, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
