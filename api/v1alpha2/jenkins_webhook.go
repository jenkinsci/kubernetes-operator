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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/dlclark/regexp2/v2"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	jenkinslog                              = logf.Log.WithName("jenkins-resource") // log is for logging in this package.
	SecValidator                            = *NewSecurityValidator()
	initialSecurityWarningsDownloadSucceded = false
	securityWarningsMutex                   sync.RWMutex
)

const (
	Hosturl                 = "https://updates.jenkins.io/update-center.actual.json"
	warningMatchTimeout     = 100 * time.Millisecond
	shortenedCheckingPeriod = 1 * time.Hour
	defaultCheckingPeriod   = 12 * time.Minute
)

func (in *Jenkins) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, in).
		WithValidator(&JenkinsCustomValidator{}).
		Complete()
}

type JenkinsCustomValidator struct{}

var _ admission.Validator[*Jenkins] = &JenkinsCustomValidator{}

func (*JenkinsCustomValidator) ValidateCreate(_ context.Context, obj *Jenkins) (admission.Warnings, error) {
	if obj == nil {
		return nil, fmt.Errorf("expected a Jenkins resource, got %T", obj)
	}
	return obj.ValidateCreate()
}

func (*JenkinsCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *Jenkins) (admission.Warnings, error) {
	if newObj == nil {
		return nil, fmt.Errorf("expected a Jenkins resource, got %T", newObj)
	}
	return newObj.ValidateUpdate(oldObj)
}

func (*JenkinsCustomValidator) ValidateDelete(_ context.Context, obj *Jenkins) (admission.Warnings, error) {
	if obj == nil {
		return nil, fmt.Errorf("expected a Jenkins resource, got %T", obj)
	}
	return obj.ValidateDelete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-jenkins-io-jenkins-io-v1alpha2-jenkins,mutating=false,failurePolicy=fail,sideEffects=None,groups=jenkins.io.jenkins.io,resources=jenkins,verbs=create;update,versions=v1alpha2,name=vjenkins.kb.io,admissionReviewVersions={v1}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Jenkins) ValidateCreate() (admission.Warnings, error) {
	if in.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate create", "name", in.Name)
		err := Validate(*in)
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Jenkins) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	if in.Spec.ValidateSecurityWarnings {
		jenkinslog.Info("validate update", "name", in.Name)
		return nil, Validate(*in)
	}

	return nil, nil
}

func (in *Jenkins) ValidateDelete() (admission.Warnings, error) {
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
	Pattern      string `json:"pattern"`
	FirstVersion string `json:"firstVersion"`
	LastVersion  string `json:"lastVersion"`
}

type PluginData struct {
	Version string
	Kind    string
}

