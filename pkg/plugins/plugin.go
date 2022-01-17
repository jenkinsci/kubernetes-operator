package plugins

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Plugin represents jenkins plugin.
type Plugin struct {
	Name                     string `json:"name"`
	Version                  string `json:"version"`
	DownloadURL              string `json:"downloadURL"`
	rootPluginNameAndVersion string
}

func (p Plugin) String() string {
	return fmt.Sprintf("%s:%s", p.Name, p.Version)
}

var (
	// NamePattern is the plugin name regex pattern
	NamePattern = regexp.MustCompile(`^[0-9a-zA-Z-_]+$`)
	// VersionPattern is the plugin version regex pattern
	VersionPattern = regexp.MustCompile(`^[0-9a-zA-Z+\\.-_]+$`)
	// DownloadURLPattern is the plugin download url regex pattern
	DownloadURLPattern = regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)
)

// New creates plugin from string, for example "name-of-plugin:0.0.1".
func New(nameWithVersion string) (*Plugin, error) {
	val := strings.SplitN(nameWithVersion, ":", 2)
	if val == nil || len(val) != 2 {
		return nil, errors.Errorf("invalid plugin format '%s'", nameWithVersion)
	}
	name := val[0]
	version := val[1]

	if err := validatePlugin(name, version, ""); err != nil {
		return nil, err
	}

	return &Plugin{
		Name:    name,
		Version: version,
	}, nil
}

// NewPlugin creates plugin from name and version, for example "name-of-plugin:0.0.1".
func NewPlugin(name, version, downloadURL string) (*Plugin, error) {
	if err := validatePlugin(name, version, downloadURL); err != nil {
		return nil, err
	}

	return &Plugin{
		Name:        name,
		Version:     version,
		DownloadURL: downloadURL,
	}, nil
}

func validatePlugin(name, version, downloadURL string) error {
	if ok := NamePattern.MatchString(name); !ok {
		return errors.Errorf("invalid plugin name '%s:%s', must follow pattern '%s'", name, version, NamePattern.String())
	}
	if ok := VersionPattern.MatchString(version); !ok {
		return errors.Errorf("invalid plugin version '%s:%s', must follow pattern '%s'", name, version, VersionPattern.String())
	}
	if len(downloadURL) > 0 {
		if ok := DownloadURLPattern.MatchString(downloadURL); !ok {
			return errors.Errorf("invalid download URL '%s' for plugin name %s:%s, must follow pattern '%s'", downloadURL, name, version, DownloadURLPattern.String())
		}
	}
	return nil
}

// Must returns plugin from pointer and throws panic when error is set.
func Must(plugin *Plugin, err error) Plugin {
	if err != nil {
		panic(err)
	}

	return *plugin
}

// VerifyDependencies checks if all plugins have compatible versions.
func VerifyDependencies(values ...map[Plugin][]Plugin) []string {
	var messages []string
	// key - plugin name, value array of versions
	allPlugins := make(map[string][]Plugin)

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
					messages = append(messages, fmt.Sprintf("Plugin '%s' requires version '%s' but plugin '%s' requires '%s' for plugin '%s'",
						firstVersion.rootPluginNameAndVersion,
						firstVersion.Version,
						secondVersion.rootPluginNameAndVersion,
						secondVersion.Version,
						pluginName,
					))
				}
			}
		}
	}

	return messages
}
