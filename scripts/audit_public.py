import json
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[1]
failures = []
allowed_roots = {'.github', '.gitignore', '.gitattributes', 'AGENTS.md', 'README.md', 'LICENSE', 'NOTICE',
                 'CONTRIBUTING.md', 'SECURITY.md', 'CHANGELOG.md', 'COVERAGE.md', 'go.mod', 'go.sum',
                 'pkg', 'cmd', 'scripts', 'examples', 'third_party'}
for path in root.iterdir():
    if path.name in {'.git', 'bin', 'dist', 'work', '__pycache__'}:
        continue
    if path.name not in allowed_roots:
        failures.append('Unexpected top-level path: ' + path.name)
for folder in ['pkg', 'cmd', 'scripts', 'examples', 'third_party', '.github']:
    for path in (root / folder).rglob('*'):
        if path.is_dir():
            continue
        relative = path.relative_to(root).as_posix()
        if path.suffix.lower() in {'.pem', '.key', '.xsd', '.xml', '.xlsx', '.p12', '.pfx'}:
            failures.append('Forbidden publication file: ' + relative)
        if path.suffix in {'.go', '.json', '.py', '.yml', '.md'}:
            text = path.read_text(encoding='utf-8')
            if path.suffix == '.go' and re.search(r'"iris-agent-api(?:/|"|$)', text):
                failures.append('Private module reference: ' + relative)
            if re.search(r'-----BEGIN (?:RSA |EC )?PRIVATE KEY-----|sk_live_[A-Za-z0-9]{16,}|ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AKIA[A-Z0-9]{16}', text):
                failures.append('Credential pattern: ' + relative)


def check_input(value, location):
    if isinstance(value, dict):
        for key, item in value.items():
            if isinstance(item, str) and (key in {'TIN', 'SSN', 'FTIN'} or key.endswith('EIN')):
                if not re.sub(r'[- .]', '', item).startswith('000'):
                    failures.append('Nonfictional identifier: ' + location)
            if isinstance(item, str) and re.search(r'password|secret|token|credential|api.?key', key, re.I):
                failures.append('Credential field in example: ' + location)
            check_input(item, location)
    elif isinstance(value, list):
        for item in value:
            check_input(item, location)


for path in (root / 'pkg/firetoiris/testdata/inputs').rglob('*.json'):
    check_input(json.loads(path.read_text()), path.relative_to(root).as_posix())
if 'Apache License' not in (root / 'LICENSE').read_text():
    failures.append('Apache license is missing')
if failures:
    print('\n'.join(failures), file=sys.stderr)
    raise SystemExit(1)
print('Public-source audit passed: no backend imports, restricted schema packages, private keys or real identifiers.')
