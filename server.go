package main

import (
	"context"
	"net"
	"net/http"
	"net/netip"

	"github.com/pbergman/logger"
)

type UserList map[string]string

func (u UserList) Authenticate(user, pass string) bool {

	if v, ok := u[user]; !ok || v != pass {
		return false
	}

	return true
}

type IPPrefixList []netip.Prefix

func (t *IPPrefixList) Contains(ip netip.Addr) bool {
	for _, prefix := range *t {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

type ServerConfig struct {
	Users  *UserList `json:"users"`
	Listen string    `json:"listen"`
	ServerUpdateConfig
}

type ServerUpdateConfig struct {
	TrustedRemotes *IPPrefixList `json:"trusted_remotes"`
	NoLocalIp      bool          `json:"no_local_ip"`
}

func NewServer(ctx context.Context, config *Config, logger *logger.Logger, plugins []PluginProvider) *http.Server {
	return &http.Server{
		Addr: config.Server.Listen,
		Handler: NewServerHandler(
			config.Server,
			logger,
			plugins,
		),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}
