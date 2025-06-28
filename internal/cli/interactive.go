package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// createInteractiveCommand creates the interactive mode command
func (a *App) createInteractiveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Interactive mode for managing specs and links",
		Long:  "Start an interactive session to manage specifications and links without needing to copy-paste IDs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runInteractiveMode()
		},
	}
}

// runInteractiveMode starts the interactive mode
func (a *App) runInteractiveMode() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🚀 ZAMM Interactive Mode")
	fmt.Println("========================")
	fmt.Println()

	for {
		fmt.Println("What would you like to do?")
		fmt.Println("1. List specifications")
		fmt.Println("2. Create new specification")
		fmt.Println("3. Edit specification")
		fmt.Println("4. Delete specification")
		fmt.Println("5. Link specification to commit")
		fmt.Println("6. View spec-commit links")
		fmt.Println("7. Delete spec-commit link")
		fmt.Println("8. Exit")
		fmt.Print("\nEnter your choice (1-8): ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		choice := strings.TrimSpace(input)
		fmt.Println()

		switch choice {
		case "1":
			if err := a.interactiveListSpecs(); err != nil {
				fmt.Printf("Error: %v\n\n", err)
			}
		case "2":
			if err := a.interactiveCreateSpec(reader); err != nil {
				fmt.Printf("Error: %v\n\n", err)
			}
		case "3":
			if err := a.interactiveEditSpec(reader); err != nil {
				fmt.Printf("Error: %v\n\n", err)
			}
		case "4":
			if err := a.interactiveDeleteSpec(reader); err != nil {
				fmt.Printf("Error: %v\n\n", err)
			}
		case "5":
			if err := a.interactiveLinkSpec(reader); err != nil {
				fmt.Printf("Error: %v\n\n", err)
			}
		case "6":
			if err := a.interactiveViewLinks(reader); err != nil {
				fmt.Printf("Error: %v\n\n", err)
			}
		case "7":
			if err := a.interactiveDeleteLink(reader); err != nil {
				fmt.Printf("Error: %v\n\n", err)
			}
		case "8":
			fmt.Println("Goodbye! 👋")
			return nil
		default:
			fmt.Println("Invalid choice. Please enter a number between 1 and 8.\n")
		}
	}
}

// interactiveListSpecs lists all specifications with numbers for easy selection
func (a *App) interactiveListSpecs() error {
	specs, err := a.specService.ListSpecs()
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No specifications found.")
		fmt.Println()
		return nil
	}

	fmt.Printf("Found %d specifications:\n\n", len(specs))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tTITLE\tCREATED\tID")
	fmt.Fprintln(w, "─\t─────\t───────\t──")

	for i, spec := range specs {
		title := spec.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			i+1,
			title,
			spec.CreatedAt.Format("2006-01-02 15:04"),
			spec.ID[:8]+"...",
		)
	}

	w.Flush()
	fmt.Println()
	return nil
}

// interactiveCreateSpec creates a new specification interactively
func (a *App) interactiveCreateSpec(reader *bufio.Reader) error {
	fmt.Println("📝 Create New Specification")
	fmt.Println("===========================")

	fmt.Print("Enter title: ")
	title, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	title = strings.TrimSpace(title)

	fmt.Print("Enter content (end with empty line): ")
	var contentLines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			break
		}
		contentLines = append(contentLines, line)
	}
	content := strings.Join(contentLines, "\n")

	spec, err := a.specService.CreateSpec(title, content)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Created specification: %s\n", spec.Title)
	fmt.Printf("   ID: %s\n\n", spec.ID)
	return nil
}

// interactiveEditSpec edits an existing specification
func (a *App) interactiveEditSpec(reader *bufio.Reader) error {
	specs, err := a.specService.ListSpecs()
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No specifications found to edit.")
		fmt.Println()
		return nil
	}

	fmt.Println("📝 Edit Specification")
	fmt.Println("====================")

	// Show specs with numbers
	a.interactiveListSpecs()

	fmt.Print("Enter specification number to edit: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || num < 1 || num > len(specs) {
		return fmt.Errorf("invalid specification number")
	}

	selectedSpec := specs[num-1]

	fmt.Printf("\nEditing: %s\n", selectedSpec.Title)
	fmt.Printf("Current content:\n%s\n\n", selectedSpec.Content)

	fmt.Printf("Enter new title (or press Enter to keep '%s'): ", selectedSpec.Title)
	titleInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	newTitle := strings.TrimSpace(titleInput)
	if newTitle == "" {
		newTitle = selectedSpec.Title
	}

	fmt.Print("Enter new content (end with empty line, or press Enter twice to keep existing): ")
	var contentLines []string
	emptyLineCount := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			emptyLineCount++
			if emptyLineCount >= 2 {
				break
			}
		} else {
			emptyLineCount = 0
		}
		contentLines = append(contentLines, line)
	}

	newContent := strings.Join(contentLines, "\n")
	if strings.TrimSpace(newContent) == "" {
		newContent = selectedSpec.Content
	}

	updatedSpec, err := a.specService.UpdateSpec(selectedSpec.ID, newTitle, newContent)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Updated specification: %s\n\n", updatedSpec.Title)
	return nil
}

