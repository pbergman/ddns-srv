package main

import (
	"context"

	"github.com/libdns/libdns"
)

type ZoneAwareProvider interface {
	Provider
	libdns.ZoneLister
}

type Provider interface {
	libdns.RecordSetter
	libdns.RecordDeleter
	libdns.RecordGetter
	libdns.RecordAppender
}

func NewZoneProvider(inner Provider, zones ...string) ZoneAwareProvider {
	var x = make([]libdns.Zone, len(zones))

	for i, c := 0, len(zones); i < c; i++ {
		x[i] = libdns.Zone{Name: zones[i]}
	}

	return &StaticZoneProvider{
		Provider: inner,
		zones:    x,
	}
}

type StaticZoneProvider struct {
	Provider
	zones []libdns.Zone
}

func (z *StaticZoneProvider) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	return z.zones, nil
}
