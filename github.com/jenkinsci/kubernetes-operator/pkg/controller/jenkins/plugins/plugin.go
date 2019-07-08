package plugins

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/pkg/errors"
)

// Plugin represents jenkins plugin
type Plugin struct {
	Name                     string `json:"name"`
	Version                  string `json:"version"`
	rootPluginNameAndVersion string
}

func (p Plugin) String() string {
	return fmt.Sprintf("%s:%s", p.Name, p.Version)
}

var (
	namePattern    = regexp.MustCompile(`^[0-9a-z-]+$`)
	versionPattern = regexp.MustCompile(`^[0-9\\.]+$`)
)

// New creates plugin from string, for example "name-of-plugin:0.0.1"
func New(nameWithVersion string) (*Plugin, error) {
	val := strings.SplitN(nameWithVersion, ":", 2)
	if val == nil || len(val) != 2 {
		return nil, errors.Errorf("invalid plugin format '%s'", nameWithVersion)
	}
	name := val[0]
	version := val[1]

	if err := validatePlugin(name, version); err != nil {
		return nil, err
	}

	return &Plugin{
		Name:    name,
		Version: version,
	}, nil
}

// NewPlugin creates plugin from name and version, for example "name-of-plugin:0.0.1"
func NewPlugin(name, version string) (*Plugin, error) {
	if err := validatePlugin(name, version); err != nil {
		return nil, err
	}

	return &Plugin{
		Name:    name,
		Version: version,
	}, nil
}

func validatePlugin(name, version string) error {
	if ok := namePattern.MatchString(name); !ok {
		return errors.Errorf("invalid plugin name '%s:%s', must follow pattern '%s'", name, version, namePattern.String())
	}
	if ok := versionPattern.MatchString(version); !ok {
		return errors.Errorf("invalid plugin version '%s:%s', must follow pattern '%s'", name, version, versionPattern.String())
	}
	return nil
}

// Must returns plugin from pointer and throws panic when error is set
func Must(plugin *Plugin, err error) Plugin {
	if err != nil {
		panic(err)
	}

	return *plugin
}

// VerifyDependencies checks if all plugins have compatible versions
func VerifyDependencies(values ...map[Plugin][]Plugin) bool {
	// key - plugin name, value array of versions
	allPlugins := make(map[string][]Plugin)
	valid := true

	for _, value := range values {
		for rootPlugin, plugins := range value {
			allPlugins[rootPlugin.Name] = append(allPlugins[rootPlugin.Name], Plugin{
				Name:                     rootPlugin.Name,
				Version:                  rootPlugin.Version,
				rootPluginNameAndVersion: rootPlugin.String()})
			for _, plugin := range plugins {
				allPlugins[plugin.Name] = append(allPlugins[plugin.Name], Plugin{
					Name:                     plugin.Name,
					Version:                  plugin.Version,
					rootPluginNameAndVersion: rootPlugin.String()})
			}
		}
	}

	for pluginName, versions := range allPlugins {
		if len(versions) == 1 {
			continue
		}

		for _, firstVersion := range versions {
			for _, secondVersion := range versions {
				if firstVersion.Version != secondVersion.Version {
					log.Log.V(log.VWarn).Info(fmt.Sprintf("Plugin '%s' requires version '%s' but plugin '%s' requires '%s' for plugin '%s'",
						firstVersion.rootPluginNameAndVersion,
						firstVersion.Version,
						secondVersion.rootPluginNameAndVersion,
						secondVersion.Version,
						pluginName,
					))
					valid = false
				}
			}
		}
	}

	return valid
}
