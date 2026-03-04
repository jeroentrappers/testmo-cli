package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		projects, err := client.ListProjects()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCOMPLETED\tRUNS\tAUTOMATION RUNS")
		for _, p := range projects {
			fmt.Fprintf(w, "%d\t%s\t%v\t%d\t%d\n",
				p.ID, p.Name, p.IsCompleted, p.RunCount, p.AutomationRunCount)
		}
		return w.Flush()
	},
}
