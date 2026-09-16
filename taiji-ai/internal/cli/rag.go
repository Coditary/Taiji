package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/runtime"
)

func newRagCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rag",
		Short: "Index and search knowledge bases (scoped RAG)",
		Long: `Index files or search collections via Wuji. Taiji picks the store from --local / --local-only;
you do not pass store_root.

Default collection comes from config (rag.default_collection, usually "default"). Override with --collection.`,
	}
	cmd.AddCommand(newRagIndexCmd(app), newRagQueryCmd(app))
	return cmd
}

func newRagIndexCmd(app *App) *cobra.Command {
	var (
		collection string
		recursive  bool
		glob       string
		excludes   []string
	)
	cmd := &cobra.Command{
		Use:   "index [path...]",
		Short: "Index files or directories into a collection",
		Long: `Index one or more paths into a RAG collection at the active scope.

Examples:
  taiji --local rag index .
  taiji rag index ./docs --glob "*.md"
  taiji rag index chapter1.md chapter2.md --collection book`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := args
			if len(paths) == 0 {
				paths = []string{"."}
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			rt, err := runtime.New(ctx, app.Config, app.runtimeOptions(cmd))
			if err != nil {
				return err
			}
			defer rt.Close()
			if rt.RAG == nil {
				return fmt.Errorf("rag is not available (check config and wuji connection)")
			}
			if collection == "" {
				collection = rt.RAG.DefaultCollection
			}
			res, err := rt.RAG.Index(ctx, rag.IndexRequest{
				Collection:  collection,
				SourcePaths: paths,
				Recursive:   recursive,
				Glob:        glob,
				Exclude:     defaultIndexExcludes(paths, excludes),
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "scope: %s (%s)\n", rt.RAG.DescribeScope(), res.Scope)
			fmt.Fprintf(os.Stderr, "collection: %s\n", res.Collection)
			fmt.Fprintf(os.Stdout, "%s (+%d chunks, %d total)\n", res.Message, res.ChunksAdded, res.TotalChunks)
			return nil
		},
	}
	cmd.Flags().StringVar(&collection, "collection", "", "collection name (default: rag.default_collection from config)")
	cmd.Flags().BoolVar(&recursive, "recursive", true, "recurse into directories")
	cmd.Flags().StringVar(&glob, "glob", "", "filename glob when indexing directories (e.g. *.md)")
	cmd.Flags().StringSliceVar(&excludes, "exclude", nil, "skip paths containing this substring (repeatable; default for .: node_modules, .git, .wuji, .taiji)")
	return cmd
}

func newRagQueryCmd(app *App) *cobra.Command {
	var (
		collection string
		topK       int
		minScore   float32
	)
	cmd := &cobra.Command{
		Use:   "query <text>",
		Short: "Search a collection for relevant passages",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(strings.Join(args, " "))
			if query == "" {
				return fmt.Errorf("query is required")
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			rt, err := runtime.New(ctx, app.Config, app.runtimeOptions(cmd))
			if err != nil {
				return err
			}
			defer rt.Close()
			if rt.RAG == nil {
				return fmt.Errorf("rag is not available (check config and wuji connection)")
			}
			if collection == "" {
				collection = rt.RAG.DefaultCollection
			}
			hits, err := rt.RAG.Query(ctx, rag.QueryRequest{
				Collection: collection,
				Query:      query,
				TopK:       topK,
				MinScore:   minScore,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "scope: %s\n", rt.RAG.DescribeScope())
			fmt.Fprintf(os.Stderr, "collection: %s\n", collection)
			if len(hits) == 0 {
				fmt.Fprintln(os.Stdout, "no hits")
				return nil
			}
			for i, hit := range hits {
				fmt.Fprintf(os.Stdout, "[%d] scope=%s score=%.3f source=%s\n%s\n\n", i+1, hit.Scope, hit.Score, hit.Source, hit.Text)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&collection, "collection", "", "collection name (default: rag.default_collection from config)")
	cmd.Flags().IntVar(&topK, "top-k", 5, "maximum number of chunks")
	cmd.Flags().Float32Var(&minScore, "min-score", 0, "minimum similarity score")
	return cmd
}

// defaultIndexExcludes returns skip patterns when indexing the project root.
func defaultIndexExcludes(paths []string, userExcludes []string) []string {
	if len(userExcludes) > 0 {
		return userExcludes
	}
	for _, p := range paths {
		if p == "." || p == "./" || strings.TrimSpace(p) == "" {
			return []string{"node_modules", ".git", ".wuji", ".taiji", ".cache", "vendor"}
		}
	}
	return nil
}
