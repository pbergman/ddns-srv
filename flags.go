package main

import (
	"flag"
)

func init() {
	flag.Bool("debug", false, "debug mode")
	flag.Int("provider-debug-level", 2, "when in debug mode and prover supports debug interface, this wil set the level (1, 2 or 3)")
	flag.String("config", "/etc/ddns-srv.conf", "config file")
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
