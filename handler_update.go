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

func NewUpdateHandler(plugins []PluginProvider, logger *logger.Logger, config *ServerUpdateConfig) Handler {
	return &UpdateHandler{
		plugins: plugins,
		logger:  logger,
		config:  config,
	}
}

type UpdateHandler struct {
	plugins []PluginProvider
	logger  *logger.Logger
	config  *ServerUpdateConfig
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

	if ip, err = getIp(query, request.RemoteAddr, request.Header, u.config); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	var lock = NewSemaphore(5)
	var result = NewUpdateResult(len(hosts), &ip)
	var zones [][]string

	defer result.WriteTo(response)

	if zones, err = u.fetchZones(request.Context(), lock); err != nil {
		http.Error(response, err.Error(), http.StatusFailedDependency)
		return
	}

	for idx, items := range u.makeUpdateLists(hosts, ip, zones, result) {
		lock.Lock()
		go u.updateRecords(request.Context(), hosts, result, items, u.plugins[idx], &ip, lock)
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

func (u *UpdateHandler) fetchZones(ctx context.Context, lock WaitableLocker) ([][]string, error) {

	var zones = make([][]string, len(u.plugins))
	var errList = make([]error, 0)

	for idx, plugin := range u.plugins {

		lock.Lock()

		var err error

		errList = append(errList, err)

		go u.fetchZonesForModule(ctx, lock, idx, &zones, plugin, &err)
	}

	lock.Wait()

	return zones, errors.Join(errList...)
}

func (u *UpdateHandler) fetchZonesForModule(ctx context.Context, lock sync.Locker, idx int, list *[][]string, provider ZoneAwareProvider, ref *error) {

	defer lock.Unlock()

	zones, err := provider.ListZones(ctx)

	if err != nil {
		*ref = err
		return
	}

	if nil == (*list)[idx] {
		(*list)[idx] = make([]string, 0)
	}

	for _, zone := range zones {
		(*list)[idx] = append((*list)[idx], strings.TrimSuffix(zone.Name, "."))
	}
}

func (u *UpdateHandler) makeUpdateLists(hosts []string, ip netip.Addr, zones [][]string, result *UpdateResult) map[int]map[string][]libdns.Record {

	var updates = make(map[int]map[string][]libdns.Record)

hostnames:
	for idx, hostname := range hosts {

		u.logger.Debug(fmt.Sprintf("lookup provider for hostname '%s'", hostname))

		for x, plugin := range u.plugins {

			for _, zone := range zones[x] {
				if strings.HasSuffix(hostname, "."+zone) {

					u.logger.Debug(fmt.Sprintf("hostname %s matches zone %s (module %s)", hostname, zone, plugin.Module().Path))

					if _, ok := updates[x]; !ok {
						updates[x] = make(map[string][]libdns.Record)
					}

					if _, ok := updates[x][zone]; !ok {
						updates[x][zone] = make([]libdns.Record, 0)
					}

					updates[x][zone] = append(updates[x][zone], libdns.Address{
						Name: libdns.RelativeName(hostname, zone),
						TTL:  (time.Minute * 5).Round(time.Second),
						IP:   ip,
					})

					continue hostnames
				}
			}

			u.logger.Debug(fmt.Sprintf("hostname %s is not supported by module %s (%s)", hostname, plugin.Module().Path, strings.Join(zones[x], ", ")))
		}

		result.Set(idx, "nohost")
	}

	return updates
}

func (u *UpdateHandler) updateRecords(ctx context.Context, hosts []string, result *UpdateResult, items map[string][]libdns.Record, provider BaseProvider, ip *netip.Addr, lock sync.Locker) {

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
