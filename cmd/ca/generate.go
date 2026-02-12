package ca

import (
	"fmt"

	"github.com/Ctere1/cert-helper/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	caValidityDays int
	caSubject      string
	caName         string
	caCommonName   string
	caOrganization string
	caOrgUnit      string
	caCountry      string
	caState        string
	caLocality     string
)

var caGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new Certificate Authority.",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, err := cmd.Flags().GetString("output-dir")
		if err != nil {
			return err
		}

		subject := internal.ParseSubjectString(caSubject)
		if caCommonName != "" {
			subject.CommonName = caCommonName
		}
		if caOrganization != "" {
			subject.Organization = caOrganization
		}
		if caOrgUnit != "" {
			subject.OrganizationalUnit = caOrgUnit
		}
		if caCountry != "" {
			subject.Country = caCountry
		}
		if caState != "" {
			subject.Province = caState
		}
		if caLocality != "" {
			subject.Locality = caLocality
		}
		if subject.Organization == "" {
			subject.Organization = "cert-helper CA"
		}

		certPath, keyPath, err := internal.GenerateRootCA(outputDir, caName, subject, caValidityDays)
		if err != nil {
			return errors.Wrap(err, "Failed to generate CA")
		}

		fmt.Printf("CA certificate generated successfully: %s\n", certPath)
		fmt.Printf("CA private key generated successfully: %s\n", keyPath)
		return nil

	},
}

func init() {
	Cmd.AddCommand(caGenerateCmd)
	caGenerateCmd.Flags().IntVarP(&caValidityDays, "validity", "v", 3600, "Validity period in days")
	caGenerateCmd.Flags().StringVarP(&caSubject, "subject", "s", "CN=Test CA", "CA subject (e.g. CN=Example CA,O=Org)")
	caGenerateCmd.Flags().StringVar(&caName, "name", "", "Name for storing the CA (defaults to 'default')")
	caGenerateCmd.Flags().StringVar(&caCommonName, "common-name", "", "Common Name (CN)")
	caGenerateCmd.Flags().StringVar(&caOrganization, "organization", "", "Organization (O)")
	caGenerateCmd.Flags().StringVar(&caOrgUnit, "organizational-unit", "", "Organizational Unit (OU)")
	caGenerateCmd.Flags().StringVar(&caCountry, "country", "", "Country (C)")
	caGenerateCmd.Flags().StringVar(&caState, "state", "", "State/Province (ST)")
	caGenerateCmd.Flags().StringVar(&caLocality, "locality", "", "Locality (L)")
}
