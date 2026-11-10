#!/usr/bin/env python3
"""Build verified static Linux releases locally; no server-side compilation."""
import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
ARCHITECTURES = ('amd64', 'arm64')


def run(*arguments, cwd=ROOT, env=None, capture=False):
    return subprocess.run(arguments, cwd=cwd, env=env, check=True,
                          stdout=subprocess.PIPE if capture else None, text=capture)


def source_digest():
    result = hashlib.sha256()
    for filename in run('git', 'ls-files', '-z', capture=True).stdout.split('\0'):
        if not filename:
            continue
        path = ROOT / filename
        if path.is_symlink() or not path.is_file():
            raise ValueError('Release sources must be regular committed files.')
        result.update(filename.encode() + b'\0' + path.read_bytes() + b'\0')
    return result.hexdigest()


def build(version, go_name, arches):
    if not re.fullmatch(r'v\d+\.\d+\.\d+', version):
        raise ValueError('Use a semantic release version such as v0.1.0.')
    go = Path(shutil.which(go_name) or go_name).resolve(strict=True)
    if run('git', 'status', '--porcelain', capture=True).stdout:
        raise ValueError('Commit the reviewed source before building a release.')
    revision = run('git', 'rev-parse', 'HEAD', capture=True).stdout.strip()
    before = source_digest()
    if run(str(go.with_name('gofmt')), '-l', 'cmd', 'internal', capture=True).stdout:
        raise ValueError('Run gofmt before building a release.')
    run(str(go), 'run', './cmd/apigen')
    run('bun', 'run', 'generate', cwd=ROOT / 'web')
    run('git', 'diff', '--exit-code', '--', 'api', 'web/src/shared/api')
    run(str(go), 'vet', './...')
    run(str(go), 'test', '-race', './...', env=dict(os.environ, CGO_ENABLED='1'))
    run('bun', 'run', 'check', cwd=ROOT / 'web')
    run('bun', 'run', 'build', cwd=ROOT / 'web')
    output = ROOT / 'dist'
    output.mkdir(exist_ok=True)
    hashes = []
    with tempfile.TemporaryDirectory(prefix='mini-games-release-') as temporary:
        binary = Path(temporary) / 'mini-games'
        for arch in arches:
            run(str(go), 'build', '-trimpath', '-buildvcs=false', '-ldflags', '-s -w -X main.version=' + version,
                '-o', str(binary), './cmd/arcade', env=dict(os.environ, GOOS='linux', GOARCH=arch, CGO_ENABLED='0'))
            files = {'mini-games': binary.read_bytes(), 'LICENSE': (ROOT / 'LICENSE').read_bytes(),
                     'THIRD-PARTY-NOTICES.txt': (ROOT / 'THIRD-PARTY-NOTICES.txt').read_bytes()}
            for path in sorted((ROOT / 'web/dist').rglob('*')):
                if path.is_symlink():
                    raise ValueError('Unexpected symbolic static build entry.')
                if path.is_dir():
                    continue
                if not path.is_file():
                    raise ValueError('Unexpected static build entry.')
                data = path.read_bytes()
                if path.suffix == '.html':
                    data = data.replace(b'<head>', f'<head><meta name="application-version" content="{version}">'.encode(), 1)
                files['site/' + path.relative_to(ROOT / 'web/dist').as_posix()] = data
            # License notices accompany JS delivered to browsers as well as the binary archive.
            files['site/licenses.txt'] = files['LICENSE'] + b'\n\n' + files['THIRD-PARTY-NOTICES.txt']
            manifest = {'schema': 1, 'version': version, 'revision': revision, 'source': before, 'arch': arch,
                        'files': {name: hashlib.sha256(data).hexdigest() for name, data in files.items()}}
            files['manifest.json'] = (json.dumps(manifest, sort_keys=True) + '\n').encode()
            archive = output / f'mini-games-linux-{arch}.tar.gz'
            with archive.open('wb') as raw, gzip.GzipFile(filename='', fileobj=raw, mode='wb', mtime=0) as compressed:
                with tarfile.open(fileobj=compressed, mode='w') as bundle:
                    for name, data in sorted(files.items()):
                        entry = tarfile.TarInfo(name)
                        entry.mode = 0o755 if name == 'mini-games' else 0o644
                        entry.size = len(data)
                        bundle.addfile(entry, io.BytesIO(data))
            digest = hashlib.sha256(archive.read_bytes()).hexdigest()
            hashes.append(f'{digest}  {archive.name}')
            print(f'{archive.name}: {archive.stat().st_size / 2**20:.2f} MiB, SHA256 {digest}')
    if source_digest() != before:
        raise ValueError('Sources changed during build; discard the release and rebuild.')
    (output / 'SHA256SUMS').write_text('\n'.join(hashes) + '\n')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--version', required=True)
    parser.add_argument('--go', default='go')
    parser.add_argument('--arch', action='append', choices=ARCHITECTURES)
    arguments = parser.parse_args()
    try:
        build(arguments.version, arguments.go, arguments.arch or ARCHITECTURES)
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        parser.exit(1, str(error) + '\n')
