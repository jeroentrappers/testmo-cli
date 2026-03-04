package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/secutec/testmo-cli/internal/api"
	"github.com/spf13/cobra"
)

var caseProjectID int

func init() {
	rootCmd.AddCommand(casesCmd)
	casesCmd.PersistentFlags().IntVarP(&caseProjectID, "project", "p", 0, "Project ID (required)")
	casesCmd.MarkPersistentFlagRequired("project")

	casesCmd.AddCommand(casesListCmd)
	casesCmd.AddCommand(casesCreateCmd)
	casesCmd.AddCommand(casesUpdateCmd)
	casesCmd.AddCommand(casesDeleteCmd)

	casesListCmd.Flags().Int("folder-id", 0, "Filter by folder ID")

	casesCreateCmd.Flags().String("name", "", "Test case name (required)")
	casesCreateCmd.Flags().Int("folder-id", 0, "Folder ID")
	casesCreateCmd.Flags().Int("template-id", 0, "Template ID")
	casesCreateCmd.Flags().Int("state-id", 0, "State ID")
	casesCreateCmd.MarkFlagRequired("name")

	casesUpdateCmd.Flags().String("ids", "", "Comma-separated case IDs to update (required)")
	casesUpdateCmd.Flags().String("name", "", "New name")
	casesUpdateCmd.Flags().Int("folder-id", 0, "Move to folder ID")
	casesUpdateCmd.Flags().Int("state-id", 0, "New state ID")
	casesUpdateCmd.MarkFlagRequired("ids")

	casesDeleteCmd.Flags().String("ids", "", "Comma-separated case IDs to delete (required)")
	casesDeleteCmd.MarkFlagRequired("ids")
}

var casesCmd = &cobra.Command{
	Use:   "cases",
	Short: "Manage test cases",
}

var casesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List test cases",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		var folderID *int
		if cmd.Flags().Changed("folder-id") {
			v, _ := cmd.Flags().GetInt("folder-id")
			folderID = &v
		}

		cases, err := client.ListCases(caseProjectID, folderID)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tKEY\tFOLDER\tNAME\tSTATE\tAUTOMATION")
		for _, c := range cases {
			fmt.Fprintf(w, "%d\tC-%d\t%d\t%s\t%d\t%v\n",
				c.ID, c.Key, c.FolderID, c.Name, c.StateID, c.HasAutomation)
		}
		fmt.Fprintf(os.Stderr, "\nTotal: %d cases\n", len(cases))
		return w.Flush()
	},
}

var casesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a test case",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		name, _ := cmd.Flags().GetString("name")
		tc := api.CreateCase{Name: name}

		if cmd.Flags().Changed("folder-id") {
			v, _ := cmd.Flags().GetInt("folder-id")
			tc.FolderID = &v
		}
		if cmd.Flags().Changed("template-id") {
			v, _ := cmd.Flags().GetInt("template-id")
			tc.TemplateID = &v
		}
		if cmd.Flags().Changed("state-id") {
			v, _ := cmd.Flags().GetInt("state-id")
			tc.StateID = &v
		}

		created, err := client.CreateCases(caseProjectID, []api.CreateCase{tc})
		if err != nil {
			return err
		}
		for _, c := range created {
			fmt.Printf("Created case: ID=%d Key=C-%d Name=%q\n", c.ID, c.Key, c.Name)
		}
		return nil
	},
}

var casesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update test cases",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		idsStr, _ := cmd.Flags().GetString("ids")
		ids, err := parseIntList(idsStr)
		if err != nil {
			return fmt.Errorf("invalid IDs: %w", err)
		}

		req := api.UpdateCaseRequest{IDs: ids}

		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			req.Name = &v
		}
		if cmd.Flags().Changed("folder-id") {
			v, _ := cmd.Flags().GetInt("folder-id")
			req.FolderID = &v
		}
		if cmd.Flags().Changed("state-id") {
			v, _ := cmd.Flags().GetInt("state-id")
			req.StateID = &v
		}

		if err := client.UpdateCases(caseProjectID, req); err != nil {
			return err
		}
		fmt.Printf("Updated %d case(s)\n", len(ids))
		return nil
	},
}

var casesDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete test cases",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		idsStr, _ := cmd.Flags().GetString("ids")
		ids, err := parseIntList(idsStr)
		if err != nil {
			return fmt.Errorf("invalid IDs: %w", err)
		}

		if err := client.DeleteCases(caseProjectID, ids); err != nil {
			return err
		}
		fmt.Printf("Deleted %d case(s)\n", len(ids))
		return nil
	},
}
