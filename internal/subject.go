package internal

import (
	"crypto/x509/pkix"
	"strings"
)

type Subject struct {
	CommonName         string
	Organization       string
	OrganizationalUnit string
	Country            string
	Province           string
	Locality           string
}

func (s Subject) PKIXName() pkix.Name {
	name := pkix.Name{
		CommonName: s.CommonName,
	}
	if s.Organization != "" {
		name.Organization = []string{s.Organization}
	}
	if s.OrganizationalUnit != "" {
		name.OrganizationalUnit = []string{s.OrganizationalUnit}
	}
	if s.Country != "" {
		name.Country = []string{s.Country}
	}
	if s.Province != "" {
		name.Province = []string{s.Province}
	}
	if s.Locality != "" {
		name.Locality = []string{s.Locality}
	}
	return name
}

func ParseSubjectString(subject string) Subject {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return Subject{}
	}
	parts := strings.Split(trimmed, ",")
	if len(parts) == 1 && !strings.Contains(parts[0], "=") {
		return Subject{CommonName: trimmed}
	}
	var result Subject
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(kv[0]))
		value := strings.TrimSpace(kv[1])
		switch key {
		case "CN":
			result.CommonName = value
		case "O":
			result.Organization = value
		case "OU":
			result.OrganizationalUnit = value
		case "C":
			result.Country = value
		case "ST":
			result.Province = value
		case "L":
			result.Locality = value
		}
	}
	return result
}
