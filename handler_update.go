package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/libdns/libdns"
	"github.com/pbergman/logger"
)

type UpdateResult struct {
	items []string
	lock  sync.Mutex
}

func (u *UpdateResult) WriteTo(w io.Writer) (n int64, err error) {
	u.lock.Lock()
	defer u.lock.Unlock()

	x, err := w.Write([]byte(strings.Join(u.items, "\n")))

	if err != nil {
		return int64(x), err
	}

	return int64(x), nil
}

func (u *UpdateResult) Set(idx int, value string) {
	u.lock.Lock()
	u.items[idx] = value
	u.lock.Unlock()
}

func NewUpdateResult(size int, ip *netip.Addr) *UpdateResult {
	var update = &UpdateResult{
		items: make([]string, size),
	}

	for i := 0; i < size; i++ {
		update.items[i] = fmt.Sprintf("nochg %s", ip)
	}

	return update
}

func NewUpdateHandler(plugins map[string]ZoneAwareProvider, logger *logger.Logger, trusted *IPPrefixList) Handler {
	return &UpdateHandler{
		plugins: plugins,
		logger:  logger,
		trusted: trusted,
	}
}

type UpdateHandler struct {
	plugins map[string]ZoneAwareProvider
	logger  *logger.Logger
	trusted *IPPrefixList
	locals  *IPPrefixList
}

func (u *UpdateHandler) Supports(url *url.URL) bool {
	return url.Path == "/nic/update"
}

func (u *UpdateHandler) Handle(response http.ResponseWriter, request *http.Request) (status HandleResult) {

	var query = request.URL.Query()
	var err error
	var hosts []string

	status = StopPropagation

	if hosts, err = u.getHosts(query); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	var ip netip.Addr

	if ip, err = u.getIp(query, request); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	var lock = NewSemaphore(5)
	var result = NewUpdateResult(len(hosts), &ip)
	var zones map[string][]string

	defer result.WriteTo(response)

	if zones, err = u.fetchZones(request.Context(), lock); err != nil {
		http.Error(response, err.Error(), http.StatusFailedDependency)
		return
	}

	for module, items := range u.makeUpdateLists(hosts, ip, zones, result) {
		lock.Lock()
		go u.updateRecords(request.Context(), hosts, result, items, u.plugins[module], &ip, lock)
	}

	lock.Wait()

	return
}

func (u *UpdateHandler) getHosts(query url.Values) ([]string, error) {

	if false == query.Has("hostname") {
		return nil, errors.New("missing required hostname")
	}

	return strings.Split(query.Get("hostname"), ","), nil
}

func (u *UpdateHandler) fetchZones(ctx context.Context, lock WaitableLocker) (map[string][]string, error) {

	var zones = make(map[string][]string)
	var errList = make([]error, 0)

	for module, plugin := range u.plugins {

		lock.Lock()

		var list = zones[module]
		var err error

		errList = append(errList, err)

		go u.fetchZonesForModule(ctx, lock, &list, plugin, &err)
	}

	lock.Wait()

	return zones, errors.Join(errList...)
}

func (u *UpdateHandler) fetchZonesForModule(ctx context.Context, lock sync.Locker, list *[]string, provider ZoneAwareProvider, ref *error) {

	defer lock.Unlock()

	zones, err := provider.ListZones(ctx)

	if err != nil {
		*ref = err
		return
	}

	*list = make([]string, len(zones), len(zones))

	for idx, zone := range zones {
		(*list)[idx] = strings.TrimSuffix(zone.Name, ".")
	}
}