// interactiveDeleteSpec deletes a specification
func (a *App) interactiveDeleteSpec(reader *bufio.Reader) error {
	specs, err := a.specService.ListSpecs()
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No specifications found to delete.")
		fmt.Println()
		return nil
	}

	fmt.Println("🗑️  Delete Specification")
	fmt.Println("========================")

	// Show specs with numbers
	a.interactiveListSpecs()

	fmt.Print("Enter specification number to delete: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || num < 1 || num > len(specs) {
		return fmt.Errorf("invalid specification number")
	}

	selectedSpec := specs[num-1]

	fmt.Printf("\n⚠️  Are you sure you want to delete '%s'? (y/N): ", selectedSpec.Title)
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println("Deletion cancelled.\n")
		return nil
	}

	if err := a.specService.DeleteSpec(selectedSpec.ID); err != nil {
		return err
	}

	fmt.Printf("✅ Deleted specification: %s\n\n", selectedSpec.Title)
	return nil
}

// interactiveLinkSpec links a specification to a commit
func (a *App) interactiveLinkSpec(reader *bufio.Reader) error {
	specs, err := a.specService.ListSpecs()
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No specifications found to link.")
		fmt.Println()
		return nil
	}

	fmt.Println("🔗 Link Specification to Commit")
	fmt.Println("===============================")

	// Show specs with numbers
	a.interactiveListSpecs()

	fmt.Print("Enter specification number to link: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || num < 1 || num > len(specs) {
		return fmt.Errorf("invalid specification number")
	}

	selectedSpec := specs[num-1]

	fmt.Print("Enter commit hash: ")
	commitInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	commitID := strings.TrimSpace(commitInput)

	fmt.Print("Enter repository path (or press Enter for current directory): ")
	repoInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	repoPath := strings.TrimSpace(repoInput)
	if repoPath == "" {
		repoPath = a.config.Git.DefaultRepo
	}

	fmt.Print("Enter link type (implements/references, default: implements): ")
	typeInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	linkType := strings.TrimSpace(typeInput)
	if linkType == "" {
		linkType = "implements"
	}

	link, err := a.linkService.LinkSpecToCommit(selectedSpec.ID, commitID, repoPath, linkType)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Created link between '%s' and commit %s\n", selectedSpec.Title, commitID[:12]+"...")
	fmt.Printf("   Link ID: %s\n\n", link.ID)
	return nil
}

// interactiveViewLinks views links for a specification
func (a *App) interactiveViewLinks(reader *bufio.Reader) error {
	specs, err := a.specService.ListSpecs()
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No specifications found.")
		fmt.Println()
		return nil
	}

	fmt.Println("🔗 View Specification Links")
	fmt.Println("===========================")

	// Show specs with numbers
	a.interactiveListSpecs()

	fmt.Print("Enter specification number to view links: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || num < 1 || num > len(specs) {
		return fmt.Errorf("invalid specification number")
	}

	selectedSpec := specs[num-1]

	links, err := a.linkService.GetCommitsForSpec(selectedSpec.ID)
	if err != nil {
		return err
	}

	fmt.Printf("\nLinks for '%s':\n", selectedSpec.Title)
	if len(links) == 0 {
		fmt.Println("No links found.\n")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMMIT\tREPO\tTYPE\tCREATED")
	fmt.Fprintln(w, "──────\t────\t────\t───────")

	for _, link := range links {
		repoName := filepath.Base(link.RepoPath)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			link.CommitID[:12]+"...",
			repoName,
			link.LinkType,
			link.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()
	fmt.Println()
	return nil
}

// interactiveDeleteLink deletes a spec-commit link
func (a *App) interactiveDeleteLink(reader *bufio.Reader) error {
	specs, err := a.specService.ListSpecs()
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No specifications found.")
		fmt.Println()
		return nil
	}

	fmt.Println("🗑️  Delete Specification Link")
	fmt.Println("=============================")

	// Show specs with numbers
	a.interactiveListSpecs()

	fmt.Print("Enter specification number to delete links from: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || num < 1 || num > len(specs) {
		return fmt.Errorf("invalid specification number")
	}

	selectedSpec := specs[num-1]

	links, err := a.linkService.GetCommitsForSpec(selectedSpec.ID)
	if err != nil {
		return err
	}

	if len(links) == 0 {
		fmt.Printf("No links found for '%s'.\n\n", selectedSpec.Title)
		return nil
	}

	fmt.Printf("\nLinks for '%s':\n", selectedSpec.Title)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tCOMMIT\tREPO\tTYPE")
	fmt.Fprintln(w, "─\t──────\t────\t────")

	for i, link := range links {
		repoName := filepath.Base(link.RepoPath)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
			i+1,
			link.CommitID[:12]+"...",
			repoName,
			link.LinkType,
		)
	}
	w.Flush()

	fmt.Print("\nEnter link number to delete: ")
	linkInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	linkNum, err := strconv.Atoi(strings.TrimSpace(linkInput))
	if err != nil || linkNum < 1 || linkNum > len(links) {
		return fmt.Errorf("invalid link number")
	}

	selectedLink := links[linkNum-1]

	fmt.Printf("⚠️  Are you sure you want to delete the link to commit %s? (y/N): ", selectedLink.CommitID[:12]+"...")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println("Deletion cancelled.\n")
		return nil
	}

	if err := a.linkService.UnlinkSpecFromCommit(selectedSpec.ID, selectedLink.CommitID, selectedLink.RepoPath); err != nil {
		return err
	}

	fmt.Printf("✅ Deleted link to commit %s\n\n", selectedLink.CommitID[:12]+"...")
	return nil
}
