import hashlib
import os
import platform
import shutil
import stat
import subprocess
import tarfile
import tempfile
import urllib.request
import zipfile
from pathlib import Path

VERSION = "0.4.1"
REPO = "HarshDevelops/snagify"

def asset_name():
    system = platform.system().lower()
    machine = platform.machine().lower()

    if machine in ("x86_64", "amd64"):
        arch = "x86_64"
    elif machine in ("aarch64", "arm64"):
        arch = "arm64"
    else:
        raise RuntimeError(f"Unsupported architecture: {machine}")

    if system == "darwin":
        return f"snagify_Darwin_{arch}.tar.gz"
    if system == "linux":
        return f"snagify_Linux_{arch}.tar.gz"
    if system == "windows" and arch == "x86_64":
        return "snagify_Windows_x86_64.zip"

    raise RuntimeError(f"Unsupported platform: {system}/{machine}")

def binary_name():
    return "snagify.exe" if platform.system().lower() == "windows" else "snagify"

def cache_dir():
    base = os.environ.get("SNAGIFY_PY_CACHE")
    if base:
        return Path(base)

    if platform.system().lower() == "windows":
        root = os.environ.get("LOCALAPPDATA", str(Path.home() / "AppData" / "Local"))
        return Path(root) / "snagify"

    return Path.home() / ".cache" / "snagify"

def download(url, dest):
    req = urllib.request.Request(url, headers={"User-Agent": "snagify-pypi-installer"})
    with urllib.request.urlopen(req, timeout=30) as r, open(dest, "wb") as f:
        shutil.copyfileobj(r, f)

def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()

def parse_checksums(text):
    out = {}
    for line in text.splitlines():
        parts = line.strip().split()
        if len(parts) >= 2:
            out[parts[1]] = parts[0]
    return out

def local_asset(asset):
    local = os.environ.get("SNAGIFY_LOCAL_DIST")
    if not local:
        return None

    path = Path(local) / asset
    if not path.exists():
        raise RuntimeError(f"SNAGIFY_LOCAL_DIST set but asset missing: {path}")

    return path

def fetch_asset(asset, tmp):
    local = local_asset(asset)
    if local:
        return local

    base = f"https://github.com/{REPO}/releases/download/v{VERSION}"
    archive = Path(tmp) / asset
    checksums = Path(tmp) / "checksums.txt"

    download(f"{base}/{asset}", archive)
    download(f"{base}/checksums.txt", checksums)

    sums = parse_checksums(checksums.read_text())
    expected = sums.get(asset)
    if not expected:
        raise RuntimeError(f"No checksum found for {asset}")

    got = sha256(archive)
    if got != expected:
        raise RuntimeError(f"Checksum mismatch for {asset}: expected={expected} got={got}")

    return archive

def extract(archive, dest):
    dest.mkdir(parents=True, exist_ok=True)

    if str(archive).endswith(".tar.gz"):
        with tarfile.open(archive, "r:gz") as t:
            t.extractall(dest)
    elif str(archive).endswith(".zip"):
        with zipfile.ZipFile(archive) as z:
            z.extractall(dest)
    else:
        raise RuntimeError(f"Unknown archive type: {archive}")

    binary = dest / binary_name()
    if not binary.exists():
        raise RuntimeError(f"Binary missing after extraction: {binary}")

    if platform.system().lower() != "windows":
        binary.chmod(binary.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

    return binary

def ensure_binary():
    dest = cache_dir() / VERSION
    binary = dest / binary_name()
    if binary.exists():
        return binary

    asset = asset_name()
    with tempfile.TemporaryDirectory(prefix="snagify-pypi-") as tmp:
        archive = fetch_asset(asset, tmp)
        return extract(archive, dest)

def run(argv):
    binary = ensure_binary()
    p = subprocess.run([str(binary)] + argv)
    return p.returncode
