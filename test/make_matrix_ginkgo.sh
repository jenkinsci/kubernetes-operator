#!/usr/bin/env bash

TESTDIR="${TESTDIR:-test}"

json_output(){
    # Make shellcheck happy,
    # declare local before assign
    local lastl
    local line
    local grep_info
    local f
    local l
    local t

    lastl=$(echo "${1}" | wc -l)
    line=0
    printf '{\"include\":['
    while read -r test; do
        line=$((line + 1))
        grep_info=$(echo "${test}"|awk -F '"' '{print $1}')
        f=$(echo "${grep_info}"|cut -d ':' -f 1)
        l=$(echo "${grep_info}"|cut -d ':' -f 2)
        t=$(echo "${test}"|awk -F '"' '{print $2}')
        printf '{\"file\":\"%s\",\"line\":\"%s\",\"test\":\"%s\"}' "$f" "$l" "$t"
        [[ $line -ne $lastl ]] && printf ","
    done <<< "${1}"
    printf "]}"
}

parse(){
    grep -nrE 'It\([^)]+\)' "$1"
}

tests_list=$(parse "${TESTDIR}"/"${1}")
json_output "${tests_list}"
