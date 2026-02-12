package cmd

import (
	"os"
	"path/filepath"

	"github.com/Ctere1/cert-helper/cmd/ca"
	"github.com/Ctere1/cert-helper/cmd/cert"
	"github.com/Ctere1/cert-helper/cmd/scep"
	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

var Version = "0.0.1-dev"

var rootCmd = &cobra.Command{
	Use:     "cert-helper",
	Short:   "Certificate generation helper",
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		output, err := cmd.Flags().GetString("output-dir")
		if err != nil {
			return err
		}
		return os.MkdirAll(output, 0o700)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("output-dir", "o", filepath.Join(xdg.DataHome, "cert-helper"), "Directory to write output files to.")
	rootCmd.AddCommand(ca.Cmd)
	rootCmd.AddCommand(cert.Cmd)
	rootCmd.AddCommand(scep.Cmd)
}
