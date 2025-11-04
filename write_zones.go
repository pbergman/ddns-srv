package main

import (
	"context"
	"fmt"
	"io"
	"sync"
)

func WriteZones(ctx context.Context, plugins []PluginProvider, lock WaitableLocker, stdout, stderr io.Writer, modules ...string) {

	var zones sync.Map

	for _, provider := range plugins {

		if len(modules) > 0 && false == inSlice(modules, provider.Module().Path) {
			continue
		}

		lock.Lock()

		go fetchZones(ctx, lock, provider, &zones, stdout, stderr)
	}

	lock.Wait()

	writeZones(&zones, stdout)
}

func fetchZones(ctx context.Context, lock sync.Locker, provider PluginProvider, mapped *sync.Map, stdout, stderr io.Writer) {

	defer lock.Unlock()

	zones, err := provider.ListZones(ctx)

	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: error listing zones: %v\n", provider.Module().Path, err)
		return
	}

	var items = make([]string, len(zones))

	for idx, zone := range zones {
		items[idx] = zone.Name
	}

	mapped.Store(provider.Module().Path, items)

}

func writeZones(mapped *sync.Map, stdout io.Writer) {
	mapped.Range(func(key, value interface{}) bool {
		_, _ = fmt.Fprintf(stdout, "• %s\n", key.(string))

		var list = value.([]string)

		for i, c := 0, len(list)-1; i <= c; i++ {

			var prefix = "├─ "

			if c == i {
				prefix = "└─ "
			}

			_, _ = fmt.Fprintf(stdout, "%s%s\n", prefix, list[i])
		}

		return true
	})
}
