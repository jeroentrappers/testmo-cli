package sync

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/secutec/testmo-cli/internal/api"
	"gopkg.in/yaml.v3"
)

// YAMLFile represents the top-level sync file.
type YAMLFile struct {
	Project int          `yaml:"project"`
	Folders []YAMLFolder `yaml:"folders"`
}

// YAMLFolder represents a folder with optional nested folders and cases.
type YAMLFolder struct {
	Name    string       `yaml:"name"`
	Docs    string       `yaml:"docs,omitempty"`
	Cases   []YAMLCase   `yaml:"cases,omitempty"`
	Folders []YAMLFolder `yaml:"folders,omitempty"`
}

// YAMLCase represents a test case in the YAML file.
type YAMLCase struct {
	Name        string `yaml:"name"`
	Priority    *int   `yaml:"priority,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// DiffResult contains the changes needed to sync local YAML to Testmo.
type DiffResult struct {
	FoldersToCreate []FolderCreate
	FoldersToUpdate []FolderUpdate
	FoldersToDelete []int
	CasesToCreate   []CaseCreate
	CasesToUpdate   []CaseUpdate
	CasesToDelete   []int
}

type FolderCreate struct {
	Name     string
	ParentID *int
	Docs     string
}

type FolderUpdate struct {
	ID   int
	Name string
	Docs string
}

type CaseCreate struct {
	Name     string
	FolderID int
}

type CaseUpdate struct {
	ID   int
	Name string
}

// LoadYAML reads and parses a YAML sync file.
func LoadYAML(path string) (*YAMLFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read YAML file: %w", err)
	}
	var f YAMLFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return &f, nil
}

// SaveYAML writes a YAML sync file.
func SaveYAML(path string, f *YAMLFile) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// PullToYAML fetches all folders and cases from Testmo and builds a YAML file.
func PullToYAML(client *api.Client, projectID int) (*YAMLFile, error) {
	folders, err := client.ListFolders(projectID)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}

	cases, err := client.ListCases(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}

	// Group cases by folder
	casesByFolder := make(map[int][]api.Case)
	for _, c := range cases {
		casesByFolder[c.FolderID] = append(casesByFolder[c.FolderID], c)
	}

	// Sort cases within each folder by name
	for fid := range casesByFolder {
		sort.Slice(casesByFolder[fid], func(i, j int) bool {
			return casesByFolder[fid][i].Name < casesByFolder[fid][j].Name
		})
	}

	// Build folder tree
	childFolders := make(map[int][]api.Folder) // parentID -> children
	var roots []api.Folder
	for _, f := range folders {
		if f.ParentID == nil {
			roots = append(roots, f)
		} else {
			childFolders[*f.ParentID] = append(childFolders[*f.ParentID], f)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].DisplayOrder < roots[j].DisplayOrder
	})

	var buildFolder func(f api.Folder) YAMLFolder
	buildFolder = func(f api.Folder) YAMLFolder {
		yf := YAMLFolder{Name: f.Name}
		if f.Docs != nil {
			yf.Docs = stripHTML(*f.Docs)
		}

		// Add cases
		for _, c := range casesByFolder[f.ID] {
			yc := YAMLCase{Name: c.Name}
			if c.CustomPriority != nil {
				yc.Priority = c.CustomPriority
			}
			if c.CustomDescription != nil {
				yc.Description = stripHTML(*c.CustomDescription)
			}
			yf.Cases = append(yf.Cases, yc)
		}

		// Add child folders
		kids := childFolders[f.ID]
		sort.Slice(kids, func(i, j int) bool {
			return kids[i].DisplayOrder < kids[j].DisplayOrder
		})
		for _, child := range kids {
			yf.Folders = append(yf.Folders, buildFolder(child))
		}

		return yf
	}

	yamlFile := &YAMLFile{Project: projectID}
	for _, root := range roots {
		yamlFile.Folders = append(yamlFile.Folders, buildFolder(root))
	}

	return yamlFile, nil
}

// ComputeDiff compares local YAML against Testmo state and returns what needs to change.
func ComputeDiff(client *api.Client, projectID int, local *YAMLFile) (*DiffResult, error) {
	remoteFolders, err := client.ListFolders(projectID)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}

	remoteCases, err := client.ListCases(projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}

	diff := &DiffResult{}

	// Build lookup maps for remote state
	// Map: parentID + "/" + name -> folder
	type folderKey struct {
		parentID int // 0 means root
		name     string
	}
	remoteFolderMap := make(map[folderKey]*api.Folder)
	for i := range remoteFolders {
		f := &remoteFolders[i]
		pid := 0
		if f.ParentID != nil {
			pid = *f.ParentID
		}
		remoteFolderMap[folderKey{pid, f.Name}] = f
	}

	// Map: folderID + "/" + name -> case
	type caseKey struct {
		folderID int
		name     string
	}
	remoteCaseMap := make(map[caseKey]*api.Case)
	for i := range remoteCases {
		c := &remoteCases[i]
		remoteCaseMap[caseKey{c.FolderID, c.Name}] = c
	}

	// Track which remote items are matched
	matchedFolders := make(map[int]bool)
	matchedCases := make(map[int]bool)

	// Walk local YAML tree and compare
	var walkFolder func(yf YAMLFolder, parentID int)
	walkFolder = func(yf YAMLFolder, parentID int) {
		key := folderKey{parentID, yf.Name}
		remoteFolder := remoteFolderMap[key]

		var folderID int
		if remoteFolder != nil {
			folderID = remoteFolder.ID
			matchedFolders[folderID] = true

			// Check if folder needs update
			remoteDocs := ""
			if remoteFolder.Docs != nil {
				remoteDocs = stripHTML(*remoteFolder.Docs)
			}
			if yf.Docs != remoteDocs {
				diff.FoldersToUpdate = append(diff.FoldersToUpdate, FolderUpdate{
					ID:   folderID,
					Name: yf.Name,
					Docs: yf.Docs,
				})
			}
		} else {
			// Folder needs to be created
			var pid *int
			if parentID > 0 {
				pid = &parentID
			}
			diff.FoldersToCreate = append(diff.FoldersToCreate, FolderCreate{
				Name:     yf.Name,
				ParentID: pid,
				Docs:     yf.Docs,
			})
			folderID = -1 // Will be resolved after creation
		}

		// Compare cases in this folder
		if folderID > 0 {
			for _, yc := range yf.Cases {
				ck := caseKey{folderID, yc.Name}
				remoteCase := remoteCaseMap[ck]
				if remoteCase != nil {
					matchedCases[remoteCase.ID] = true
					// Case exists - could check for updates here
				} else {
					diff.CasesToCreate = append(diff.CasesToCreate, CaseCreate{
						Name:     yc.Name,
						FolderID: folderID,
					})
				}
			}
		}

		// Recurse into subfolders
		for _, subFolder := range yf.Folders {
			if folderID > 0 {
				walkFolder(subFolder, folderID)
			}
		}
	}

	for _, rootFolder := range local.Folders {
		walkFolder(rootFolder, 0)
	}

	// Find unmatched remote items (candidates for deletion)
	for _, f := range remoteFolders {
		if !matchedFolders[f.ID] {
			diff.FoldersToDelete = append(diff.FoldersToDelete, f.ID)
		}
	}
	for _, c := range remoteCases {
		if !matchedCases[c.ID] {
			diff.CasesToDelete = append(diff.CasesToDelete, c.ID)
		}
	}

	return diff, nil
}

// ApplyDiff applies the computed diff to Testmo.
func ApplyDiff(client *api.Client, projectID int, diff *DiffResult, deleteOrphans bool) error {
	// 1. Create folders
	if len(diff.FoldersToCreate) > 0 {
		var toCreate []api.CreateFolder
		for _, f := range diff.FoldersToCreate {
			cf := api.CreateFolder{Name: f.Name, ParentID: f.ParentID}
			if f.Docs != "" {
				cf.Docs = &f.Docs
			}
			toCreate = append(toCreate, cf)
		}
		created, err := client.CreateFolders(projectID, toCreate)
		if err != nil {
			return fmt.Errorf("create folders: %w", err)
		}
		fmt.Printf("Created %d folder(s)\n", len(created))
	}

	// 2. Update folders
	for _, f := range diff.FoldersToUpdate {
		req := api.UpdateFolderRequest{
			IDs:  []int{f.ID},
			Docs: &f.Docs,
		}
		if err := client.UpdateFolders(projectID, req); err != nil {
			return fmt.Errorf("update folder %d: %w", f.ID, err)
		}
	}
	if len(diff.FoldersToUpdate) > 0 {
		fmt.Printf("Updated %d folder(s)\n", len(diff.FoldersToUpdate))
	}

	// 3. Create cases
	if len(diff.CasesToCreate) > 0 {
		var toCreate []api.CreateCase
		for _, c := range diff.CasesToCreate {
			fid := c.FolderID
			toCreate = append(toCreate, api.CreateCase{
				Name:     c.Name,
				FolderID: &fid,
			})
		}
		created, err := client.CreateCases(projectID, toCreate)
		if err != nil {
			return fmt.Errorf("create cases: %w", err)
		}
		fmt.Printf("Created %d case(s)\n", len(created))
	}

	// 4. Delete orphans (only with --delete flag)
	if deleteOrphans {
		if len(diff.CasesToDelete) > 0 {
			if err := client.DeleteCases(projectID, diff.CasesToDelete); err != nil {
				return fmt.Errorf("delete cases: %w", err)
			}
			fmt.Printf("Deleted %d case(s)\n", len(diff.CasesToDelete))
		}
		if len(diff.FoldersToDelete) > 0 {
			if err := client.DeleteFolders(projectID, diff.FoldersToDelete); err != nil {
				return fmt.Errorf("delete folders: %w", err)
			}
			fmt.Printf("Deleted %d folder(s)\n", len(diff.FoldersToDelete))
		}
	}

	return nil
}

// PrintDiff prints a human-readable summary of changes.
func PrintDiff(diff *DiffResult) {
	if len(diff.FoldersToCreate) == 0 && len(diff.FoldersToUpdate) == 0 &&
		len(diff.FoldersToDelete) == 0 && len(diff.CasesToCreate) == 0 &&
		len(diff.CasesToDelete) == 0 && len(diff.CasesToUpdate) == 0 {
		fmt.Println("No changes detected.")
		return
	}

	if len(diff.FoldersToCreate) > 0 {
		fmt.Printf("\nFolders to CREATE (%d):\n", len(diff.FoldersToCreate))
		for _, f := range diff.FoldersToCreate {
			fmt.Printf("  + %s\n", f.Name)
		}
	}

	if len(diff.FoldersToUpdate) > 0 {
		fmt.Printf("\nFolders to UPDATE (%d):\n", len(diff.FoldersToUpdate))
		for _, f := range diff.FoldersToUpdate {
			fmt.Printf("  ~ %s (ID: %d)\n", f.Name, f.ID)
		}
	}

	if len(diff.FoldersToDelete) > 0 {
		fmt.Printf("\nFolders to DELETE (%d):\n", len(diff.FoldersToDelete))
		for _, id := range diff.FoldersToDelete {
			fmt.Printf("  - ID: %d\n", id)
		}
	}

	if len(diff.CasesToCreate) > 0 {
		fmt.Printf("\nCases to CREATE (%d):\n", len(diff.CasesToCreate))
		for _, c := range diff.CasesToCreate {
			fmt.Printf("  + %s (folder: %d)\n", c.Name, c.FolderID)
		}
	}

	if len(diff.CasesToUpdate) > 0 {
		fmt.Printf("\nCases to UPDATE (%d):\n", len(diff.CasesToUpdate))
		for _, c := range diff.CasesToUpdate {
			fmt.Printf("  ~ %s (ID: %d)\n", c.Name, c.ID)
		}
	}

	if len(diff.CasesToDelete) > 0 {
		fmt.Printf("\nCases to DELETE (%d):\n", len(diff.CasesToDelete))
		for _, id := range diff.CasesToDelete {
			fmt.Printf("  - ID: %d\n", id)
		}
	}

	fmt.Println()
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
	return strings.TrimSpace(result.String())
}
