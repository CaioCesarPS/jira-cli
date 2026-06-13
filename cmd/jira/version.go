package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/caiocesarps/jira-cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Exibe a versão da CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Info())
	},
}
