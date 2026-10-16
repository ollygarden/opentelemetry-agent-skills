#!/usr/bin/env python3
"""Validate emission provenance against cached upstream Git trees (no dependencies)."""
import argparse
import datetime
import json
import pathlib
import re
import subprocess
import sys


def git(repo, *args):
    result = subprocess.run(['git', '-C', str(repo), *args], capture_output=True, text=True)
    if result.returncode:
        raise ValueError(result.stderr.strip())
    return result.stdout.strip()


def frontmatter(text):
    # Scanner metadata is a flat mapping of scalar strings (or a null date).
    if not text.startswith('---\n') or '\n---\n' not in text[4:]:
        raise ValueError('missing frontmatter')
    fields = {}
    for line in text.split('---', 2)[1].strip().splitlines():
        key, separator, value = line.partition(':')
        if not separator or key in fields:
            raise ValueError('invalid or duplicate metadata field')
        fields[key] = value.strip().strip('\"\'')
    return fields


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output-root', type=pathlib.Path,
                        default=pathlib.Path(__file__).resolve().parents[3] / 'skills/otel-telemetry-emissions')
    parser.add_argument('--repo-cache', type=pathlib.Path, default=pathlib.Path('/tmp/otel-component-telemetry'))
    parser.add_argument('--component', help='Exact <owner>/<repo>/<component path>')
    args = parser.parse_args()
    errors, seen, trees = [], {}, {}
    checked = 0
    for file in sorted(args.output_root.rglob('v*.md')):
        try:
            text = file.read_text()
            if args.component:
                repo_match = re.search(r'^repo:\s*([^\n]+)', text, re.M)
                path_match = re.search(r'^path:\s*([^\n]+)', text, re.M)
                advertised = (repo_match[1].strip() + '/' + path_match[1].strip()) if repo_match and path_match else None
                canonical_parent = args.output_root / args.component.split('/', 1)[1]
                if advertised != args.component and file.parent != canonical_parent:
                    continue
            fields = frontmatter(text)
            required = {'repo', 'path', 'version', 'scope_name', 'commit_sha', 'last_verified'}
            if required - fields.keys():
                raise ValueError('missing fields: ' + ', '.join(sorted(required - fields.keys())))
            component = fields['repo'] + '/' + fields['path']
            if args.component and component != args.component:
                raise ValueError('component metadata differs from its directory')
            checked += 1
            relative = file.relative_to(args.output_root).as_posix()
            expected = fields['repo'].split('/')[-1] + '/' + fields['path'] + '/' + fields['version'] + '.md'
            if relative != expected:
                raise ValueError('noncanonical path; expected ' + expected)
            key = (component, fields['version'])
            if key in seen:
                raise ValueError('duplicate component/version: ' + seen[key])
            seen[key] = relative
            sha = fields['commit_sha']
            if not re.fullmatch('[0-9a-f]{40}', sha):
                raise ValueError('commit_sha must be a full commit SHA')
            repo = args.repo_cache / fields['repo'].split('/')[-1]
            if git(repo, 'rev-parse', '--verify', sha + '^{commit}') != sha:
                raise ValueError('commit_sha identifies a tag object, not a commit')
            if not git(repo, 'tag', '--points-at', sha):
                raise ValueError('source commit is not a cached release tag')
            basename = fields['repo'].split('/')[-1]
            tag = None
            if basename in ('opentelemetry-collector', 'opentelemetry-collector-contrib', 'opentelemetry-java-instrumentation'):
                tag = fields['version']
            elif basename == 'opentelemetry-go-contrib':
                tag = fields['path'] + '/' + fields['version']
            elif basename == 'opentelemetry-dotnet-contrib':
                tag = fields['path'].split('/')[-1].removeprefix('OpenTelemetry.') + '-' + fields['version'].removeprefix('v')
            elif basename == 'opentelemetry-js-contrib':
                tag = fields['path'].split('/')[-1] + '-' + fields['version']
            if tag and git(repo, 'rev-parse', '--verify', 'refs/tags/' + tag + '^{commit}') != sha:
                raise ValueError('commit_sha differs from the component release tag: ' + tag)
            package_version = None
            if basename in ('opentelemetry-js', 'opentelemetry-js-contrib'):
                package_version = json.loads(git(repo, 'show', sha + ':' + fields['path'] + '/package.json'))['version']
            elif basename == 'opentelemetry-python-contrib':
                module = fields['path'].split('/')[-1].removeprefix('opentelemetry-instrumentation-').replace('-', '_')
                source = git(repo, 'show', f"{sha}:{fields['path']}/src/opentelemetry/instrumentation/{module}/version.py")
                match = re.search(r'__version__\s*=\s*[\"\']([^\"\']+)', source)
                if not match:
                    raise ValueError('package version constant not found')
                package_version = match[1]
            if package_version and fields['version'] != 'v' + package_version:
                raise ValueError('version differs from package manifest: v' + package_version)
            date = fields['last_verified']
            if date != 'null':
                verified = datetime.date.fromisoformat(date)
                commit_date = datetime.date.fromisoformat(git(repo, 'show', '-s', '--format=%cs', sha))
                if verified > datetime.date.today() or verified < commit_date:
                    raise ValueError('last_verified is in the future or predates the source commit; recheck or use null')
            tree_key = (repo, sha)
            if tree_key not in trees:
                trees[tree_key] = set(git(repo, 'ls-tree', '-r', '--name-only', sha).splitlines())
            paths = trees[tree_key]
            if not any(p.startswith(fields['path'] + '/') for p in paths):
                raise ValueError('component path absent from source commit')
            sources = text.split('## Sources', 1)
            if len(sources) != 2 or not any(line.startswith('- ') for line in sources[1].splitlines()):
                raise ValueError('missing Sources entries')
            for line in sources[1].splitlines():
                if not line.startswith('- '):
                    continue
                source = line[2:].strip()
                if source.startswith('`'):
                    source = source.split('`')[1]
                source = source.split()[0] if source else ''
                # Historical prose/glob references need manual review; literal
                # file and directory references can be checked without fetching blobs.
                if re.fullmatch(r'[A-Za-z0-9_./@+\-]+', source) and ('/' in source or re.search(r'\.[A-Za-z0-9]+$', source)) and '...' not in source:
                    if source not in paths and not any(p.startswith(source.rstrip('/') + '/') for p in paths):
                        errors.append(f'{relative}: source path absent: {source}')
        except (ValueError, OSError) as error:
            errors.append(f'{file}: {error}')
    if checked == 0:
        errors.append('no emission files matched')
    for error in errors:
        print(error, file=sys.stderr)
    print(f'Checked {checked} emission files; {len(errors)} errors')
    return bool(errors)


if __name__ == '__main__':
    sys.exit(main())
