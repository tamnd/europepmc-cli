package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) articleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "article <id>",
		Short: "Show a single article by PMID, PMC ID, or DOI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching article %q...", args[0])
			art, err := a.client.ArticleByID(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render([]interface{}{art})
		},
	}
}
