package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	var sort string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Europe PMC articles",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(20)
			a.progressf("searching for %q...", args[0])
			articles, err := a.client.Search(cmd.Context(), args[0], n, sort)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(articles, len(articles))
		},
	}
	cmd.Flags().StringVar(&sort, "sort", "cited", "sort order: cited|date|relevance")
	return cmd
}
