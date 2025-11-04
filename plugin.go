package main

import (
	"debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"runtime/debug"

	"github.com/pbergman/logger"
)

type Plugin struct {
	ref   reflect.Value
	build *debug.Module
}

func (p *Plugin) New() BaseProvider {
	return reflect.New(p.ref.Elem().Type().Elem()).Interface().(BaseProvider)
}

func lookupProvider(plugin *plugin.Plugin, symbolName string, name string) (*Plugin, error) {
	symbol, err := lookup(plugin, symbolName, name)

	if err != nil {
		return nil, err
	}

	var value = reflect.ValueOf(symbol)

	if value.Elem().Type().Implements(reflect.TypeOf((*BaseProvider)(nil)).Elem()) {
		return &Plugin{ref: value}, nil
	}

	return nil, fmt.Errorf("symbol %s found in plugin %s but is not a valid BaseProvider", symbolName, filepath.Base(name))
}

func lookup(plugin *plugin.Plugin, symbolName string, name string) (plugin.Symbol, error) {

	symbol, err := plugin.Lookup(symbolName)

	if err != nil {
		return nil, errors.Join(fmt.Errorf("could not find symbol %s for plugin %s", symbolName, filepath.Base(name)), err)
	}

	return symbol, nil
}

func loadPlugin(path string) (*Plugin, error) {

	fd, err := plugin.Open(path)

	if err != nil {
		return nil, err
	}

	provider, err := lookupProvider(fd, "Plugin", path)

	if err != nil {
		return nil, err
	}

	info, err := buildinfo.ReadFile(path)

	if err != nil {
		return nil, err
	}

	var build *debug.Module
	var pkg = provider.ref.Elem().Type().Elem().PkgPath()

	for _, dep := range info.Deps {
		if dep.Path == pkg {
			build = dep
			break
		}
	}

	if nil == build {
		return nil, fmt.Errorf("could not find build info for %s", pkg)
	}

	provider.build = build

	return provider, nil
}

func isValidElfFile(name string, logger *logger.Logger) bool {

	fd, err := os.Open(name)

	if err != nil {
		logger.Error(err)
		return false
	}

	defer fd.Close()

	stat, err := fd.Stat()

	if err != nil {
		logger.Error(err)
		return false
	}

	if 0 == (stat.Mode()&os.ModeType) || 0 != (stat.Mode()&os.ModeSymlink) {

		if stat.Size() < 4 {
			logger.Notice(fmt.Sprintf("skipping %s, file to small", filepath.Base(name)))
			return false
		}

		var buf = make([]byte, 4)

		if _, err := fd.Read(buf); err != nil {
			logger.Notice(err)
			return false
		}

		if buf[0] != 0x7f || buf[1] != 0x45 || buf[2] != 0x4c || buf[3] != 0x46 {
			logger.Notice(fmt.Sprintf("skipping %s, invalid file identification [0x%0x 0x%0x 0x%0x 0x%0x]", filepath.Base(name), buf[0], buf[1], buf[2], buf[3]))
			return false
		}

		return true
	} else {
		logger.Notice(fmt.Sprintf("skipping %s, not a valid file", filepath.Base(name)))
	}

	return false
}

func ReadPluginFiles(logger *logger.Logger, root string) ([]*Plugin, error) {

	matches, err := filepath.Glob(filepath.Join(root, "*.so"))

	if err != nil {
		return nil, err
	}

	var plugins = make([]*Plugin, 0)

	for i, c := 0, len(matches); i < c; i++ {
		if isValidElfFile(matches[i], logger) {
			x, err := loadPlugin(matches[i])

			if err != nil {
				logger.Notice(fmt.Sprintf("loading plugin %s failed: %s", filepath.Base(matches[i]), err.Error()))
				continue
			}

			logger.Debug(fmt.Sprintf("loaded plugin %s (%s) from '%s'", x.build.Path, x.build.Version, filepath.Base(matches[i])))

			plugins = append(plugins, x)
		}
	}

	return plugins, nil
}
