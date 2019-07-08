#!/bin/bash
set -e

FMT="%-40s%-14s %-14s %-16s %s\n"

function main() {
    printf "$FMT" "PLUGIN ID" "LOCAL VERSION" "LATEST" "" "NEW PLUGIN:VERSION"

    # Copy all the const values from base_plugins.go as the parameter for getLatest
    # (column-select is great here; alt+shift+arrow in vscode)
	getLatest "ace-editor:1.1"
	getLatest "apache-httpcomponents-client-4-api:4.5.5-3.0"
	getLatest "authentication-tokens:1.3"
	getLatest "branch-api:2.5.1"
	getLatest "cloudbees-folder:6.8"
	getLatest "configuration-as-code:1.16"
	getLatest "configuration-as-code-support:1.16"
	getLatest "credentials-binding:1.18"
	getLatest "credentials:2.1.19"
	getLatest "display-url-api:2.3.1"
	getLatest "docker-commons:1.15"
	getLatest "docker-workflow:1.18"
	getLatest "durable-task:1.29"
	getLatest "git-client:2.7.7"
	getLatest "git:3.10.0"
	getLatest "git-server:1.7"
	getLatest "handlebars:1.1.1"
	getLatest "jackson2-api:2.9.9"
	getLatest "job-dsl:1.74"
	getLatest "jquery-detached:1.2.1"
	getLatest "jsch:0.1.55"
	getLatest "junit:1.28"
	getLatest "kubernetes-credentials:0.4.0"
	getLatest "kubernetes-credentials-provider:0.12.1"
	getLatest "kubernetes:1.15.5"
	getLatest "lockable-resources:2.5"
	getLatest "mailer:1.23"
	getLatest "matrix-project:1.14"
	getLatest "momentjs:1.1.1"
	getLatest "pipeline-build-step:2.9"
	getLatest "pipeline-graph-analysis:1.10"
	getLatest "pipeline-input-step:2.10"
	getLatest "pipeline-milestone-step:1.3.1"
	getLatest "pipeline-model-api:1.3.8"
	getLatest "pipeline-model-declarative-agent:1.1.1"
	getLatest "pipeline-model-definition:1.3.8"
	getLatest "pipeline-model-extensions:1.3.8"
	getLatest "pipeline-rest-api:2.11"
	getLatest "pipeline-stage-step:2.3"
	getLatest "pipeline-stage-tags-metadata:1.3.8"
	getLatest "pipeline-stage-view:2.11"
	getLatest "plain-credentials:1.5"
	getLatest "scm-api:2.4.1"
	getLatest "script-security:1.59"
	getLatest "ssh-credentials:1.16"
	getLatest "structs:1.19"
	getLatest "variant:1.2"
	getLatest "workflow-aggregator:2.6"
	getLatest "workflow-api:2.34"
	getLatest "workflow-basic-steps:2.16"
	getLatest "workflow-cps-global-lib:2.13"
	getLatest "workflow-cps:2.68"
	getLatest "workflow-durable-task-step:2.30"
	getLatest "workflow-job:2.32"
	getLatest "workflow-multibranch:2.21"
	getLatest "workflow-scm-step:2.7"
	getLatest "workflow-step-api:2.19"
	getLatest "workflow-support:3.3"
}

# Usage:
# getLatest "plugin-id:current-version"
function getLatest() {
    local pluginId="$(echo "$1" | cut -d: -f1)"
    local localVersion="$(echo "$1" | cut -d: -f2)"
    local version="$(curl -s https://plugins.jenkins.io/$pluginId \
        | sed -n 's/.*class="v" data-reactid="18">\([^<]*\).*/\1/p')"
    if [ "$localVersion" = "$version" ]; then
        changed=""
    else
        changed="UPDATE AVAILABLE"
    fi
    printf "$FMT" "$pluginId" "$localVersion" "$version" "$changed" "$pluginId:$version"
}

main
