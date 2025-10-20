/*
Copyright © 2025 Inácio Moraes da Silva

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of SET CLI",
	Long:  `Display version information for SET CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("SET CLI v%s\n", version)
		fmt.Println("Software Estimation Tool")
		fmt.Println("Copyright © 2025 Inácio Moraes da Silva")
		fmt.Println("Licensed under MIT License")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
