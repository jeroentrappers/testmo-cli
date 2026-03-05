package cmd

import (
	"fmt"

	syncpkg "github.com/secutec/testmo-cli/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncProjectID int
	syncFolderID  int
	syncFile      string
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.PersistentFlags().IntVarP(&syncProjectID, "project", "p", 0, "Project ID (required)")
	syncCmd.PersistentFlags().IntVar(&syncFolderID, "folder", 0, "Folder ID to scope sync to a subtree")
	syncCmd.PersistentFlags().StringVarP(&syncFile, "file", "f", "testmo.yaml", "YAML sync file path")
	syncCmd.MarkPersistentFlagRequired("project")

	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncDiffCmd)

	syncPushCmd.Flags().Bool("delete", false, "Delete cases/folders in Testmo that are not in the YAML file")
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync test cases between YAML file and Testmo",
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull all test cases from Testmo into a YAML file",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		folderID := resolveFolderID()
		if folderID != nil {
			fmt.Printf("Pulling from project %d, folder %d...\n", syncProjectID, *folderID)
		} else {
			fmt.Printf("Pulling from project %d...\n", syncProjectID)
		}
		yamlFile, err := syncpkg.PullToYAML(client, syncProjectID, folderID)
		if err != nil {
			return err
		}

		if err := syncpkg.SaveYAML(syncFile, yamlFile); err != nil {
			return err
		}

		// Count totals
		totalFolders, totalCases := countYAML(yamlFile)
		fmt.Printf("Saved %d folders and %d cases to %s\n", totalFolders, totalCases, syncFile)
		return nil
	},
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push YAML file changes to Testmo",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()
		deleteOrphans, _ := cmd.Flags().GetBool("delete")

		local, err := syncpkg.LoadYAML(syncFile)
		if err != nil {
			return err
		}

		if local.Project != 0 && local.Project != syncProjectID {
			return fmt.Errorf("project ID mismatch: --project=%d but %s specifies project: %d", syncProjectID, syncFile, local.Project)
		}

		folderID := resolveFolderID()
		if folderID == nil && local.Folder != 0 {
			folderID = &local.Folder
		}
		if folderID != nil && local.Folder != 0 && *folderID != local.Folder {
			return fmt.Errorf("folder ID mismatch: --folder=%d but %s specifies folder: %d", *folderID, syncFile, local.Folder)
		}

		fmt.Printf("Computing diff for project %d...\n", syncProjectID)
		diff, err := syncpkg.ComputeDiff(client, syncProjectID, folderID, local)
		if err != nil {
			return err
		}

		syncpkg.PrintDiff(diff)

		if err := syncpkg.ApplyDiff(client, syncProjectID, diff, deleteOrphans); err != nil {
			return err
		}

		fmt.Println("Push complete.")
		return nil
	},
}

var syncDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show what would change without applying (dry run)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := mustClient()

		local, err := syncpkg.LoadYAML(syncFile)
		if err != nil {
			return err
		}

		if local.Project != 0 && local.Project != syncProjectID {
			return fmt.Errorf("project ID mismatch: --project=%d but %s specifies project: %d", syncProjectID, syncFile, local.Project)
		}

		folderID := resolveFolderID()
		if folderID == nil && local.Folder != 0 {
			folderID = &local.Folder
		}
		if folderID != nil && local.Folder != 0 && *folderID != local.Folder {
			return fmt.Errorf("folder ID mismatch: --folder=%d but %s specifies folder: %d", *folderID, syncFile, local.Folder)
		}

		fmt.Printf("Computing diff for project %d...\n", syncProjectID)
		diff, err := syncpkg.ComputeDiff(client, syncProjectID, folderID, local)
		if err != nil {
			return err
		}

		syncpkg.PrintDiff(diff)
		return nil
	},
}

func resolveFolderID() *int {
	if syncFolderID != 0 {
		return &syncFolderID
	}
	return nil
}

func countYAML(f *syncpkg.YAMLFile) (folders, cases int) {
	var count func(yf syncpkg.YAMLFolder)
	count = func(yf syncpkg.YAMLFolder) {
		folders++
		cases += len(yf.Cases)
		for _, sub := range yf.Folders {
			count(sub)
		}
	}
	for _, root := range f.Folders {
		count(root)
	}
	return
}
