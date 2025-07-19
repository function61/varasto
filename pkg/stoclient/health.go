package stoclient

// Check the server's health.

import (
	"context"
	"fmt"
	"strings"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func healthEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:    "health",
		Short:  "Run healthcheck against the server",
		Hidden: true, // run by automation
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(wrapWithStopSupport(health))
		},
	}
}

func health(ctx context.Context) error {
	clientConfig, err := ReadConfig()
	if err != nil {
		return err
	}

	healthRoot := stoservertypes.Health{}
	if _, err := ezhttp.Get(ctx, clientConfig.URLBuilder().GetHealth(), ezhttp.RespondsJson(&healthRoot, true)); err != nil {
		return err
	}

	rootCauses := []string{}

	// root causes are leaf-most health nodes where health does not pass
	traverseHealthLeaves(healthRoot, func(leafPath []stoservertypes.Health) {
		// trim everything off *after* rightmost non-passing node
		//
		// root ✅
		// └── child ❌
		//     └── leaf ✅
		//
		// =>
		//
		// root ✅
		// └── child ❌
		trimmedToRightmostProblem := healthPathTrimPassesFromRight(leafPath)

		if len(trimmedToRightmostProblem) == 0 {
			return
		}

		// map nodes to string like "root / child"
		rootCause := strings.Join(lo.Map(trimmedToRightmostProblem, func(h stoservertypes.Health, _ int) string { return h.Title }), " / ")

		// no need to deduplicate since we visit leafs only
		rootCauses = append(rootCauses, rootCause)
	}, nil)

	if len(rootCauses) == 0 {
		return nil
	}

	return fmt.Errorf("problems: %s", strings.Join(rootCauses, ", "))
}

func healthPathTrimPassesFromRight(path []stoservertypes.Health) []stoservertypes.Health {
	trimmed := path
	for i := len(trimmed) - 1; i >= 0; i-- {
		if path[i].Health != stoservertypes.HealthStatusPass {
			break
		}

		trimmed = path[:i-1]
	}
	return trimmed
}

// NOTE: written by AI
func traverseHealthLeaves(h stoservertypes.Health, visitLeaf func([]stoservertypes.Health), path []stoservertypes.Health) {
	path = append(path, h)

	isLeaf := len(h.Children) == 0
	if isLeaf {
		visitLeaf(path)
		return
	}

	for _, child := range h.Children {
		traverseHealthLeaves(child, visitLeaf, path)
	}
}
