package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/zamm-mvp/internal/config"
	"github.com/yourorg/zamm-mvp/internal/models"
	"github.com/yourorg/zamm-mvp/internal/services"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

// App represents the CLI application
type App struct {
	config      *config.Config
	storage     storage.Storage
	specService services.SpecService
	linkService services.LinkService
}

// NewApp creates a new CLI application
func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if err := config.EnsureDirectories(cfg); err != nil {
		return nil, err
	}

	store, err := storage.NewSQLiteStorage(cfg.Database.Path)
	if err != nil {
		return nil, err
	}

	return &App{
		config:      cfg,
		storage:     store,
		specService: services.NewSpecService(store),
		linkService: services.NewLinkService(store),
	}, nil
}

// Close closes the application and cleans up resources
func (a *App) Close() error {
	if a.storage != nil {
		return a.storage.Close()
	}
	return nil
}

// CreateRootCommand creates the root command for the CLI
func (a *App) CreateRootCommand() *cobra.Command {
	var jsonOutput bool
	var quiet bool

	rootCmd := &cobra.Command{
		Use:   "zamm",
		Short: "ZAMM - Zen and the Automation of Metaprogramming for the Masses",
		Long:  "ZAMM is a tool for linking specifications to Git commits, enabling traceability between requirements and implementation.",
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet output")

	// Add subcommands
	rootCmd.AddCommand(a.createSpecCommand(jsonOutput, quiet))
	rootCmd.AddCommand(a.createLinkCommand(jsonOutput, quiet))
	rootCmd.AddCommand(a.createInitCommand())
	rootCmd.AddCommand(a.createStatusCommand(jsonOutput))
	rootCmd.AddCommand(a.createVersionCommand())
	rootCmd.AddCommand(a.createInteractiveCommand())

	return rootCmd
}

// createSpecCommand creates the spec management commands
func (a *App) createSpecCommand(jsonOutput, quiet bool) *cobra.Command {
	specCmd := &cobra.Command{
		Use:   "spec",
		Short: "Manage specifications",
		Long:  "Create, read, update, and delete specification nodes.",
	}

	// spec create
	var title, content string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new specification",
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := a.specService.CreateSpec(title, content)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(spec)
			}

			if !quiet {
				fmt.Printf("Created spec: %s\n", spec.ID)
				fmt.Printf("Title: %s\n", spec.Title)
			}
			return nil
		},
	}
	createCmd.Flags().StringVar(&title, "title", "", "Specification title (required)")
	createCmd.Flags().StringVar(&content, "content", "", "Specification content (required)")
	createCmd.MarkFlagRequired("title")
	createCmd.MarkFlagRequired("content")

	// spec list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all specifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			specs, err := a.specService.ListSpecs()
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(specs)
			}

			return a.outputSpecTable(specs)
		},
	}

	// spec show
	showCmd := &cobra.Command{
		Use:   "show <spec-id>",
		Short: "Show a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := a.specService.GetSpec(args[0])
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(spec)
			}

			return a.outputSpecDetails(spec)
		},
	}

	// spec update
	updateCmd := &cobra.Command{
		Use:   "update <spec-id>",
		Short: "Update a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := a.specService.UpdateSpec(args[0], title, content)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(spec)
			}

			if !quiet {
				fmt.Printf("Updated spec: %s\n", spec.ID)
			}
			return nil
		},
	}
	updateCmd.Flags().StringVar(&title, "title", "", "New specification title")
	updateCmd.Flags().StringVar(&content, "content", "", "New specification content")

	// spec delete
	deleteCmd := &cobra.Command{
		Use:   "delete <spec-id>",
		Short: "Delete a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.specService.DeleteSpec(args[0]); err != nil {
				return err
			}

			if !quiet {
				fmt.Printf("Deleted spec: %s\n", args[0])
			}
			return nil
		},
	}

	specCmd.AddCommand(createCmd, listCmd, showCmd, updateCmd, deleteCmd)
	return specCmd
}

