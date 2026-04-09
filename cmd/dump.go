package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/QQGoblin/damonctl/pkg/damon"
)

var dumpConfigOutput string

var DumpConfigCmd = &cobra.Command{
	Use:   "dump-config",
	Short: "Export default configuration as JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := json.MarshalIndent(damon.DefaultStartConfig(), "", "  ")
		if err != nil {
			return fmt.Errorf("dump-config: %w", err)
		}
		data = append(data, '\n')

		if dumpConfigOutput == "" {
			_, err = os.Stdout.Write(data)
			return err
		}

		if err := os.WriteFile(dumpConfigOutput, data, 0o644); err != nil {
			return fmt.Errorf("dump-config: %w", err)
		}
		fmt.Printf("written to %s\n", dumpConfigOutput)
		return nil
	},
}

func init() {
	DumpConfigCmd.Flags().StringVarP(&dumpConfigOutput, "output", "o", "", "output file path (default: stdout)")
}
