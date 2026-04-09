package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QQGoblin/damonctl/pkg/damon"
)

var stopID int

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a kdamond by slot ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		if stopID < 0 {
			return fmt.Errorf("-id is required")
		}

		nr, err := damon.ReadNrKdamonds()
		if err != nil {
			return fmt.Errorf("stop: %w", err)
		}
		if stopID >= nr {
			return fmt.Errorf("stop: id %d out of range (nr_kdamonds=%d)", stopID, nr)
		}

		kd := damon.NewKdamon(stopID)
		if err := kd.Stop(); err != nil {
			return fmt.Errorf("stop: %w", err)
		}
		fmt.Printf("kdamond %d stopped\n", stopID)
		return nil
	},
}

func init() {
	StopCmd.Flags().IntVar(&stopID, "id", -1, "kdamond slot ID to stop (required)")

	_ = StopCmd.MarkFlagRequired("id")
}
