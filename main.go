package main

import (
	"github.com/QQGoblin/damonctl/cmd"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "damonctl",
	Short: "Manage kernel DAMON kdamond instances",
	Long: `damon-ctl is a command-line tool for managing kernel DAMON (Data Access Monitoring
Framework) kdamond instances. It uses a pre-allocated slot strategy to efficiently
manage dynamic monitoring and memory reclaim for multiple processes.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func main() {
	rootCmd.AddCommand(cmd.InitCmd)
	rootCmd.AddCommand(cmd.StartCmd)
	rootCmd.AddCommand(cmd.StopCmd)
	rootCmd.AddCommand(cmd.ShowCmd)
	rootCmd.AddCommand(cmd.DumpConfigCmd)
	rootCmd.AddCommand(cmd.SchemeStateCmd)
	Execute()
}
