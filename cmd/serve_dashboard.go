package cmd

import (
	"crypto/x509"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Ctere1/cert-helper/internal"
)

func handleDashboard(w http.ResponseWriter, r *http.Request, outputDir string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rootCAs, err := internal.ListRootCAs(outputDir)
	errorMessage := r.URL.Query().Get("error")
	if err != nil && errorMessage == "" {
		errorMessage = "Could not read Root CA list."
	}

	intermediateCAs, err := internal.ListAllIntermediateCAs(outputDir)
	if err != nil && errorMessage == "" {
		errorMessage = "Could not read intermediate CA list."
	}

	fileInfos, err := listFileInfos(outputDir, "/files")
	if err != nil && errorMessage == "" {
		errorMessage = "Could not read file list."
	}

	fileBrowserData, err := buildDashboardFileBrowser(outputDir, r.URL.Query().Get("files"))
	if err != nil && errorMessage == "" {
		errorMessage = "Could not read file browser."
	}
	if err != nil {
		fileBrowserData, _ = buildDashboardFileBrowser(outputDir, "/")
	}

	certificates, summary, err := collectCertificates(outputDir)
	if err != nil && errorMessage == "" {
		errorMessage = "Could not read certificate status."
	}

	scepRunning, scepURL, scepPort := detectSCEPStatus()

	data := DashboardData{
		Title:         "Certificate Helper Dashboard",
		Message:       r.URL.Query().Get("message"),
		Error:         errorMessage,
		RootCAs:       rootCAs,
		IssuerOptions: buildIssuerOptions(rootCAs, intermediateCAs),
		Files:         fileInfos,
		Defaults:      defaultFormValues(),
		Summary:       summary,
		Certificates:  certificates,
		OutputDir:     outputDir,
		FileSummary:   buildFileSummary(fileInfos),
		FileBrowser:   fileBrowserData,
		SCEPRunning:   scepRunning,
		SCEPURL:       scepURL,
		SCEPPort:      scepPort,
	}

	tmpl := template.New("dashboard").Funcs(template.FuncMap{
		"formatSize": internal.FormatSize,
		"fileExt":    filepath.Ext,
		"js":         template.JSEscapeString,
	})
	tmpl, err = tmpl.ParseFS(templateFS, dashboardTemplateFile)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, filepath.Base(dashboardTemplateFile), data); err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

func handleGenerateRoot(w http.ResponseWriter, r *http.Request, outputDir string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subject := subjectFromForm(r)
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = subject.CommonName
	}
	validityDays := parseValidityDays(r.FormValue("validity_days"), 3600)
	keyBits := parseKeyBits(r.FormValue("key_bits"), internal.DefaultKeyBits)
	keyUsage := parseKeyUsage(r.Form["key_usage"], internal.DefaultCAKeyUsage)

	_, _, err := internal.GenerateRootCAWithOptions(outputDir, name, subject, validityDays, keyBits, keyUsage)
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("Failed to create root CA: %v", err), true)
		return
	}
	redirectWithMessage(w, r, "Root CA created successfully.", false)
}

func handleGenerateIntermediate(w http.ResponseWriter, r *http.Request, outputDir string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subject := subjectFromForm(r)
	rootName := strings.TrimSpace(r.FormValue("root_name"))
	if rootName == "" {
		redirectWithMessage(w, r, "Root CA selection is required.", true)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = subject.CommonName
	}
	validityDays := parseValidityDays(r.FormValue("validity_days"), 1800)
	keyBits := parseKeyBits(r.FormValue("key_bits"), internal.DefaultKeyBits)
	keyUsage := parseKeyUsage(r.Form["key_usage"], internal.DefaultCAKeyUsage)

	_, _, err := internal.GenerateIntermediateCAWithOptions(outputDir, rootName, name, subject, validityDays, keyBits, keyUsage)
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("Failed to create intermediate CA: %v", err), true)
		return
	}
	redirectWithMessage(w, r, "Intermediate CA created successfully.", false)
}

func handleGenerateCert(w http.ResponseWriter, r *http.Request, outputDir string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subject := subjectFromForm(r)
	issuerType, issuerRoot, issuerName, err := parseIssuerSelection(r.FormValue("issuer"))
	if err != nil {
		redirectWithMessage(w, r, "Signing CA selection is invalid.", true)
		return
	}
	sans := parseSANs(r.FormValue("subject_alt_names"))
	validityDays := parseValidityDays(r.FormValue("validity_days"), 365)
	pfxPassword := r.FormValue("pfx_password")
	keyBits := parseKeyBits(r.FormValue("key_bits"), internal.DefaultKeyBits)
	keyUsage := parseKeyUsage(r.Form["key_usage"], 0)
	extKeyUsage := parseExtKeyUsage(r.Form["extended_key_usage"])
	keyType := parseKeyType(r.FormValue("key_type"))
	exportPrivateKey := r.FormValue("export_private_key") != ""

	options := internal.CertificateOptions{
		KeyBits:          keyBits,
		KeyUsage:         keyUsage,
		ExtKeyUsage:      extKeyUsage,
		KeyType:          keyType,
		ExportPrivateKey: exportPrivateKey,
	}
	_, _, _, err = internal.GenerateCertificateWithOptions(outputDir, issuerType, issuerRoot, issuerName, subject, sans, validityDays, pfxPassword, options)
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("Failed to create certificate: %v", err), true)
		return
	}
	redirectWithMessage(w, r, "Certificate created successfully.", false)
}

