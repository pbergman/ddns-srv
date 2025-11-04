package main

import (
	"context"
	"runtime/debug"

	"github.com/libdns/libdns"
)

type PluginProvider interface {
	ZoneAwareProvider
	Module() *debug.Module
}
type ZoneAwareProvider interface {
	BaseProvider
	libdns.ZoneLister
}

type BaseProvider interface {
	libdns.RecordSetter
	libdns.RecordDeleter
	libdns.RecordGetter
	libdns.RecordAppender
}

func NewStaticZoneProvider(inner BaseProvider, zones ...string) ZoneAwareProvider {
	var x = make([]libdns.Zone, len(zones))

	for i, c := 0, len(zones); i < c; i++ {
		x[i] = libdns.Zone{Name: zones[i]}
	}

	return &StaticZoneProvider{
		BaseProvider: inner,
		zones:        x,
	}
}

type StaticZoneProvider struct {
	BaseProvider
	zones []libdns.Zone
}

func (z *StaticZoneProvider) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	return z.zones, nil
}

type Provider struct {
	ZoneAwareProvider
	module *debug.Module
}

func (p *Provider) Module() *debug.Module {
	return p.module
}
