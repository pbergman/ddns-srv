package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libdns/libdns"
)

type zoneRecords map[string][]*libdns.RR

func WriteRecords(ctx context.Context, plugins []PluginProvider, lock WaitableLocker, stdout, stderr io.Writer, modules ...string) {

	var mapped sync.Map
	var sizes [4]uint64

	for _, provider := range plugins {

		if len(modules) > 0 && false == inSlice(modules, provider.Module().Path) {
			continue
		}

		lock.Lock()

		go fetchRecords(ctx, lock, provider, &mapped, &sizes, stderr)
	}

	lock.Wait()

	writeRecords(&mapped, sizes, stdout)
}

func fetchRecords(ctx context.Context, lock sync.Locker, provider PluginProvider, mapped *sync.Map, sizes *[4]uint64, stderr io.Writer) {

	defer lock.Unlock()

	zones, err := provider.ListZones(ctx)

	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: error listing zones: %v\n", provider.Module().Path, err)
		return
	}

	var records = make(zoneRecords)

	for _, zone := range zones {

		items, err := provider.GetRecords(ctx, zone.Name)

		if err != nil {

			if false == errors.Is(err, context.Canceled) {
				_, _ = fmt.Fprintf(stderr, "%s: error listing records for zone %q: %v\n", provider.Module().Path, zone.Name, err)
			}

			return
		}

		for _, record := range items {

			rr := record.RR()
			rr.Name = libdns.AbsoluteName(rr.Name, zone.Name)

			var rrSizes = [4]uint64{
				uint64(len(rr.Name)),
				uint64(len(rr.Type)),
				uint64(len(rr.TTL.String())),
				uint64(len(rr.Data)),
			}

			for i := 0; i < 4; i++ {
				var loop = true
				for loop {
					var curr = atomic.LoadUint64(&sizes[i])
					if curr < rrSizes[i] {
						loop = false == atomic.CompareAndSwapUint64(&sizes[i], curr, rrSizes[i])
					} else {
						loop = false
					}
				}
			}

			records[zone.Name] = append(records[zone.Name], &rr)
		}
	}

	if len(records) > 0 {
		mapped.Store(provider.Module().Path, records)
	}
}

func writeRecords(mapped *sync.Map, sizes [4]uint64, writer io.Writer) {

	var maxHead uint64
	var headers = []string{"Name", "Type", "TTL", "Data"}
	var modules = make([]string, 0)
	var zones = make(map[string][]string, 0)

	mapped.Range(func(x, y interface{}) bool {

		var name = x.(string)
		var items = y.(zoneRecords)

		if x := uint64(len(name)); x > maxHead {
			maxHead = x
		}

		modules = append(modules, name)

		for zone, _ := range items {

			if x := uint64(len(zone)) + 3; x > maxHead {
				maxHead = x
			}

			zones[name] = append(zones[name], zone)
		}

		return true
	})

	for i, c := 0, len(headers); i < c; i++ {
		if x := len(headers[i]); x > int(sizes[i]) {
			sizes[i] = uint64(x)
		}
	}

	if total := sizes[0] + sizes[1] + sizes[2] + sizes[3]; total < maxHead {
		sizes[0] = maxHead - total
	}

	var format = fmt.Sprintf("%%s%%-%ds%%s%%-%ds%%s%%-%ds%%s%%-%ds%%s\n", sizes[0]+2, sizes[1]+2, sizes[2]+2, sizes[3]+2) // 8

	var line = make([]any, 9)
	line[1] = strings.Repeat("─", int(sizes[0]+2))
	line[3] = strings.Repeat("─", int(sizes[1]+2))
	line[5] = strings.Repeat("─", int(sizes[2]+2))
	line[7] = strings.Repeat("─", int(sizes[3]+2))

	sort.Strings(modules)

	for i, c := 0, len(modules); i < c; i++ {
		var module = modules[i]

		_, _ = fmt.Fprintf(writer, format, barArgs(line, "┌", "─", "┐")...)
		_, _ = fmt.Fprintf(writer, "│ %-"+strconv.Itoa(int(sizes[0]+sizes[1]+sizes[2]+sizes[3])+10)+"s│\n", module)
		_, _ = fmt.Fprintf(writer, format, barArgs(line, "├", "┬", "┤")...)
		_, _ = fmt.Fprintf(writer, format, rowArgs(headers[0], headers[1], headers[2], headers[3], "│", "│", "│")...)
		_, _ = fmt.Fprintf(writer, format, barArgs(line, "├", "┼", "┤")...)

		var zoneList = zones[module]
		var zones, _ = mapped.Load(module)

		sort.Strings(zoneList)

		for x, y := 0, len(zoneList); x < y; x++ {
			for _, record := range zones.(zoneRecords)[zoneList[x]] {
				_, _ = fmt.Fprintf(writer, format, rowArgs(record.Name, record.Type, record.TTL.Round(time.Second).String(), record.Data, "│", "│", "│")...)
			}
		}

		_, _ = fmt.Fprintf(writer, format, barArgs(line, "└", "┴", "┘")...)
	}
}

func padding(x string) string {
	return " " + x + " "
}

func rowArgs(first, second, third, fourth, left, divider, right string) []any {
	return []any{
		left,
		padding(first),
		divider,
		padding(second),
		divider,
		padding(third),
		divider,
		padding(fourth),
		right,
	}
}

func barArgs(args []any, left, divider, right string) []any {
	args[0] = left
	args[2] = divider
	args[4] = divider
	args[6] = divider
	args[8] = right
	return args
}
