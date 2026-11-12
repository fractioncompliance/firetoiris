import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import zipfile

TARGETS = {
    'macos-arm64': ('darwin', 'arm64'),
    'macos-amd64': ('darwin', 'amd64'),
    'windows-amd64': ('windows', 'amd64'),
    'windows-arm64': ('windows', 'arm64'),
    'linux-amd64': ('linux', 'amd64'),
    'linux-arm64': ('linux', 'arm64'),
}


def package(root, output, go, target, smoke):
    goos, goarch = TARGETS[target]
    executable = 'FIREtoIRIS.exe' if goos == 'windows' else 'FIREtoIRIS'
    commit = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip()
    dirty = bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=root, text=True).strip())
    with tempfile.TemporaryDirectory(prefix='firetoiris-package-') as temp:
        folder = Path(temp) / 'FIREtoIRIS'
        folder.mkdir()
        env = dict(os.environ, GOOS=goos, GOARCH=goarch, CGO_ENABLED='0')
        subprocess.run([go, 'build', '-trimpath', '-o', str(folder / executable), './cmd/firetoiris'],
                       cwd=root, env=env, check=True)
        for name in ['README.md', 'LICENSE', 'NOTICE', 'SECURITY.md']:
            shutil.copy2(root / name, folder / name)
        shutil.copy2(root / 'examples/nec.fire.txt', folder / 'nec.fire.txt')
        shutil.copytree(root / 'third_party', folder / 'third_party')
        if smoke:
            actual = subprocess.check_output([go, 'env', 'GOOS', 'GOARCH'], cwd=root, text=True).split()
            if actual != [goos, goarch]:
                raise RuntimeError('Native smoke test requires the target OS and architecture')
            result = subprocess.check_output([str(folder / executable), 'inspect', '--input', str(folder / 'nec.fire.txt'), '--tax-year', '2026'], text=True)
            report = json.loads(result)
            if report['records'] != 1 or report['amount_field_sums']['1099NEC.NonemployeeCompensationAmt'] != '1234.56' or '000654321' in result:
                raise RuntimeError('Native executable smoke test failed')
        manifest = {
            'name': 'FIREtoIRIS', 'platform': target, 'goos': goos, 'goarch': goarch,
            'source_commit': commit, 'source_dirty': dirty, 'native_smoke_tested': smoke,
            'files': {p.relative_to(folder).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(folder.rglob('*')) if p.is_file()},
        }
        (folder / 'MANIFEST.json').write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
        archive = output / ('FIREtoIRIS-' + target + '.zip')
        staged = output / (archive.name + '.pending')
        try:
            with zipfile.ZipFile(staged, 'w', zipfile.ZIP_DEFLATED) as z:
                for path in sorted(folder.rglob('*')):
                    if path.is_file():
                        z.write(path, 'FIREtoIRIS/' + path.relative_to(folder).as_posix())
            with zipfile.ZipFile(staged) as z:
                if z.testzip() is not None:
                    raise RuntimeError('Archive validation failed')
                for name, expected in manifest['files'].items():
                    if hashlib.sha256(z.read('FIREtoIRIS/' + name)).hexdigest() != expected:
                        raise RuntimeError('Archive digest mismatch')
            staged.replace(archive)
        finally:
            staged.unlink(missing_ok=True)
    return archive


def main():
    parser = argparse.ArgumentParser(description='Build and verify portable FIREtoIRIS ZIP downloads.')
    parser.add_argument('--target', action='append', choices=TARGETS, help='Repeat to select targets; default: all six.')
    parser.add_argument('--output-dir', type=Path, default=Path('dist/firetoiris'))
    parser.add_argument('--smoke-native', action='store_true', help='Smoke-test only builds matching this computer while packaging all targets.')
    parser.add_argument('--smoke', action='store_true', help='Run the native target executable against the fictional NEC export.')
    parser.add_argument('--go', default=shutil.which('go'))
    args = parser.parse_args()
    if not args.go:
        parser.error('Go 1.26+ is required; supply --go or add Go to PATH.')
    root = Path(__file__).resolve().parents[1]
    output = args.output_dir.resolve()
    output.mkdir(parents=True, exist_ok=True)
    host = tuple(subprocess.check_output([args.go, 'env', 'GOOS', 'GOARCH'], cwd=root, text=True).split())
    files = [package(root, output, args.go, target, args.smoke or (args.smoke_native and TARGETS[target] == host)) for target in (args.target or TARGETS)]
    hashes = '\n'.join(hashlib.sha256(p.read_bytes()).hexdigest() + '  ' + p.name for p in sorted(output.glob('FIREtoIRIS-*.zip'))) + '\n'
    (output / 'SHA256SUMS.txt').write_text(hashes, encoding='utf-8')
    print(json.dumps({'packages': [str(p) for p in files]}, indent=2))


if __name__ == '__main__':
    main()
