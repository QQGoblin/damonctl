package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QQGoblin/damonctl/pkg/damon"
)

var schemeStateID int
var accessThreshold int

var SchemeStateCmd = &cobra.Command{
	Use:   "sstate",
	Short: "Update and show tried_regions statistics for all schemes of a kdamond",
	Long: `Write "update_schemes_tried_regions" to the kdamond state file, then read
all tried_regions for every scheme under the specified kdamond slot and print
per-region details along with summary statistics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if schemeStateID < 0 {
			return fmt.Errorf("--id is required")
		}

		nr, err := damon.ReadNrKdamonds()
		if err != nil {
			return fmt.Errorf("scheme-state: %w", err)
		}
		if schemeStateID >= nr {
			return fmt.Errorf("scheme-state: id %d out of range (nr_kdamonds=%d)", schemeStateID, nr)
		}

		kd := damon.NewKdamon(schemeStateID)

		running, err := kd.IsRunning()
		if err != nil {
			return fmt.Errorf("scheme-state: check state: %w", err)
		}
		if !running {
			return fmt.Errorf("scheme-state: kdamond %d is not running", schemeStateID)
		}

		if err := kd.UpdateSchemesTried(); err != nil {
			return fmt.Errorf("scheme-state: update tried_regions: %w", err)
		}

		allSchemes, err := kd.ReadTriedRegions()
		if err != nil {
			return fmt.Errorf("scheme-state: read tried_regions: %w", err)
		}

		if len(allSchemes) == 0 {
			fmt.Println("no schemes configured")
			return nil
		}

		for _, st := range allSchemes {

			fmt.Printf("\n" + formatTitle(fmt.Sprintf(" Scheme %d ", st.SchemeID), 90, "=") + "\n")
			if len(st.Regions) == 0 {
				fmt.Println("  (no tried regions)")
				continue
			}

			fmt.Printf("  %-4s  %-18s  %-18s  %-12s  %-12s  %s\n",
				"#", "START", "END", "NR_ACCESSES", "AGE", "SIZE")
			type bucket struct {
				count int
				size  uint64
				age   int
			}
			var cold, warm, hot bucket // cold: ==0, warm: (0,N), hot: >=N

			for i, r := range st.Regions {
				sz := r.End - r.Start
				fmt.Printf("  %-4d  0x%016x  0x%016x  %-12d  %-12d  %s\n",
					i, r.Start, r.End, r.NrAccesses, r.Age, formatBytes(sz))
				b := &cold
				switch {
				case r.NrAccesses == 0:
					b = &cold
				case r.NrAccesses < accessThreshold:
					b = &warm
				default:
					b = &hot
				}
				b.count++
				b.size += sz
				b.age += r.Age
			}

			avgAge := func(bk bucket) string {
				if bk.count == 0 {
					return "-"
				}
				return fmt.Sprintf("%.1f", float64(bk.age)/float64(bk.count))
			}
			fmt.Println("\n" + formatTitle(" Summary ", 90, "-") + "\n")
			fmt.Printf("%-12s  %-8s  %-14s  %s\n", "ACCESSES", "REGIONS", "TOTAL_SIZE", "AVG_AGE")
			fmt.Printf("%-12s  %-8d  %-14s  %s\n", "[0, 0]", cold.count, formatBytes(cold.size), avgAge(cold))
			fmt.Printf("%-12s  %-8d  %-14s  %s\n", fmt.Sprintf("(0, %d)", accessThreshold), warm.count, formatBytes(warm.size), avgAge(warm))
			fmt.Printf("%-12s  %-8d  %-14s  %s\n", fmt.Sprintf("[%d, ∞)", accessThreshold), hot.count, formatBytes(hot.size), avgAge(hot))
			fmt.Println()
		}
		return nil
	},
}

func formatBytes(b uint64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatTitle(title string, width int, split string) string {
	pad := width - len(title)
	left := pad / 2
	right := pad - left
	return strings.Repeat(split, left) + title + strings.Repeat(split, right)
}

func init() {
	SchemeStateCmd.Flags().IntVar(&schemeStateID, "id", -1, "kdamond slot ID (required)")
	SchemeStateCmd.Flags().IntVar(&accessThreshold, "access-threshold", 10, "access count threshold N for bucketed statistics")
	_ = SchemeStateCmd.MarkFlagRequired("id")
}
