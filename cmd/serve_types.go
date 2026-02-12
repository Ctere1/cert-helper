package cmd

import "time"

type FileInfo struct {
	Name              string
	Size              int64
	ModTime           time.Time
	IsDir             bool
	Path              string
	BrowserPath       string
	SystemPath        string
	FolderPath        string
	BrowserFolderPath string
	SystemFolderPath  string
}

type FileSummary struct {
	Total        int
	Directories  int
	Certificates int
	Keys         int
	Bundles      int
	TotalSize    int64
}

type PageData struct {
	CurrentPath string
	ParentPath  string
	Files       []FileInfo
	Title       string
	RootPath    string
	Summary     FileSummary
}

type IssuerOption struct {
	Label string
	Value string
}

type DefaultFormValues struct {
	Organization           string
	OrganizationalUnit     string
	Country                string
	State                  string
	Locality               string
	RootCommonName         string
	RootName               string
	IntermediateCommonName string
	IntermediateName       string
	CertificateCommonName  string
	CertificateSANs        string
}

type CertificateSummary struct {
	Total            int
	Valid            int
	Expiring         int
	Expired          int
	ValidPercent     int
	ExpiringPercent  int
	ExpiredPercent   int
	NextExpiryName   string
	NextExpiryDate   string
	ExpiringDaysHint string
}

type CertificateEntry struct {
	Name             string
	Type             string
	Issuer           string
	NotBefore        time.Time
	NotAfter         time.Time
	DaysLeft         int
	Status           string
	StatusClass      string
	Path             string
	SystemPath       string
	FolderPath       string
	SystemFolderPath string
}

type DashboardData struct {
	Title         string
	Message       string
	Error         string
	RootCAs       []string
	IssuerOptions []IssuerOption
	Files         []FileInfo
	Defaults      DefaultFormValues
	Summary       CertificateSummary
	Certificates  []CertificateEntry
	OutputDir     string
	FileSummary   FileSummary
	FileBrowser   PageData
	SCEPRunning   bool
	SCEPURL       string
	SCEPPort      string
}
