package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) preprintsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preprints",
		Short: "Recent preprints from bioRxiv, medRxiv, ChemRxiv, and more",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching %d recent preprints...", n)
			articles, err := a.client.Search(cmd.Context(), "source:PPR", n, "date")
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(articles, len(articles))
		},
	}
}
