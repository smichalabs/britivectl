package output

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

// PrintTable renders a table with the given headers and rows to stdout.
func PrintTable(headers []string, rows [][]string) {
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithHeader(headers),
	)
	for _, row := range rows {
		iface := make([]interface{}, len(row))
		for i, v := range row {
			iface[i] = v
		}
		_ = table.Append(iface...)
	}
	_ = table.Render()
}
