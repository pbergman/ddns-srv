package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/libdns/libdns"
)

func WriteShort(ctx context.Context, plugins []PluginProvider, lock WaitableLocker, stdout, stderr io.Writer, rtype, hostname string) {

	for _, provider := range plugins {

		lock.Lock()

		go writeShort(ctx, provider, lock, rtype, hostname, stdout, stderr)
	}

	lock.Wait()
}

func writeShort(ctx context.Context, provider PluginProvider, lock sync.Locker, rtype string, hostname string, stdout, stderr io.Writer) {

	defer lock.Unlock()

	zones, err := provider.ListZones(ctx)

	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: error listing zones: %v\n", provider.Module().Path, err)
		return
	}

	for _, zone := range zones {

		var zoneName = strings.TrimSuffix(zone.Name, ".")

		if hostname == zoneName || strings.HasSuffix(hostname, zoneName) {
			items, err := provider.GetRecords(ctx, zone.Name)

			if err != nil {
				_, _ = fmt.Fprintf(stderr, "%s: error getting records for zone %s: %v\n", provider.Module().Path, zone.Name, err)
				return
			}

			for _, record := range items {

				var name = strings.TrimSuffix(libdns.AbsoluteName(record.RR().Name, zoneName), ".")
				var rr = record.RR()

				if name == hostname && strings.EqualFold(rr.Type, rtype) {
					_, _ = stdout.Write([]byte(rr.Data + "\n"))
				}
			}
		}
	}
}
