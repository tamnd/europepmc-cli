package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/europepmc-cli/europepmc"
)

func (a *App) referencesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "references <id>",
		Short: "List references cited by a given article",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(25)
			source, id := europepmc.ResolveID(args[0])
			if source == "DOI" {
				// Resolve DOI to (source, id) first.
				art, err := a.client.ArticleByID(cmd.Context(), args[0])
				if err != nil {
					return mapFetchErr(err)
				}
				source = art.Source
				id = art.PMID
			}
			a.progressf("fetching references for %s/%s...", source, id)
			arts, err := a.client.References(cmd.Context(), source, id, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(arts, len(arts))
		},
	}
}
