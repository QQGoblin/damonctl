package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QQGoblin/damonctl/pkg/damon"
)

var (
	stopID  int
	stopAll bool
)

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a kdamond by slot ID, or all running kdamonds",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !stopAll && stopID < 0 {
			return fmt.Errorf("either --id or --all is required")
		}

		nr, err := damon.ReadNrKdamonds()
		if err != nil {
			return fmt.Errorf("stop: %w", err)
		}

		if stopAll {
			stopped := 0
			for i := 0; i < nr; i++ {
				kd := damon.NewKdamon(i)
				running, err := kd.IsRunning()
				if err != nil {
					return fmt.Errorf("stop: check kdamond %d: %w", i, err)
				}
				if !running {
					continue
				}
				if err := kd.Stop(); err != nil {
					return fmt.Errorf("stop: kdamond %d: %w", i, err)
				}
				fmt.Printf("kdamond %d stopped\n", i)
				stopped++
			}
			if stopped == 0 {
				fmt.Println("no running kdamonds")
			}
			return nil
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
	StopCmd.Flags().IntVar(&stopID, "id", -1, "kdamond slot ID to stop")
	StopCmd.Flags().BoolVar(&stopAll, "all", false, "stop all running kdamonds")

	StopCmd.MarkFlagsMutuallyExclusive("id", "all")
}
