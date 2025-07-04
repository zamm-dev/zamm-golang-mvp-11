package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/yourorg/zamm-mvp/internal/models"
)

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
	fmt.Fprintln(w, "ID\tTITLE\tCREATED")

	for _, spec := range specs {
		title := spec.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			spec.ID,
			title,
			spec.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	return w.Flush()
}

func (a *App) outputSpecDetails(spec *models.SpecNode) error {
	fmt.Printf("ID: %s\n", spec.ID)
	fmt.Printf("Title: %s\n", spec.Title)
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
			link.LinkLabel,
			link.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	return w.Flush()
}
