#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../" && pwd)"

PLUGINS=(
    "configuration-as-code"
    "git"
    "job-dsl"
    "kubernetes"
    "kubernetes-credentials-provider"
    "workflow-aggregator"
)

version_compare() {
    local ver1="$1"
    local ver2="$2"

    ver1=$(echo "$ver1" | sed 's/\([0-9.]*\).*/\1/')
    ver2=$(echo "$ver2" | sed 's/\([0-9.]*\).*/\1/')

    if [ "$ver1" = "$ver2" ]; then
        return 0
    fi

    local sorted=$(printf '%s\n%s' "$ver1" "$ver2" | sort -V)
    local first=$(echo "$sorted" | head -n1)

    if [ "$first" = "$ver1" ]; then
        return 1
    else
        return 2
    fi
}

fetch_update_center_data() {
    local jenkins_version="$1"
    local url="https://updates.jenkins.io/update-center.actual.json?version=${jenkins_version}"
    local data=$(curl -s -L -f "$url")

    if ! echo "$data" | jq . > /dev/null 2>&1; then
        echo "error something happened" >&2
        return 1
    fi

    echo "$data"
}

find_compatible_version() {
    local plugin_id="$1"
    local jenkins_version="$2"
    local update_data="$3"
    local plugin_data=$(echo "$update_data" | jq ".plugins[\"$plugin_id\"]")
    local plugin_version=$(echo "$plugin_data" | jq -r '.version')
    local compatible_since=$(echo "$plugin_data" | jq -r '.compatibleSinceVersion // empty')

    if [ -n "$compatible_since" ]; then
        version_compare "$jenkins_version" "$compatible_since"
        local result=$?
        if [ $result -eq 1 ]; then
            echo "Error: Plugin '$plugin_id' requires Jenkins $compatible_since or newer" >&2
            return 1
        fi
    fi

    echo "$plugin_version"
}

update_go_file() {
    local file_path="$1"
    local plugin_id="$2"
    local new_version="$3"
    local dry_run="$4"

    local var_name=""
    case "$plugin_id" in
        "configuration-as-code") var_name="configurationAsCodePlugin" ;;
        "git") var_name="gitPlugin" ;;
        "job-dsl") var_name="jobDslPlugin" ;;
        "kubernetes") var_name="kubernetesPlugin" ;;
        "kubernetes-credentials-provider") var_name="kubernetesCredentialsProviderPlugin" ;;
        "workflow-aggregator") var_name="workflowAggregatorPlugin" ;;
    esac

    local relative_path=$(realpath --relative-to="$PROJECT_ROOT" "$file_path")
    local new_line="	${var_name} = \"${plugin_id}:${new_version}\""

    if [ "$dry_run" = "true" ]; then
        echo "$relative_path: $plugin_id:$new_version"
    else
        sed -i "s|^[[:space:]]*${var_name}[[:space:]]*=.*|${new_line}|" "$file_path"
        echo "$relative_path: $var_name -> $plugin_id:$new_version"
    fi
}

main() {
    if [ $# -lt 1 ]; then
        echo "usage: $0 <jenkins-version> [--dry-run]" >&2
        exit 1
    fi

    local jenkins_version="$1"
    local dry_run="false"

    if [ $# -gt 1 ] && [ "$2" = "--dry-run" ]; then
        dry_run="true"
    fi

    local update_data
    if ! update_data=$(fetch_update_center_data "$jenkins_version"); then
        exit 1
    fi

    for plugin_id in "${PLUGINS[@]}"; do
        local compatible_version
        if compatible_version=$(find_compatible_version "$plugin_id" "$jenkins_version" "$update_data"); then
            #printf "%-35s %-30s\n" "$plugin_id" "$compatible_version"
            update_go_file "$PROJECT_ROOT/pkg/plugins/base_plugins.go" "$plugin_id" "$compatible_version" "$dry_run"
            update_go_file "$PROJECT_ROOT/test/e2e/configuration_test.go" "$plugin_id" "$compatible_version" "$dry_run"
        fi
    done
}

main "$@"
