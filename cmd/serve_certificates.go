package cmd

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const expiringSoonDays = 30

func collectCertificates(outputDir string) ([]CertificateEntry, CertificateSummary, error) {
	var entries []CertificateEntry
	now := time.Now()

	err := filepath.WalkDir(outputDir, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".pem" {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}
		block, _ := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(outputDir, filePath)
		if err != nil {
			return nil
		}
		status, statusClass, daysLeft := certificateStatus(cert.NotAfter, now)
		entryType := certificateType(cert, relPath)
		name := cert.Subject.CommonName
		if name == "" {
			name = filepath.Base(filePath)
		}
		issuer := cert.Issuer.CommonName
		if issuer == "" {
			issuer = cert.Issuer.String()
		}
		entries = append(entries, CertificateEntry{
			Name:             name,
			Type:             entryType,
			Issuer:           issuer,
			NotBefore:        cert.NotBefore,
			NotAfter:         cert.NotAfter,
			DaysLeft:         daysLeft,
			Status:           status,
			StatusClass:      statusClass,
			Path:             path.Join("/files", filepath.ToSlash(relPath)),
			SystemPath:       filePath,
			FolderPath:       normalizeURLPath(path.Dir(path.Join("/files", filepath.ToSlash(relPath)))),
			SystemFolderPath: filepath.Dir(filePath),
		})
		return nil
	})
	if err != nil {
		return nil, CertificateSummary{}, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].NotAfter.Before(entries[j].NotAfter)
	})

	summary := buildCertificateSummary(entries)
	return entries, summary, nil
}

func certificateType(cert *x509.Certificate, relPath string) string {
	if cert.IsCA {
		if strings.Contains(filepath.ToSlash(relPath), "ca/intermediate/") {
			return "Intermediate CA"
		}
		return "Root CA"
	}
	return "Certificate"
}

func certificateStatus(expiry, now time.Time) (string, string, int) {
	daysLeft := int(math.Max(0, expiry.Sub(now).Hours()/24))
	if expiry.Before(now) {
		return "Expired", "expired", daysLeft
	}
	if expiry.Before(now.AddDate(0, 0, expiringSoonDays)) {
		return "Expiring Soon", "expiring", daysLeft
	}
	return "Valid", "valid", daysLeft
}

func buildCertificateSummary(entries []CertificateEntry) CertificateSummary {
	summary := CertificateSummary{
		Total:            len(entries),
		ExpiringDaysHint: fmt.Sprintf("Renew within %d days", expiringSoonDays),
	}
	for _, entry := range entries {
		switch entry.StatusClass {
		case "expired":
			summary.Expired++
		case "expiring":
			summary.Expiring++
		default:
			summary.Valid++
		}
	}

	if summary.Total > 0 {
		summary.ValidPercent = percent(summary.Valid, summary.Total)
		summary.ExpiringPercent = percent(summary.Expiring, summary.Total)
		summary.ExpiredPercent = percent(summary.Expired, summary.Total)
	}

	for _, entry := range entries {
		if entry.StatusClass != "expired" {
			summary.NextExpiryName = entry.Name
			summary.NextExpiryDate = entry.NotAfter.Format("2006-01-02")
			break
		}
	}
	return summary
}

func percent(part, total int) int {
	if total == 0 {
		return 0
	}
	return int(float64(part) / float64(total) * 100)
}