// Validates security warnings for both updating and creating a Jenkins CR
func Validate(r Jenkins) error {
	securityWarningsMutex.RLock()
	isCached, pluginDataCache := SecValidator.isCached, SecValidator.PluginDataCache
	securityWarningsMutex.RUnlock()
	if !isCached {
		return errors.New("plugins data has not been fetched")
	}

	pluginSet := make(map[string][]PluginData)
	var faultyBasePlugins string
	var faultyUserPlugins string
	basePlugins := plugins.BasePlugins()

	for _, plugin := range basePlugins {
		pluginSet[plugin.Name] = append(pluginSet[plugin.Name], PluginData{Version: plugin.Version, Kind: "base"})
	}

	for _, plugin := range r.Spec.Master.Plugins {
		// Check each requested version; Jenkins plugin versions are not semantic versions.
		if !slices.ContainsFunc(pluginSet[plugin.Name], func(data PluginData) bool { return data.Version == plugin.Version }) {
			pluginSet[plugin.Name] = append(pluginSet[plugin.Name], PluginData{Version: plugin.Version, Kind: "user-defined"})
		}
	}

	for _, plugin := range pluginDataCache.Plugins {
		for _, pluginData := range pluginSet[plugin.Name] {
			var hasVulnerabilities bool
			for _, warning := range plugin.SecurityWarnings {
				// checking if this warning applies to our version as well
				affected, err := warning.affectsVersion(pluginData.Version)
				if err != nil {
					return fmt.Errorf("cannot evaluate security warning %s for %s:%s: %w", warning.ID, plugin.Name, pluginData.Version, err)
				}
				if affected {
					jenkinslog.Info("Security Vulnerability detected in "+pluginData.Kind+" "+plugin.Name+":"+pluginData.Version, "Warning message", warning.Message, "For more details,check security advisory", warning.URL)
					hasVulnerabilities = true
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
	in.checkingPeriod = defaultCheckingPeriod

	// should only be executed once when the operator starts
	if !initialSecurityWarningsDownloadSucceded {
		securityWarningsFetched <- true
		initialSecurityWarningsDownloadSucceded = true
	}
}

// Downloads and reads the update-center JSON data.
func (in *SecurityValidator) fetchPluginData() error {
	jenkinslog.Info("Initializing/Updating the plugin data cache")
	var err error
	for in.Attempts = 0; in.Attempts < 5; in.Attempts++ {
		var candidate PluginsInfo
		candidate, err = in.download()
		if err != nil {
			jenkinslog.V(log.VDebug).Info("Cache Plugin Data", "failed to download plugin data", err)
			continue
		}
		securityWarningsMutex.Lock()
		in.PluginDataCache = candidate
		in.isCached = true
		securityWarningsMutex.Unlock()
		return nil
	}
	return err
}

func (in *SecurityValidator) download() (PluginsInfo, error) {
	req, err := http.NewRequest(http.MethodGet, Hosturl, nil)
	if err != nil {
		return PluginsInfo{}, err
	}
	req.Header.Set("Accept", "application/json")

	client := http.Client{Timeout: 1 * time.Minute}
	response, err := client.Do(req)
	if err != nil {
		return PluginsInfo{}, err
	}
	defer httpResponseCloser(response)
	if response.StatusCode != http.StatusOK {
		return PluginsInfo{}, fmt.Errorf("download plugin data: HTTP %s", response.Status)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return PluginsInfo{}, fmt.Errorf("read plugin data: %w", err)
	}
	return parsePluginWarnings(data)
}

// Loads the JSON data into memory and stores it in a candidate cache.
func parsePluginWarnings(data []byte) (PluginsInfo, error) {
	var metadata struct {
		Warnings []struct {
			Warning
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return PluginsInfo{}, fmt.Errorf("decode update-center JSON: %w", err)
	}
	if metadata.Warnings == nil {
		return PluginsInfo{}, errors.New("update-center JSON has no warnings array")
	}

	candidate := PluginsInfo{}
	pluginIndexes := make(map[string]int)
	for _, warning := range metadata.Warnings {
		if warning.Type != "plugin" {
			continue
		}
		if warning.Name == "" {
			return PluginsInfo{}, fmt.Errorf("security warning %s has no plugin name", warning.ID)
		}
		for _, version := range warning.Versions {
			if _, err := compileWarningPattern(version.Pattern); err != nil {
				return PluginsInfo{}, fmt.Errorf("invalid pattern for security warning %s: %w", warning.ID, err)
			}
		}
		index, exists := pluginIndexes[warning.Name]
		if !exists {
			index = len(candidate.Plugins)
			pluginIndexes[warning.Name] = index
			candidate.Plugins = append(candidate.Plugins, PluginInfo{Name: warning.Name})
		}
		candidate.Plugins[index].SecurityWarnings = append(candidate.Plugins[index].SecurityWarnings, warning.Warning)
	}
	return candidate, nil
}

func compileWarningPattern(pattern string) (*regexp2.Regexp, error) {
	if pattern == "" {
		return nil, errors.New("missing version pattern")
	}
	// Jenkins uses Matcher.matches(), which requires the entire version to match.
	compiled, err := regexp2.Compile(`\A(?:` + pattern + `)\z`)
	if err != nil {
		return nil, err
	}
	compiled.MatchTimeout = warningMatchTimeout
	return compiled, nil
}

func (in Warning) affectsVersion(version string) (bool, error) {
	if len(in.Versions) == 0 {
		return true, nil
	}
	for _, affected := range in.Versions {
		pattern, err := compileWarningPattern(affected.Pattern)
		if err != nil {
			return false, err
		}
		matches, err := pattern.MatchString(version)
		if err != nil {
			return false, err
		}
		if matches {
			return true, nil
		}
	}
	return false, nil
}

func httpResponseCloser(response *http.Response) {
	if err := response.Body.Close(); err != nil {
		log.Log.Error(err, "failed to close http response body")
	}
}