func subjectFromForm(r *http.Request) internal.Subject {
	return internal.Subject{
		CommonName:         strings.TrimSpace(r.FormValue("common_name")),
		Organization:       strings.TrimSpace(r.FormValue("organization")),
		OrganizationalUnit: strings.TrimSpace(r.FormValue("organizational_unit")),
		Country:            strings.TrimSpace(r.FormValue("country")),
		Province:           strings.TrimSpace(r.FormValue("state")),
		Locality:           strings.TrimSpace(r.FormValue("locality")),
	}
}

func parseValidityDays(value string, fallback int) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseKeyBits(value string, fallback int) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return internal.NormalizeKeyBits(fallback)
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return internal.NormalizeKeyBits(fallback)
	}
	return internal.NormalizeKeyBits(parsed)
}

func parseKeyUsage(values []string, fallback x509.KeyUsage) x509.KeyUsage {
	if len(values) == 0 {
		return fallback
	}
	var usage x509.KeyUsage
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "digital_signature":
			usage |= x509.KeyUsageDigitalSignature
		case "key_encipherment":
			usage |= x509.KeyUsageKeyEncipherment
		case "data_encipherment":
			usage |= x509.KeyUsageDataEncipherment
		case "key_agreement":
			usage |= x509.KeyUsageKeyAgreement
		case "cert_sign":
			usage |= x509.KeyUsageCertSign
		case "crl_sign":
			usage |= x509.KeyUsageCRLSign
		}
	}
	if usage == 0 {
		return fallback
	}
	return usage
}

func parseExtKeyUsage(values []string) []x509.ExtKeyUsage {
	if len(values) == 0 {
		return nil
	}
	var usages []x509.ExtKeyUsage
	replacer := strings.NewReplacer("\n", " ", "\r", " ")
	for _, value := range values {
		cleaned := strings.ToLower(replacer.Replace(strings.TrimSpace(value)))
		switch cleaned {
		case "server_auth":
			usages = append(usages, x509.ExtKeyUsageServerAuth)
		case "client_auth":
			usages = append(usages, x509.ExtKeyUsageClientAuth)
		case "code_signing":
			usages = append(usages, x509.ExtKeyUsageCodeSigning)
		case "email_protection":
			usages = append(usages, x509.ExtKeyUsageEmailProtection)
		case "time_stamping":
			usages = append(usages, x509.ExtKeyUsageTimeStamping)
		case "ocsp_signing":
			usages = append(usages, x509.ExtKeyUsageOCSPSigning)
		default:
			if cleaned != "" {
				log.Printf("Ignoring unknown extended key usage value: %s", cleaned)
			}
		}
	}
	return usages
}

func parseKeyType(value string) string {
	return internal.NormalizeKeyType(value)
}

func parseSANs(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var sans []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			sans = append(sans, trimmed)
		}
	}
	return sans
}

func parseIssuerSelection(value string) (string, string, string, error) {
	parts := strings.Split(value, ":")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid issuer")
	}
	switch parts[0] {
	case "root":
		return "root", "", parts[1], nil
	case "intermediate":
		if len(parts) < 3 {
			return "", "", "", fmt.Errorf("invalid intermediate issuer")
		}
		return "intermediate", parts[1], parts[2], nil
	default:
		return "", "", "", fmt.Errorf("invalid issuer type")
	}
}

const (
	defaultSCEPPort          = "8001"
	scepDetectionDialTimeout = 100 * time.Millisecond
)

func detectSCEPStatus() (bool, string, string) {
	host := strings.TrimSpace(os.Getenv("CERT_HELPER_SCEP_HOST"))
	configuredPort := strings.TrimSpace(os.Getenv("CERT_HELPER_SCEP_PORT"))
	if host == "" {
		host = "localhost"
	}
	port := configuredPort
	if port == "" {
		port = defaultSCEPPort
	}
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, scepDetectionDialTimeout)
	if err == nil {
		conn.Close()
		return true, fmt.Sprintf("http://%s", address), port
	}
	return false, fmt.Sprintf("http://%s", address), port
}

func buildIssuerOptions(roots []string, intermediates []internal.IntermediateCAInfo) []IssuerOption {
	var options []IssuerOption
	for _, root := range roots {
		options = append(options, IssuerOption{
			Label: fmt.Sprintf("Root CA: %s", root),
			Value: fmt.Sprintf("root:%s", root),
		})
	}
	for _, intermediate := range intermediates {
		options = append(options, IssuerOption{
			Label: fmt.Sprintf("Intermediate CA: %s (root: %s)", intermediate.Name, intermediate.RootName),
			Value: fmt.Sprintf("intermediate:%s:%s", intermediate.RootName, intermediate.Name),
		})
	}
	return options
}

func redirectWithMessage(w http.ResponseWriter, r *http.Request, message string, isError bool) {
	key := "message"
	if isError {
		key = "error"
	}
	redirectURL := fmt.Sprintf("/?%s=%s", key, url.QueryEscape(message))
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func defaultFormValues() DefaultFormValues {
	return DefaultFormValues{
		Organization:           "Example Org",
		OrganizationalUnit:     "IT",
		Country:                "TR",
		State:                  "Ankara",
		Locality:               "Ankara",
		RootCommonName:         "Example Root CA",
		RootName:               "default",
		IntermediateCommonName: "Example Intermediate CA",
		IntermediateName:       "example-intermediate",
		CertificateCommonName:  "service.example.local",
		CertificateSANs:        "service.example.local,service.internal",
	}
}
