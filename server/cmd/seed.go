package cmd

import (
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with data",
	Long:  "Commands for seeding the database with demo or test data.",
}

func init() {
	rootCmd.AddCommand(seedCmd)
}
