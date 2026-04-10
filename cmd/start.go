package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QQGoblin/damonctl/pkg/damon"
)

var (
	startPid    int
	startConfig string
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start monitoring a process, returns slot ID",
	Long: `Start DAMON monitoring for a target process. Finds a free kdamond slot,
configures context, monitoring attributes, target, and DAMOS scheme, then
turns on the kdamond. Prints the allocated slot ID on success.

Use -config to specify a JSON configuration file. When omitted, built-in
defaults are used. The JSON file may contain any subset of fields; unspecified
fields keep their default values.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg := damon.DefaultStartConfig()
		if startConfig != "" {
			var err error
			cfg, err = damon.LoadStartConfig(startConfig)
			if err != nil {
				return fmt.Errorf("start: %w", err)
			}
		}

		if cfg.Ops == "vaddr" && startPid <= 0 {
			return fmt.Errorf("-pid is required")
		}

		slotID, err := damon.FindFreeSlot()
		if err != nil {
			return fmt.Errorf("start: %w", err)
		}

		kd := damon.NewKdamon(slotID)
		if err := kd.Start(startPid, cfg); err != nil {
			return fmt.Errorf("start: %w", err)
		}

		fmt.Println(slotID)
		return nil
	},
}

func init() {
	StartCmd.Flags().IntVar(&startPid, "pid", 0, "target process PID (required)")
	StartCmd.Flags().StringVar(&startConfig, "config", "", "path to JSON configuration file")
}