// createLinkCommand creates the link management commands
func (a *App) createLinkCommand(jsonOutput, quiet bool) *cobra.Command {
	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Manage spec-commit links",
		Long:  "Create and manage links between specifications and Git commits.",
	}

	// link create
	var specID, commitID, repoPath, linkType string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a link between spec and commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				repoPath = a.config.Git.DefaultRepo
			}
			if linkType == "" {
				linkType = "implements"
			}

			link, err := a.linkService.LinkSpecToCommit(specID, commitID, repoPath, linkType)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(link)
			}

			if !quiet {
				fmt.Printf("Created link: %s\n", link.ID)
			}
			return nil
		},
	}
	createCmd.Flags().StringVar(&specID, "spec", "", "Specification ID (required)")
	createCmd.Flags().StringVar(&commitID, "commit", "", "Commit hash (required)")
	createCmd.Flags().StringVar(&repoPath, "repo", "", "Repository path (default: current directory)")
	createCmd.Flags().StringVar(&linkType, "type", "implements", "Link type (implements or references)")
	createCmd.MarkFlagRequired("spec")
	createCmd.MarkFlagRequired("commit")

	// link list-by-spec
	listBySpecCmd := &cobra.Command{
		Use:   "list-by-spec <spec-id>",
		Short: "List commits linked to a specification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			links, err := a.linkService.GetCommitsForSpec(args[0])
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(links)
			}

			return a.outputLinkTable(links)
		},
	}

	// link list-by-commit
	listByCommitCmd := &cobra.Command{
		Use:   "list-by-commit <commit-hash>",
		Short: "List specs linked to a commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				repoPath = a.config.Git.DefaultRepo
			}

			specs, err := a.linkService.GetSpecsForCommit(args[0], repoPath)
			if err != nil {
				return err
			}

			if jsonOutput {
				return a.outputJSON(specs)
			}

			return a.outputSpecTable(specs)
		},
	}
	listByCommitCmd.Flags().StringVar(&repoPath, "repo", "", "Repository path (default: current directory)")

	// link delete
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a link between spec and commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if repoPath == "" {
				repoPath = a.config.Git.DefaultRepo
			}

			if err := a.linkService.UnlinkSpecFromCommit(specID, commitID, repoPath); err != nil {
				return err
			}

			if !quiet {
				fmt.Printf("Deleted link between spec %s and commit %s\n", specID, commitID)
			}
			return nil
		},
	}
	deleteCmd.Flags().StringVar(&specID, "spec", "", "Specification ID (required)")
	deleteCmd.Flags().StringVar(&commitID, "commit", "", "Commit hash (required)")
	deleteCmd.Flags().StringVar(&repoPath, "repo", "", "Repository path (default: current directory)")
	deleteCmd.MarkFlagRequired("spec")
	deleteCmd.MarkFlagRequired("commit")

	linkCmd.AddCommand(createCmd, listBySpecCmd, listByCommitCmd, deleteCmd)
	return linkCmd
}

// createInitCommand creates the init command
func (a *App) createInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize zamm in current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.WriteDefaultConfig(); err != nil {
				return err
			}

			// Run database migrations
			if err := a.runMigrations(); err != nil {
				return err
			}

			fmt.Println("Initialized zamm successfully")
			configPath, _ := config.GetConfigPath()
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Printf("Database: %s\n", a.config.Database.Path)
			return nil
		},
	}
}

// createStatusCommand creates the status command
func (a *App) createStatusCommand(jsonOutput bool) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status and statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			specs, err := a.specService.ListSpecs()
			if err != nil {
				return err
			}

			status := map[string]interface{}{
				"config_path":   a.config.Database.Path,
				"database_path": a.config.Database.Path,
				"spec_count":    len(specs),
			}

			if jsonOutput {
				return a.outputJSON(status)
			}

			fmt.Printf("ZAMM Status\n")
			fmt.Printf("===========\n")
			fmt.Printf("Database: %s\n", a.config.Database.Path)
			fmt.Printf("Specifications: %d\n", len(specs))
			return nil
		},
	}
}

// createVersionCommand creates the version command
func (a *App) createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ZAMM MVP v0.1.0")
		},
	}
}

// runMigrations runs database migrations
func (a *App) runMigrations() error {
	// Read migration file
	migrationPath := "migrations/001_initial.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to read migration file", err)
	}

	// Execute migration using the storage interface
	if err := a.storage.RunMigration(string(migrationSQL)); err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to execute migration", err)
	}

	fmt.Printf("Migration executed successfully: %s\n", migrationPath)
	return nil
}

// Output formatting helpers

func (a *App) outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (a *App) outputSpecTable(specs []*models.SpecNode) error {
	if len(specs) == 0 {
		fmt.Println("No specifications found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTABLE ID\tVERSION\tTITLE\tCREATED")

	for _, spec := range specs {
		title := spec.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			spec.ID,
			spec.StableID,
			spec.Version,
			title,
			spec.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	return w.Flush()
}

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

func (a *App) outputSpecDetails(spec *models.SpecNode) error {
	fmt.Printf("ID: %s\n", spec.ID)
	fmt.Printf("Stable ID: %s\n", spec.StableID)
	fmt.Printf("Version: %d\n", spec.Version)
	fmt.Printf("Title: %s\n", spec.Title)
	fmt.Printf("Type: %s\n", spec.NodeType)
	fmt.Printf("Created: %s\n", spec.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", spec.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("\nContent:\n%s\n", strings.Repeat("-", 40))
	fmt.Printf("%s\n", spec.Content)
	return nil
}

func (a *App) outputLinkTable(links []*models.SpecCommitLink) error {
	if len(links) == 0 {
		fmt.Println("No links found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMMIT\tREPO\tTYPE\tCREATED")

	for _, link := range links {
		repoName := filepath.Base(link.RepoPath)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			link.CommitID[:12]+"...",
			repoName,
			link.LinkType,
			link.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	return w.Flush()
}
