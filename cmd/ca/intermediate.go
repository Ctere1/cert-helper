package ca

import (
	"fmt"

	"github.com/Ctere1/cert-helper/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	intermediateValidityDays int
	intermediateSubject      string
	intermediateName         string
	intermediateRootName     string
	intermediateCommonName   string
	intermediateOrganization string
	intermediateOrgUnit      string
	intermediateCountry      string
	intermediateState        string
	intermediateLocality     string
)

var intermediateGenerateCmd = &cobra.Command{
	Use:   "intermediate",
	Short: "Generate a new Intermediate Certificate Authority.",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, err := cmd.Flags().GetString("output-dir")
		if err != nil {
			return err
		}

		subject := internal.ParseSubjectString(intermediateSubject)
		if intermediateCommonName != "" {
			subject.CommonName = intermediateCommonName
		}
		if intermediateOrganization != "" {
			subject.Organization = intermediateOrganization
		}
		if intermediateOrgUnit != "" {
			subject.OrganizationalUnit = intermediateOrgUnit
		}
		if intermediateCountry != "" {
			subject.Country = intermediateCountry
		}
		if intermediateState != "" {
			subject.Province = intermediateState
		}
		if intermediateLocality != "" {
			subject.Locality = intermediateLocality
		}
		if intermediateName == "" && subject.CommonName != "" {
			intermediateName = subject.CommonName
		}

		certPath, keyPath, err := internal.GenerateIntermediateCA(outputDir, intermediateRootName, intermediateName, subject, intermediateValidityDays)
		if err != nil {
			return errors.Wrap(err, "Failed to generate intermediate CA")
		}

		fmt.Printf("Intermediate CA certificate generated successfully: %s\n", certPath)
		fmt.Printf("Intermediate CA private key generated successfully: %s\n", keyPath)
		return nil
	},
}

func init() {
	Cmd.AddCommand(intermediateGenerateCmd)
	intermediateGenerateCmd.Flags().IntVarP(&intermediateValidityDays, "validity", "v", 1800, "Validity period in days")
	intermediateGenerateCmd.Flags().StringVarP(&intermediateSubject, "subject", "s", "CN=Intermediate CA", "Intermediate CA subject (e.g. CN=Intermediate,O=Org)")
	intermediateGenerateCmd.Flags().StringVar(&intermediateName, "name", "", "Name for storing the intermediate CA")
	intermediateGenerateCmd.Flags().StringVar(&intermediateRootName, "root", "default", "Root CA name to sign the intermediate CA")
	intermediateGenerateCmd.Flags().StringVar(&intermediateCommonName, "common-name", "", "Common Name (CN)")
	intermediateGenerateCmd.Flags().StringVar(&intermediateOrganization, "organization", "", "Organization (O)")
	intermediateGenerateCmd.Flags().StringVar(&intermediateOrgUnit, "organizational-unit", "", "Organizational Unit (OU)")
	intermediateGenerateCmd.Flags().StringVar(&intermediateCountry, "country", "", "Country (C)")
	intermediateGenerateCmd.Flags().StringVar(&intermediateState, "state", "", "State/Province (ST)")
	intermediateGenerateCmd.Flags().StringVar(&intermediateLocality, "locality", "", "Locality (L)")
}
