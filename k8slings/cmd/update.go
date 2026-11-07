package cmd

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/cleonte/k8slings"
	"github.com/cleonte/k8slings/k8slings/workspace"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// UpdateCmd brings an existing workspace up to date with the exercises of this
// version of k8slings.
func UpdateCmd(src fs.FS, root string) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Add the exercises of this version that your folder does not have yet",
		Long: `A new version of k8slings can bring new exercises, and your exercises folder, made
by an older version, does not know about them. update adds them to the folder and to
info.toml, each one where it belongs in the order of the course.

It never overwrites an exercise file you may have worked on. It keeps a copy of
info.toml as info.toml.bak. READMEs are replaced by their newer text, and so is an
exercise file that you have not changed since k8slings wrote it. An exercise file
that has a newer version but that you have edited is listed: 'k8slings reset <name>'
brings the new one, and loses your changes.

Use --dry-run to see what it would do.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := workspace.Update(src, root, dryRun)
			if err != nil {
				color.Red("Cannot update: %v (run it from the folder with info.toml)", err)
				return err
			}
			printUpdate(res, dryRun)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "only say what would be added or replaced")
	return cmd
}

func printUpdate(res workspace.UpdateResult, dryRun bool) {
	verb := "Added"
	if dryRun {
		verb = "Would add"
	}
	if len(res.Added) == 0 && res.Readmes == 0 && len(res.Refreshed) == 0 && len(res.Newer) == 0 {
		color.Green("Your exercises are up to date with this version of k8slings.")
		return
	}
	if len(res.Added) > 0 {
		topic := ""
		color.Green("%s %d new exercises:", verb, len(res.Added))
		for _, ex := range res.Added {
			if ex.Topic != topic {
				topic = ex.Topic
				color.Cyan("\n  %s", topic)
			}
			fmt.Printf("    %-24s %s\n", ex.Name, ex.Mode)
		}
		fmt.Println()
		if !dryRun {
			color.White("They are in the order of the course; `k8slings list` marks them \"new\". Your earlier info.toml is info.toml.bak.")
		}
	}
	if res.Readmes > 0 {
		fmt.Printf("%s %d README files with their newer text.\n", map[bool]string{false: "Replaced", true: "Would replace"}[dryRun], res.Readmes)
	}
	if len(res.Refreshed) > 0 {
		fmt.Printf("%s the file of %d exercises you had not started: %s\n", map[bool]string{false: "Replaced", true: "Would replace"}[dryRun], len(res.Refreshed), workspace.Describe(res.Refreshed))
	}
	if len(res.Newer) > 0 {
		color.Yellow("%d exercises have a newer file that was not applied, because you changed the old one: %s", len(res.Newer), strings.Join(res.Newer, ", "))
		color.Yellow("`k8slings reset <name>` brings the new one, and loses your changes.")
	}
}

// newExercisesLine is the one line shown by run, list, watch and doctor when the
// workspace is behind this version of k8slings. It is empty when it is not.
func newExercisesLine(src fs.FS, root string) string {
	missing := workspace.Missing(src, root)
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf("This version of k8slings has %d exercises that your folder does not have yet (%s): run `k8slings update`.", len(missing), workspace.Describe(missing))
}

// noticeNewExercises prints the line above, in yellow, when there is one.
func noticeNewExercises() {
	if line := newExercisesLine(k8slings.FS, "."); line != "" {
		color.Yellow(line)
	}
}