func (u *UpdateHandler) makeUpdateLists(hosts []string, ip netip.Addr, zones map[string][]string, result *UpdateResult) map[string]map[string][]libdns.Record {

	var updates = make(map[string]map[string][]libdns.Record)

hostnames:
	for idx, hostname := range hosts {

		u.logger.Debug(fmt.Sprintf("lookup provider for hostname '%s'", hostname))

		for module, _ := range u.plugins {

			for _, zone := range zones[module] {
				if strings.HasSuffix(hostname, "."+zone) {

					u.logger.Debug(fmt.Sprintf("hostname %s matches zone %s (module %s)", hostname, zone, module))

					if _, ok := updates[module]; !ok {
						updates[module] = make(map[string][]libdns.Record)
					}

					if _, ok := updates[module][zone]; !ok {
						updates[module][zone] = make([]libdns.Record, 0)
					}

					updates[module][zone] = append(updates[module][zone], libdns.Address{
						Name: libdns.RelativeName(hostname, zone),
						TTL:  (time.Minute * 5).Round(time.Second),
						IP:   ip,
					})

					continue hostnames
				}
			}

			u.logger.Debug(fmt.Sprintf("hostname %s is not supported by module %s", hostname, module))
		}

		result.Set(idx, "nohost")
	}

	return updates
}

func (u *UpdateHandler) getIp(query url.Values, request *http.Request) (netip.Addr, error) {

	if query.Has("myip") {
		if value, err := netip.ParseAddr(query.Get("myip")); err == nil {
			return value, nil
		}
	}

	remote, err := netip.ParseAddrPort(request.RemoteAddr)

	if err != nil {
		return netip.Addr{}, err
	}

	if nil == u.trusted || false == u.trusted.Contains(remote.Addr()) {
		return remote.Addr(), nil
	}

	var list = u.getAddressesFromList(request.Header.Values("x-forwarded-for"))
	var local = u.getLocalPrefixList()

	for i := len(list) - 1; i >= 0; i-- {
		if list[i].IsValid() && false == u.trusted.Contains(list[i]) && false == local.Contains(list[i]) {
			return list[i], nil
		}
	}

	return remote.Addr(), nil
}

func (u *UpdateHandler) getLocalPrefixList() *IPPrefixList {
	if nil == u.locals {

		var locals IPPrefixList = make([]netip.Prefix, 3)

		// see https://datatracker.ietf.org/doc/html/rfc1918
		for idx, x := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
			if prefix, _ := netip.ParsePrefix(x); prefix.IsValid() {
				locals[idx] = prefix
			}
		}

		u.locals = &locals
	}

	return u.locals
}

func (u *UpdateHandler) getAddressesFromList(list []string) []netip.Addr {

	if 0 == len(list) {
		return nil
	}

	var items = make([]netip.Addr, 0)

	for i, c := 0, len(list); i < c; i++ {

		var values = strings.Split(list[i], ",")

		for x, y := 0, len(values); x < y; x++ {
			// ignore error for now and check later with IsValid
			item, _ := netip.ParseAddr(strings.TrimSpace(values[x]))
			items = append(items, item)
		}
	}

	return items
}

func (u *UpdateHandler) updateRecords(ctx context.Context, hosts []string, result *UpdateResult, items map[string][]libdns.Record, provider Provider, ip *netip.Addr, lock sync.Locker) {

	defer lock.Unlock()

	for zone, records := range items {
		sets, err := provider.SetRecords(ctx, zone, records)

		if err != nil {
			u.logger.Error(fmt.Sprintf("failed updating records for zone %s: %s", zone, err.Error()))
			u.setResponses(hosts, result, zone, records, "dnserr")
			continue
		}

		if len(sets) > 0 {
			u.setResponses(hosts, result, zone, sets, fmt.Sprintf("good %s", ip))
		}
	}
}

func (u *UpdateHandler) setResponses(hosts []string, result *UpdateResult, zone string, items []libdns.Record, value string) {
	for _, item := range items {
		if x := u.getHostIdx(hosts, item.RR().Name, zone); x != -1 {
			result.Set(x, value)
		}
	}
}

func (u *UpdateHandler) getHostIdx(hosts []string, name string, zone string) int {

	var hostname = strings.TrimSuffix(libdns.AbsoluteName(name, zone), ".")

	for i, c := 0, len(hosts); i < c; i++ {
		if hostname == hosts[i] {
			return i
		}
	}

	return -1
}
