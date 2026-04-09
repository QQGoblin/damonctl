package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QQGoblin/damonctl/pkg/damon"
)

var initSlots int

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Pre-allocate kdamond slots",
	Long: `Pre-allocate a fixed number of kdamond sysfs directory slots.
This must be run before any start/stop operations. Requires root privileges.
WARNING: This destroys all existing kdamond configurations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if initSlots <= 0 {
			return fmt.Errorf("number of slots must be positive")
		}

		if err := damon.Init(initSlots); err != nil {
			return fmt.Errorf("init: %w", err)
		}
		fmt.Printf("initialized %d kdamond slots\n", initSlots)
		return nil
	},
}

func init() {
	InitCmd.Flags().IntVarP(&initSlots, "number", "n", 64, "number of kdamond slots to pre-allocate")
}
