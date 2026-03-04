package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/secutec/testmo-cli/internal/api"
	"github.com/spf13/cobra"
)

var folderProjectID int

func init() {
	rootCmd.AddCommand(foldersCmd)
	foldersCmd.PersistentFlags().IntVarP(&folderProjectID, "project", "p", 0, "Project ID (required)")
	foldersCmd.MarkPersistentFlagRequired("project")

	foldersCmd.AddCommand(foldersListCmd)
	foldersCmd.AddCommand(foldersCreateCmd)
	foldersCmd.AddCommand(foldersUpdateCmd)
	foldersCmd.AddCommand(foldersDeleteCmd)

	foldersCreateCmd.Flags().String("name", "", "Folder name (required)")
	foldersCreateCmd.Flags().Int("parent-id", 0, "Parent folder ID")
	foldersCreateCmd.Flags().String("docs", "", "Folder description")
	foldersCreateCmd.MarkFlagRequired("name")

	foldersUpdateCmd.Flags().Int("id", 0, "Folder ID to update (required)")
	foldersUpdateCmd.Flags().String("name", "", "New folder name")
	foldersUpdateCmd.Flags().String("docs", "", "New description")
	foldersUpdateCmd.MarkFlagRequired("id")

	foldersDeleteCmd.Flags().String("ids", "", "Comma-separated folder IDs to delete (required)")
	foldersDeleteCmd.MarkFlagRequired("ids")
}

var foldersCmd = &cobra.Command{
	Use:   "folders",
	Short: "Manage test case folders",
}

var foldersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List folders (tree view)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		folders, err := client.ListFolders(folderProjectID)
		if err != nil {
			return err
		}

		printFolderTree(folders)
		return nil
	},
}

var foldersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a folder",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		name, _ := cmd.Flags().GetString("name")
		parentID, _ := cmd.Flags().GetInt("parent-id")
		docs, _ := cmd.Flags().GetString("docs")

		f := api.CreateFolder{Name: name}
		if parentID > 0 {
			f.ParentID = &parentID
		}
		if docs != "" {
			f.Docs = &docs
		}

		created, err := client.CreateFolders(folderProjectID, []api.CreateFolder{f})
		if err != nil {
			return err
		}

		for _, folder := range created {
			fmt.Printf("Created folder: ID=%d Name=%q\n", folder.ID, folder.Name)
		}
		return nil
	},
}

var foldersUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a folder",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		id, _ := cmd.Flags().GetInt("id")
		req := api.UpdateFolderRequest{IDs: []int{id}}

		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			req.Name = &name
		}
		if cmd.Flags().Changed("docs") {
			docs, _ := cmd.Flags().GetString("docs")
			req.Docs = &docs
		}

		if err := client.UpdateFolders(folderProjectID, req); err != nil {
			return err
		}
		fmt.Printf("Updated folder %d\n", id)
		return nil
	},
}

var foldersDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete folders",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		idsStr, _ := cmd.Flags().GetString("ids")
		ids, err := parseIntList(idsStr)
		if err != nil {
			return fmt.Errorf("invalid IDs: %w", err)
		}

		if err := client.DeleteFolders(folderProjectID, ids); err != nil {
			return err
		}
		fmt.Printf("Deleted %d folder(s)\n", len(ids))
		return nil
	},
}

func printFolderTree(folders []api.Folder) {
	// Build parent->children map
	children := make(map[int][]api.Folder) // parentID -> children
	var roots []api.Folder

	for _, f := range folders {
		if f.ParentID == nil {
			roots = append(roots, f)
		} else {
			children[*f.ParentID] = append(children[*f.ParentID], f)
		}
	}

	// Sort roots by display_order
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].DisplayOrder < roots[j].DisplayOrder
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tDOCS")

	var printTree func(folder api.Folder, indent string)
	printTree = func(folder api.Folder, indent string) {
		docs := ""
		if folder.Docs != nil {
			docs = truncate(*folder.Docs, 60)
		}
		fmt.Fprintf(w, "%d\t%s%s\t%s\n", folder.ID, indent, folder.Name, docs)

		kids := children[folder.ID]
		sort.Slice(kids, func(i, j int) bool {
			return kids[i].DisplayOrder < kids[j].DisplayOrder
		})
		for _, child := range kids {
			printTree(child, indent+"  ")
		}
	}

	for _, root := range roots {
		printTree(root, "")
	}
	w.Flush()
}

func truncate(s string, maxLen int) string {
	// Strip HTML tags for display
	s = stripHTML(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func parseIntList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	var ids []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %w", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
