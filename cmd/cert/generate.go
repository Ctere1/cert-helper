package cert

import (
	"fmt"

	"github.com/Ctere1/cert-helper/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var certGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new Certificate.",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, err := cmd.Flags().GetString("output-dir")
		if err != nil {
			return err
		}

		cn, _ := cmd.Flags().GetString("common-name")
		org, _ := cmd.Flags().GetString("organization")
		orgUnit, _ := cmd.Flags().GetString("organizational-unit")
		country, _ := cmd.Flags().GetString("country")
		state, _ := cmd.Flags().GetString("state")
		locality, _ := cmd.Flags().GetString("locality")
		issuerType, _ := cmd.Flags().GetString("issuer-type")
		issuerName, _ := cmd.Flags().GetString("issuer-name")
		issuerRoot, _ := cmd.Flags().GetString("issuer-root")
		subjectAltNames, _ := cmd.Flags().GetStringSlice("subject-alt-names")
		pfxPassword, _ := cmd.Flags().GetString("pfx-password")
		validityDays, _ := cmd.Flags().GetInt("validity-days")

		subject := internal.Subject{
			CommonName:         cn,
			Organization:       org,
			OrganizationalUnit: orgUnit,
			Country:            country,
			Province:           state,
			Locality:           locality,
		}
		if subject.CommonName == "" && len(args) > 0 {
			subject.CommonName = args[0]
		}
		if subject.CommonName == "" {
			return errors.New("common name is required")
		}

		if issuerType == "intermediate" && issuerRoot == "" {
			issuerRoot = "default"
		}

		certPath, keyPath, pfxPath, err := internal.GenerateCertificate(outputDir, issuerType, issuerRoot, issuerName, subject, subjectAltNames, validityDays, pfxPassword)
		if err != nil {
			return errors.Wrap(err, "Failed to generate certificate")
		}

		fmt.Printf("Certificate generated successfully: %s\n", certPath)
		fmt.Printf("Certificate private key: %s\n", keyPath)
		fmt.Printf("Certificate PFX bundle: %s\n", pfxPath)
		return nil
	},
}

func init() {
	Cmd.AddCommand(certGenerateCmd)
	certGenerateCmd.Flags().StringSlice("subject-alt-names", []string{}, "Subject Alternative Names")
	certGenerateCmd.Flags().String("pfx-password", "", "Password for PFX file")
	certGenerateCmd.Flags().IntP("validity-days", "v", 365, "Validity period in days")
	certGenerateCmd.Flags().String("common-name", "", "Common Name (CN)")
	certGenerateCmd.Flags().String("organization", "", "Organization (O)")
	certGenerateCmd.Flags().String("organizational-unit", "", "Organizational Unit (OU)")
	certGenerateCmd.Flags().String("country", "", "Country (C)")
	certGenerateCmd.Flags().String("state", "", "State/Province (ST)")
	certGenerateCmd.Flags().String("locality", "", "Locality (L)")
	certGenerateCmd.Flags().String("issuer-type", "root", "Issuer type: root or intermediate")
	certGenerateCmd.Flags().String("issuer-name", "default", "Issuer name (root CA name or intermediate CA name)")
	certGenerateCmd.Flags().String("issuer-root", "default", "Root CA name when issuer type is intermediate")
}
