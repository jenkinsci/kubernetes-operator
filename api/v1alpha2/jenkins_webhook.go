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
	"context"
	"encoding/json"
	"errors"
	"io"
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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	jenkinslog                                                      = logf.Log.WithName("jenkins-resource") // log is for logging in this package.
	SecValidator                                                    = *NewSecurityValidator()
	_                                       webhook.CustomValidator = &Jenkins{}
	initialSecurityWarningsDownloadSucceded                         = false
)

const (
	Hosturl                      = "https://ci.jenkins.io/job/Infra/job/plugin-site-api/job/generate-data/lastSuccessfulBuild/artifact/plugins.json.gzip"
	PluginDataFileCompressedPath = "/tmp/plugins.json.gzip"
	PluginDataFile               = "/tmp/plugins.json"
	shortenedCheckingPeriod      = 1 * time.Hour
	defaultCheckingPeriod        = 12 * time.Minute
)

func (in *Jenkins) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-jenkins-io-jenkins-io-v1alpha2-jenkins,mutating=false,failurePolicy=fail,sideEffects=None,groups=jenkins.io.jenkins.io,resources=jenkins,verbs=create;update,versions=v1alpha2,name=vjenkins.kb.io,admissionReviewVersions={v1}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *Jenkins) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	jenkins, ok := obj.(*Jenkins)
	if !ok {
		return nil, errors.New("expected a Jenkins object")
	}
	if jenkins.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate create", "name", jenkins.Name)
		err := Validate(*jenkins)
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *Jenkins) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	jenkins, ok := newObj.(*Jenkins)
	if !ok {
		return nil, errors.New("expected a Jenkins object")
	}
	if jenkins.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate update", "name", jenkins.Name)
		return nil, Validate(*jenkins)
	}

	return nil, nil
}

func (in *Jenkins) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

type SecurityValidator struct {
	PluginDataCache PluginsInfo
	isCached        bool
	Attempts        int
	checkingPeriod  time.Duration
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
	if !SecValidator.isCached {
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

	for _, plugin := range SecValidator.PluginDataCache.Plugins {
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

// NewMonitor creates a new worker and instantiates all the data structures required
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		isCached:       false,
		Attempts:       0,
		checkingPeriod: shortenedCheckingPeriod,
	}
}

func (in *SecurityValidator) MonitorSecurityWarnings(securityWarningsFetched chan bool) {
	jenkinslog.Info("Security warnings check: enabled\n")
	for {
		in.checkForSecurityVulnerabilities(securityWarningsFetched)
		<-time.After(in.checkingPeriod)
	}
}

func (in *SecurityValidator) checkForSecurityVulnerabilities(securityWarningsFetched chan bool) {
	err := in.fetchPluginData()
	if err != nil {
		jenkinslog.Info("Cache plugin data", "failed to fetch plugin data", err)
		in.checkingPeriod = shortenedCheckingPeriod
		return
	}
	in.isCached = true
	in.checkingPeriod = defaultCheckingPeriod

	// should only be executed once when the operator starts
	if !initialSecurityWarningsDownloadSucceded {
		securityWarningsFetched <- in.isCached
		initialSecurityWarningsDownloadSucceded = true
	}
}

// Downloads extracts and reads the JSON data in every 12 hours
func (in *SecurityValidator) fetchPluginData() error {
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

func (in *SecurityValidator) download() error {
	pluginDataFileCompressed, err := os.Create(PluginDataFileCompressedPath)
	if err != nil {
		return err
	}

	// ensure pluginDataFileCompressed is closed
	defer func() {
		if err := pluginDataFileCompressed.Close(); err != nil {
			jenkinslog.V(log.VDebug).Info("Failed to close SecurityValidator.download io", "error", err)
		}
	}()

	req, err := http.NewRequest(http.MethodGet, Hosturl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	Client := http.Client{
		Timeout: 1 * time.Minute,
	}

	response, err := Client.Do(req)
	if err != nil {
		return err
	}

	defer httpResponseCloser(response)

	_, err = io.Copy(pluginDataFileCompressed, response.Body)

	return err
}

func (in *SecurityValidator) extract() error {
	reader, err := os.Open(PluginDataFileCompressedPath)

	if err != nil {
		return err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.Log.Error(err, "failed to close SecurityValidator.extract.reader ")
		}
	}()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}

	defer func() {
		if err := archive.Close(); err != nil {
			log.Log.Error(err, "failed to close SecurityValidator.extract.archive ")
		}
	}()
	writer, err := os.Create(PluginDataFile)
	if err != nil {
		return err
	}

	defer func() {
		if err := writer.Close(); err != nil {
			log.Log.Error(err, "failed to close SecurityValidator.extract.writer")
		}
	}()

	_, err = io.Copy(writer, archive)
	return err
}

// Loads the JSON data into memory and stores it
func (in *SecurityValidator) cache() error {
	jsonFile, err := os.Open(PluginDataFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := jsonFile.Close(); err != nil {
			log.Log.Error(err, "failed to close SecurityValidator.cache.jsonFile")
		}
	}()
	byteValue, err := io.ReadAll(jsonFile)
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

func httpResponseCloser(response *http.Response) {
	if err := response.Body.Close(); err != nil {
		log.Log.Error(err, "failed to close http response body")
	}
}
