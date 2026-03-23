// Dev tool: generates a reference file listing all registered action keys.
// Run from the module root:
//
//	go run ./tools/gen-actionkey-ref > docs/actionkeys_ref.md
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gokafka/internal/shared"
)

func main() {
	keys := shared.RegisteredActionKeys()
	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})

	// Group by parent resource (middle segment)
	groups := make(map[string][]string)
	for _, k := range keys {
		parts := strings.SplitN(string(k), ":", 3)
		if len(parts) != 3 {
			continue
		}
		parent := parts[1]
		groups[parent] = append(groups[parent], string(k))
	}

	groupNames := make([]string, 0, len(groups))
	for g := range groups {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	out := os.Stdout
	fmt.Fprintln(out, "# Action Keys Reference")
	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "Format: `resource:parent:verb`  —  total: %d\n", len(keys))
	fmt.Fprintln(out, "")

	for _, g := range groupNames {
		fmt.Fprintf(out, "## %s\n\n", g)
		for _, k := range groups[g] {
			fmt.Fprintf(out, "- `%s`\n", k)
		}
		fmt.Fprintln(out, "")
	}
}
