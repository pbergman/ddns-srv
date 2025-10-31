package main

import (
	"encoding/json"
	"fmt"

	"github.com/libdns/libdns"
	"github.com/pbergman/logger"
	"github.com/pbergman/provider"
)

func bootstrap(log *logger.Logger, file string, level provider.OutputLevel) (*Config, map[string]ZoneAwareProvider, error) {

	log.Debug(fmt.Sprintf("reading config file '%s'", file))

	config, err := ReadConfig(file)

	if err != nil {
		return nil, nil, err
	}

	plugins, err := ReadPluginFiles(log, config.PluginDir)

	if err != nil {
		return nil, nil, err
	}

	var providers = make(map[string]ZoneAwareProvider, 0)

	for i, c := 0, len(config.Plugins); i < c; i++ {
		var base PluginConfig

		if err := json.Unmarshal(config.Plugins[i], &base); err != nil {
			return nil, nil, err
		}

		ref, ok := plugins[base.Module]

		if !ok {
			return nil, nil, fmt.Errorf("no plugin loaded for module: %s", base.Module)
		}

		var object = ref.New()

		if err := json.Unmarshal(config.Plugins[i], object); err != nil {
			return nil, nil, err
		}

		if _, ok := object.(libdns.ZoneLister); !ok && len(base.Zones) == 0 {
			return nil, nil, fmt.Errorf("could not determin zones for module %s", base.Module)
		}

		if len(base.Zones) > 0 {
			object = NewZoneProvider(object, base.Zones...)
		}

		if v, ok := object.(provider.DebugAware); ok {
			v.SetDebug(level, log.WithName(base.Module).NewWriter(logger.Debug))
		}

		providers[base.Module] = object.(ZoneAwareProvider)
	}

	return config, providers, nil
}
