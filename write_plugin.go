package main

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
)

func WritePlugin(ctx context.Context, plugins []PluginProvider, lock WaitableLocker, stdout, stderr io.Writer, modules ...string) {

	var tab = tabwriter.NewWriter(stdout, 0, 2, 2, ' ', 0)

	for _, provider := range plugins {

		if len(modules) > 0 && false == inSlice(modules, provider.Module().Path) {
			continue
		}

		fmt.Fprint(tab, "\n")
		fmt.Fprintf(tab, "  Plugin\t%s\n", provider.Module().Path)
		fmt.Fprintf(tab, "  Version\t%s\n", provider.Module().Version)
		fmt.Fprintf(tab, "  Sum\t%s\n", provider.Module().Sum)

		if nil != provider.Module().Replace {
			fmt.Fprintf(tab, "  Replace\t%s@%s\n", provider.Module().Replace.Path, provider.Module().Replace.Version)
		}
	}

	fmt.Fprint(tab, "\n")
	tab.Flush()
}
