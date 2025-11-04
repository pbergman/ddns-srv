package main

import (
	"encoding/json"
	"os"
)

type PluginConfig struct {
	Plugin string   `json:"plugin"`
	Zones  []string `json:"zones,omitempty"`
}

type Config struct {
	PluginDir string            `json:"plugin_dir"`
	Server    *ServerConfig     `json:"server"`
	Plugins   []json.RawMessage `json:"plugins"`
}

func ReadConfig(file string) (*Config, error) {

	var config = &Config{
		PluginDir: "/usr/share/ddns-srv",
		Server: &ServerConfig{
			Listen: ":8080",

			ServerUpdateConfig: ServerUpdateConfig{
				NoLocalIp: false,
			},
		},
	}

	fd, err := os.Open(file)

	if err != nil {
		return nil, err
	}

	defer fd.Close()

	if err := json.NewDecoder(fd).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
