package cmd

import (
	"fmt"

	"github.com/cleonte/k8slings/k8slings/exercises"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func ListCmd(infoFile string) *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List all exercises, by topic, with their state",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, err := exercises.List(infoFile)
			if err != nil {
				return err
			}
			topic, done := "", 0
			for _, ex := range all {
				if ex.Topic != topic {
					topic = ex.Topic
					color.Cyan("\n%s", topic)
				}
				mark := "  "
				if ex.State() == exercises.Done {
					mark = color.GreenString("✓ ")
					done++
				}
				fmt.Printf("  %s%-24s %s\n", mark, ex.Name, ex.Mode)
			}
			fmt.Printf("\n%d of %d exercises done\n", done, len(all))
			return nil
		},
	}
}
