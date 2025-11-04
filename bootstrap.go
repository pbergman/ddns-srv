package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/libdns/libdns"
	"github.com/pbergman/logger"
	"github.com/pbergman/provider"
)

func bootstrap(log *logger.Logger, file string, level provider.OutputLevel) (*Config, []PluginProvider, error) {

	log.Debug(fmt.Sprintf("reading config file '%s'", file))

	config, err := ReadConfig(file)

	if err != nil {
		return nil, nil, err
	}

	plugins, err := ReadPluginFiles(log, config.PluginDir)

	if err != nil {
		return nil, nil, err
	}

	var providers = make([]PluginProvider, len(plugins))

	for i, c := 0, len(config.Plugins); i < c; i++ {
		var base PluginConfig

		if err := json.Unmarshal(config.Plugins[i], &base); err != nil {
			return nil, nil, err
		}

		var ref = lookupPlugin(base.Plugin, plugins)

		if ref == nil {
			return nil, nil, fmt.Errorf("no plugin loaded for: %s", base.Plugin)
		}

		var object = ref.New()

		if err := json.Unmarshal(config.Plugins[i], object); err != nil {
			return nil, nil, err
		}

		if _, ok := object.(libdns.ZoneLister); !ok && len(base.Zones) == 0 {
			return nil, nil, fmt.Errorf("could not determin zones for plugin %s", ref.build.Path)
		}

		if len(base.Zones) > 0 {
			object = NewStaticZoneProvider(object, base.Zones...)
		}

		if v, ok := object.(provider.DebugAware); ok {
			v.SetDebug(level, log.WithName(base.Plugin).NewWriter(logger.Debug))
		}

		providers[i] = &Provider{ZoneAwareProvider: object.(ZoneAwareProvider), module: ref.build}
	}

	return config, providers, nil
}

func lookupPlugin(plugin string, plugins []*Plugin) *Plugin {
	// check it full namespace or name within the libdns repository
	// which are just names without a slash and wil prefixed with
	// github.com/libdns to find a match in the plugin list
	if -1 == strings.Index(plugin, "/") {
		plugin = "github.com/libdns/" + plugin
	}

	for idx, x := range plugins {
		if x.build.Path == plugin {
			return plugins[idx]
		}
	}

	return nil
}
