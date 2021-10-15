/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"

	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	jenkinslog                   = logf.Log.WithName("jenkins-resource") // log is for logging in this package.
	PluginsMgr PluginDataManager = *NewPluginsDataManager("/tmp/plugins.json.gzip", "/tmp/plugins.json", false, time.Duration(1000)*time.Second)
	_          webhook.Validator = &Jenkins{}
)

const Hosturl = "https://ci.jenkins.io/job/Infra/job/plugin-site-api/job/generate-data/lastSuccessfulBuild/artifact/plugins.json.gzip"

func (in *Jenkins) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-jenkins-io-jenkins-io-v1alpha2-jenkins,mutating=false,failurePolicy=fail,sideEffects=None,groups=jenkins.io.jenkins.io,resources=jenkins,verbs=create;update,versions=v1alpha2,name=vjenkins.kb.io,admissionReviewVersions={v1,v1beta1}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Jenkins) ValidateCreate() error {
	if in.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate create", "name", in.Name)
		return Validate(*in)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Jenkins) ValidateUpdate(old runtime.Object) error {
	if in.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate update", "name", in.Name)
		return Validate(*in)
	}

	return nil
}

func (in *Jenkins) ValidateDelete() error {
	return nil
}

type PluginDataManager struct {
	PluginDataCache    PluginsInfo
	Timeout            time.Duration
	CompressedFilePath string
	PluginDataFile     string
	IsCached           bool
	Attempts           int
	SleepTime          time.Duration
}

type PluginsInfo struct {
	Plugins []PluginInfo `json:"plugins"`
}

type PluginInfo struct {
	Name             string    `json:"name"`
	SecurityWarnings []Warning `json:"securityWarnings"`
}

type Warning struct {
	Versions []Version `json:"versions"`
	ID       string    `json:"id"`
	Message  string    `json:"message"`
	URL      string    `json:"url"`
	Active   bool      `json:"active"`
}

type Version struct {
	FirstVersion string `json:"firstVersion"`
	LastVersion  string `json:"lastVersion"`
}

type PluginData struct {
	Version string
	Kind    string
}

// Validates security warnings for both updating and creating a Jenkins CR
func Validate(r Jenkins) error {
	if !PluginsMgr.IsCached {
		return errors.New("plugins data has not been fetched")
	}

	pluginSet := make(map[string]PluginData)
	var faultyBasePlugins string
	var faultyUserPlugins string
	basePlugins := plugins.BasePlugins()

	for _, plugin := range basePlugins {
		// Only Update the map if the plugin is not present or a lower version is being used
		if pluginData, ispresent := pluginSet[plugin.Name]; !ispresent || semver.Compare(makeSemanticVersion(plugin.Version), pluginData.Version) == 1 {
			pluginSet[plugin.Name] = PluginData{Version: plugin.Version, Kind: "base"}
		}
	}

	for _, plugin := range r.Spec.Master.Plugins {
		if pluginData, ispresent := pluginSet[plugin.Name]; !ispresent || semver.Compare(makeSemanticVersion(plugin.Version), pluginData.Version) == 1 {
			pluginSet[plugin.Name] = PluginData{Version: plugin.Version, Kind: "user-defined"}
		}
	}

	for _, plugin := range PluginsMgr.PluginDataCache.Plugins {
		if pluginData, ispresent := pluginSet[plugin.Name]; ispresent {
			var hasVulnerabilities bool
			for _, warning := range plugin.SecurityWarnings {
				for _, version := range warning.Versions {
					firstVersion := version.FirstVersion
					lastVersion := version.LastVersion
					if len(firstVersion) == 0 {
						firstVersion = "0" // setting default value in case of empty string
					}
					if len(lastVersion) == 0 {
						lastVersion = pluginData.Version // setting default value in case of empty string
					}
					// checking if this warning applies to our version as well
					if compareVersions(firstVersion, lastVersion, pluginData.Version) {
						jenkinslog.Info("Security Vulnerability detected in "+pluginData.Kind+" "+plugin.Name+":"+pluginData.Version, "Warning message", warning.Message, "For more details,check security advisory", warning.URL)
						hasVulnerabilities = true
					}
				}
			}

			if hasVulnerabilities {
				if pluginData.Kind == "base" {
					faultyBasePlugins += "\n" + plugin.Name + ":" + pluginData.Version
				} else {
					faultyUserPlugins += "\n" + plugin.Name + ":" + pluginData.Version
				}
			}
		}
	}
	if len(faultyBasePlugins) > 0 || len(faultyUserPlugins) > 0 {
		var errormsg string
		if len(faultyBasePlugins) > 0 {
			errormsg += "security vulnerabilities detected in the following base plugins: " + faultyBasePlugins
		}
		if len(faultyUserPlugins) > 0 {
			errormsg += "security vulnerabilities detected in the following user-defined plugins: " + faultyUserPlugins
		}
		return errors.New(errormsg)
	}

	return nil
}

