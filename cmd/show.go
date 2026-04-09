package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QQGoblin/damonctl/pkg/damon"
)

var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all kdamond slot states",
	RunE: func(cmd *cobra.Command, args []string) error {
		slots, err := damon.ListSlots()
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}
		if len(slots) == 0 {
			fmt.Println("no kdamond slots (run 'init' first)")
			return nil
		}

		fmt.Printf("%-4s  %-8s  %-10s\n", "ID", "STATE", "KTHREAD")
		for _, s := range slots {
			kthread := "-"
			if s.KdamondPid > 0 {
				kthread = fmt.Sprintf("%d", s.KdamondPid)
				fmt.Printf("%-4d  %-8s  %-10s\n", s.ID, s.State, kthread)
			}
		}
		return nil
	},
}
