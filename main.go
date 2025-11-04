package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
)

func main() {

	flag.Parse()

	var logger, level = getOutput(
		inputOption("debug", false),
		inputOption("provider-debug-level", 2),
	)

	switch c := flag.Arg(0); c {
	case "version":
		if info, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(info.Main.Version)
		}
	case "run":
		run(logger, level)
	case "records", "zones", "lookup", "inspect":

		var locker = NewSemaphore(5)
		var _, providers, err = bootstrap(
			logger,
			inputOption("config", ""),
			level,
		)

		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}

		switch c {
		case "records":
			WriteRecords(context.Background(), providers, locker, os.Stdout, os.Stderr, flag.Args()[1:]...)
		case "zones":
			WriteZones(context.Background(), providers, locker, os.Stdout, os.Stderr, flag.Args()[1:]...)
		case "inspect":
			WritePlugin(context.Background(), providers, locker, os.Stdout, os.Stderr, flag.Args()[1:]...)
		default:

			if flag.NArg() < 2 || flag.NArg() > 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s lookup [type] <hostname>\n", os.Args[0])
				os.Exit(1)
			}

			var rtype, hostname string

			if flag.NArg() == 2 {
				rtype = "A"
				hostname = flag.Arg(1)
			} else {
				rtype = flag.Arg(1)
				hostname = flag.Arg(2)
			}

			WriteShort(context.Background(), providers, locker, os.Stdout, os.Stderr, rtype, hostname)
		}
	default:
		flag.Usage()
	}
}
