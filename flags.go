package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

func init() {
	flag.Bool("debug", false, "debug mode")
	flag.Int("provider-debug-level", 2, "when in debug mode and prover supports debug interface, this wil set the level (1, 2 or 3)")
	flag.String("config", "/etc/ddns-srv.conf", "config file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <package...>\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\n Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n Commands:\n")

		var tab = tabwriter.NewWriter(flag.CommandLine.Output(), 0, 2, 2, ' ', 0)

		fmt.Fprintln(tab, "  run\t\twill run server")
		fmt.Fprintln(tab, "  lookup\t[type] <hostname>\tdo quick lookup hot given hostname")
		fmt.Fprintln(tab, "  records\t[module...]\tprint record")
		fmt.Fprintln(tab, "  zones\t[module...]\tprint zones")
		fmt.Fprintln(tab, "  inspect\t[module...]\tprint plugin information")
		fmt.Fprintln(tab, "  version\t\tprint version of application")

		tab.Flush()
	}

}

func inputOption[T bool | string | int](name string, empty T) T {

	var input = flag.Lookup(name)

	if input == nil {
		return empty
	}

	if getter, x := input.Value.(flag.Getter); x {
		if ret, y := getter.Get().(T); y {
			return ret
		}
	}

	return empty
}
