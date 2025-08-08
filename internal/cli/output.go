package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/yourorg/zamm-mvp/internal/models"
)

// Output formatting helpers

func (a *App) outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (a *App) outputSpecTable(specs []*models.Spec) error {
	if len(specs) == 0 {
		fmt.Println("No specifications found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTITLE")

	for _, spec := range specs {
		title := spec.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\n",
			spec.ID,
			title,
		)
	}

	return w.Flush()
}

func (a *App) outputNodeTable(nodes []models.Node) error {
	if len(nodes) == 0 {
		fmt.Println("No nodes found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tTITLE")

	for _, node := range nodes {
		title := node.GetTitle()
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			node.GetID(),
			node.GetType(),
			title,
		)
	}

	return w.Flush()
}

func (a *App) outputSpecDetails(node models.Node) error {
	fmt.Printf("ID: %s\n", node.GetID())
	fmt.Printf("Title: %s\n", node.GetTitle())
	fmt.Printf("Type: %s\n", node.GetType())
	fmt.Printf("\nContent:\n%s\n", strings.Repeat("-", 40))
	fmt.Printf("%s\n", node.GetContent())
	return nil
}

func (a *App) outputLinkTable(links []*models.SpecCommitLink) error {
	if len(links) == 0 {
		fmt.Println("No links found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "COMMIT\tREPO\tTYPE")

	for _, link := range links {
		repoName := filepath.Base(link.RepoPath)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			link.CommitID[:12]+"...",
			repoName,
			link.LinkLabel,
		)
	}

	return w.Flush()
}
