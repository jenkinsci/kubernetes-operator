package resources

import (
	"fmt"
	"text/template"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/internal/render"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const installPluginsCommand = "jenkins-plugin-cli"

var initBashTemplate = template.Must(template.New(InitScriptName).Parse(`#!/usr/bin/env bash
set -e
set -x

if [ "${DEBUG_JENKINS_OPERATOR}" == "true" ]; then
	echo "Printing debug messages - begin"
	id
	env
	ls -la {{ .JenkinsHomePath }}
	echo "Printing debug messages - end"
else
    echo "To print debug messages set environment variable 'DEBUG_JENKINS_OPERATOR' to 'true'"
fi

# https://wiki.jenkins.io/display/JENKINS/Post-initialization+script
mkdir -p {{ .JenkinsHomePath }}/init.groovy.d
cp -n {{ .InitConfigurationPath }}/*.groovy {{ .JenkinsHomePath }}/init.groovy.d

mkdir -p {{ .JenkinsHomePath }}/scripts
cp {{ .JenkinsScriptsVolumePath }}/*.sh {{ .JenkinsHomePath }}/scripts
chmod +x {{ .JenkinsHomePath }}/scripts/*.sh

{{if not .SkipPlugins }}
{{- $jenkinsHomePath := .JenkinsHomePath }}
{{- $installPluginsCommand := .InstallPluginsCommand }}

echo "Installing plugins required by Operator - begin"
cat > {{ .JenkinsHomePath }}/base-plugins.txt << EOF
{{ range $index, $plugin := .BasePlugins }}
{{ $plugin.Name }}:{{ $plugin.Version }}{{if $plugin.DownloadURL}}:{{ $plugin.DownloadURL }}{{end}}
{{ end }}
EOF

{{ $installPluginsCommand }} --verbose --latest {{ .LatestPlugins }} -f {{ .JenkinsHomePath }}/base-plugins.txt
echo "Installing plugins required by Operator - end"

echo "Installing plugins required by user - begin"
cat > {{ .JenkinsHomePath }}/user-plugins.txt << EOF
{{ range $index, $plugin := .UserPlugins }}
{{ $plugin.Name }}:{{ $plugin.Version }}{{if $plugin.DownloadURL}}:{{ $plugin.DownloadURL }}{{end}}
{{ end }}
EOF

{{ $installPluginsCommand }} --verbose --latest {{ .LatestPlugins }} -f {{ .JenkinsHomePath }}/user-plugins.txt
echo "Installing plugins required by user - end"
{{end}}
`))

func buildConfigMapTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	}
}

func buildInitBashScript(jenkins *v1alpha2.Jenkins) (*string, error) {
	latestP := jenkins.Spec.Master.LatestPlugins
	if latestP == nil {
		latestP = new(bool)
		*latestP = true
	}
	skipPlugins := jenkins.Spec.Master.SkipPlugins
	if skipPlugins == nil {
		skipPlugins = new(bool)
		*skipPlugins = false
	}
	data := struct {
		JenkinsHomePath          string
		InitConfigurationPath    string
		InstallPluginsCommand    string
		JenkinsScriptsVolumePath string
		BasePlugins              []v1alpha2.Plugin
		UserPlugins              []v1alpha2.Plugin
		LatestPlugins            bool
		SkipPlugins              bool
	}{
		JenkinsHomePath:          getJenkinsHomePath(jenkins),
		InitConfigurationPath:    jenkinsInitConfigurationVolumePath,
		BasePlugins:              jenkins.Spec.Master.BasePlugins,
		UserPlugins:              jenkins.Spec.Master.Plugins,
		InstallPluginsCommand:    installPluginsCommand,
		JenkinsScriptsVolumePath: JenkinsScriptsVolumePath,
		LatestPlugins:            *latestP,
		SkipPlugins:              *skipPlugins,
	}

	output, err := render.Render(initBashTemplate, data)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func getScriptsConfigMapName(jenkins *v1alpha2.Jenkins) string {
	return fmt.Sprintf("%s-scripts-%s", constants.OperatorName, jenkins.ObjectMeta.Name)
}

// NewScriptsConfigMap builds Kubernetes config map used to store scripts
func NewScriptsConfigMap(meta metav1.ObjectMeta, jenkins *v1alpha2.Jenkins) (*corev1.ConfigMap, error) {
	meta.Name = getScriptsConfigMapName(jenkins)

	initBashScript, err := buildInitBashScript(jenkins)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		TypeMeta:   buildConfigMapTypeMeta(),
		ObjectMeta: meta,
		Data: map[string]string{
			InitScriptName: *initBashScript,
		},
	}, nil
}
