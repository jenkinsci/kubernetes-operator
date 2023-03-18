package base

import (
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"

	"github.com/bndr/gojenkins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"
	stackerr "github.com/pkg/errors"
)

func (r *JenkinsBaseConfigurationReconciler) verifyPlugins(jenkinsClient jenkinsclient.Jenkins) (bool, error) {
	allPluginsInJenkins, err := jenkinsClient.GetPlugins(fetchAllPlugins)
	if err != nil {
		return false, stackerr.WithStack(err)
	}

	var installedPlugins []string
	for _, jenkinsPlugin := range allPluginsInJenkins.Raw.Plugins {
		if isValidPlugin(jenkinsPlugin) {
			installedPlugins = append(installedPlugins, plugins.Plugin{Name: jenkinsPlugin.ShortName, Version: jenkinsPlugin.Version}.String())
		}
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Installed plugins '%+v'", installedPlugins))

	status := true
	allRequiredPlugins := [][]v1alpha2.Plugin{r.Configuration.Jenkins.Spec.Master.BasePlugins, r.Configuration.Jenkins.Spec.Master.Plugins}
	for _, requiredPlugins := range allRequiredPlugins {
		for _, plugin := range requiredPlugins {
			if _, ok := isPluginInstalled(allPluginsInJenkins, plugin); !ok {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s'", plugin))
				status = false
				continue
			}
			if found, ok := isPluginVersionCompatible(allPluginsInJenkins, plugin); !ok {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("The plugin you specified as code is incompatible with this jenkins version: plugin '%s' version, actual '%+v'", plugin, found.Version))
			}
		}
	}

	return status, nil
}

func isPluginVersionCompatible(plugins *gojenkins.Plugins, plugin v1alpha2.Plugin) (gojenkins.Plugin, bool) {
	p := plugins.Contains(plugin.Name)
	if p == nil {
		return gojenkins.Plugin{}, false
	}

	return *p, p.Version == plugin.Version
}

func isValidPlugin(plugin gojenkins.Plugin) bool {
	return plugin.Active && plugin.Enabled && !plugin.Deleted
}

func isPluginInstalled(plugins *gojenkins.Plugins, requiredPlugin v1alpha2.Plugin) (gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return gojenkins.Plugin{}, false
	}

	return *p, isValidPlugin(*p)
}
