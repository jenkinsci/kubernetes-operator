#!/usr/bin/env python3

import argparse
import html
from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[2]
LATEST = 'website/content/en/docs/Getting Started/latest'


def read_plugins(source):
    constants = dict(re.findall(r'(\w+)\s*=\s*"([^"\n]+:[^"\n]+)"', source))
    match = re.search(r'var basePluginsList = \[\]Plugin\{(.*?)\n\}', source, re.S)
    if not match:
        raise ValueError('basePluginsList not found')
    entry = r'Must\(New\((\w+)\)\),'
    names = re.findall(entry, match[1])
    if not names or re.sub(entry, '', match[1]).strip():
        raise ValueError('Unrecognized basePluginsList entries')
    if any(name not in constants for name in names):
        raise ValueError('Missing base plugin constant')
    plugins = [tuple(constants[name].split(':', 1)) for name in names]
    if len({name for name, _ in plugins}) != len(plugins):
        raise ValueError('Duplicate base plugin names')
    return plugins


def replace_section(text, pattern, replacement, path):
    updated, count = re.subn(pattern, lambda _: replacement, text, flags=re.S)
    if count != 1:
        raise ValueError(f'{path}: expected one plugin section, found {count}')
    return updated


def update_docs(root, plugins, check=False):
    yaml_list = ''.join(f'- name: {name}\n  version: "{version}"\n' for name, version in plugins)
    comment_list = ''.join(f'  # {line}\n' for line in yaml_list.splitlines())
    go_list = ''.join(f'\t// - name: {name}\n\t// version: "{version}"\n' for name, version in plugins)
    example = ''.join(f'    {line}\n' for line in yaml_list.splitlines())
    schema = ''.join(f'- name: {html.escape(name)}\nversion: &ldquo;{html.escape(version)}&rdquo;\n' for name, version in plugins)
    sections = {
        'chart/jenkins-operator/values.yaml': [
            (r'  # basePlugins:\n.*?(?=\n  basePlugins: \[\])', '  # basePlugins:\n' + comment_list),
        ],
        'api/v1alpha2/jenkins_types.go': [
            (r'\t// BasePlugins contains plugins required by operator\n\t// \+optional\n\t// Defaults to :\n.*?(?=\tBasePlugins \[\]Plugin)',
             '\t// BasePlugins contains plugins required by operator\n\t// +optional\n\t// Defaults to :\n' + go_list),
        ],
        f'{LATEST}/customizing-jenkins.md': [
            (r'Pre-installed plugins:\n\n.*?(?=\nRest of the plugins)', 'Pre-installed plugins:\n\n' + ''.join(f'* `{name}:{version}`\n' for name, version in plugins)),
            (r'    basePlugins:\n.*?(?=```)', '    basePlugins:\n' + example),
        ],
        f'{LATEST}/schema.md': [
            (r'<p>BasePlugins contains plugins required by operator\nDefaults to :\n.*?</p>', '<p>BasePlugins contains plugins required by operator\nDefaults to :\n' + schema.rstrip() + '</p>'),
        ],
    }
    changes = {}
    for name, replacements in sections.items():
        path = root / name
        original = text = path.read_text()
        for pattern, replacement in replacements:
            text = replace_section(text, pattern, replacement, name)
        if text != original:
            changes[path] = text
    if not check:
        for path, text in changes.items():
            path.write_text(text)
    return list(changes)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    try:
        plugins = read_plugins((ROOT / 'pkg/plugins/base_plugins.go').read_text())
        changed = update_docs(ROOT, plugins, check=args.check)
    except (ValueError, OSError) as error:
        parser.exit(1, f'{error}\n')
    for path in changed:
        print(path.relative_to(ROOT))
    if args.check and changed:
        raise SystemExit(1)