func NewPluginsDataManager(compressedFilePath string, pluginDataFile string, isCached bool, timeout time.Duration) *PluginDataManager {
	return &PluginDataManager{
		CompressedFilePath: compressedFilePath,
		PluginDataFile:     pluginDataFile,
		IsCached:           isCached,
		Timeout:            timeout,
	}
}

func (in *PluginDataManager) ManagePluginData(sig chan bool) {
	var isInit bool
	var retryInterval time.Duration
	for {
		var isCached bool
		err := in.fetchPluginData()
		if err == nil {
			isCached = true
		} else {
			jenkinslog.Info("Cache plugin data", "failed to fetch plugin data", err)
		}
		// should only be executed once when the operator starts
		if !isInit {
			sig <- isCached // sending signal to main to continue
			isInit = true
		}

		in.IsCached = in.IsCached || isCached
		if !isCached {
			retryInterval = time.Duration(1) * time.Hour
		} else {
			retryInterval = time.Duration(12) * time.Hour
		}
		time.Sleep(retryInterval)
	}
}

// Downloads extracts and reads the JSON data in every 12 hours
func (in *PluginDataManager) fetchPluginData() error {
	jenkinslog.Info("Initializing/Updating the plugin data cache")
	var err error
	for in.Attempts = 0; in.Attempts < 5; in.Attempts++ {
		err = in.download()
		if err != nil {
			jenkinslog.V(log.VDebug).Info("Cache Plugin Data", "failed to download file", err)
			continue
		}
		break
	}

	if err != nil {
		return err
	}

	for in.Attempts = 0; in.Attempts < 5; in.Attempts++ {
		err = in.extract()
		if err != nil {
			jenkinslog.V(log.VDebug).Info("Cache Plugin Data", "failed to extract file", err)
			continue
		}
		break
	}

	if err != nil {
		return err
	}

	for in.Attempts = 0; in.Attempts < 5; in.Attempts++ {
		err = in.cache()
		if err != nil {
			jenkinslog.V(log.VDebug).Info("Cache Plugin Data", "failed to read plugin data file", err)
			continue
		}
		break
	}

	return err
}

func (in *PluginDataManager) download() error {
	out, err := os.Create(in.CompressedFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	client := http.Client{
		Timeout: in.Timeout,
	}

	resp, err := client.Get(Hosturl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (in *PluginDataManager) extract() error {
	reader, err := os.Open(in.CompressedFilePath)

	if err != nil {
		return err
	}
	defer reader.Close()
	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	defer archive.Close()
	writer, err := os.Create(in.PluginDataFile)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err
}

// Loads the JSON data into memory and stores it
func (in *PluginDataManager) cache() error {
	jsonFile, err := os.Open(in.PluginDataFile)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &in.PluginDataCache)
	return err
}

// returns a semantic version that can be used for comparison, allowed versioning format vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH
func makeSemanticVersion(version string) string {
	if version[0] != 'v' {
		version = "v" + version
	}
	return semver.Canonical(version)
}

// Compare if the current version lies between first version and last version
func compareVersions(firstVersion string, lastVersion string, pluginVersion string) bool {
	firstSemVer := makeSemanticVersion(firstVersion)
	lastSemVer := makeSemanticVersion(lastVersion)
	pluginSemVer := makeSemanticVersion(pluginVersion)
	if semver.Compare(pluginSemVer, firstSemVer) == -1 || semver.Compare(pluginSemVer, lastSemVer) == 1 {
		return false
	}
	return true
}
