package main

import (
	"fmt"
	"io"

	"github.com/akyrey/postman-sync/internal/postman"
)

var changeSymbols = map[postman.ChangeKind]string{
	postman.ChangeAdded:    "+",
	postman.ChangeRemoved:  "-",
	postman.ChangeModified: "~",
}

// printChanges writes a human-readable list of collection changes followed by
// a summary line.
func printChanges(w io.Writer, changes []postman.ItemChange) {
	counts := map[postman.ChangeKind]int{}
	for _, c := range changes {
		counts[c.Kind]++
		fmt.Fprintf(w, "%s %s\n", changeSymbols[c.Kind], c.Path)
		for _, d := range c.Details {
			fmt.Fprintf(w, "    %s\n", d)
		}
	}
	if len(changes) == 0 {
		fmt.Fprintln(w, "No changes.")
		return
	}
	fmt.Fprintf(w, "Summary: %d added, %d removed, %d modified.\n",
		counts[postman.ChangeAdded], counts[postman.ChangeRemoved], counts[postman.ChangeModified])
}
