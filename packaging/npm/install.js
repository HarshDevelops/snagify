#!/usr/bin/env node

const fs = require("fs");
const os = require("os");
const path = require("path");
const https = require("https");
const crypto = require("crypto");
const childProcess = require("child_process");

const VERSION = "0.4.1";
const REPO = "HarshDevelops/snagify";
const ROOT = __dirname;
const VENDOR = path.join(ROOT, "vendor");

function assetName() {
  const platform = process.platform;
  const arch = process.arch;

  if (platform === "darwin" && arch === "arm64") return "snagify_Darwin_arm64.tar.gz";
  if (platform === "darwin" && arch === "x64") return "snagify_Darwin_x86_64.tar.gz";
  if (platform === "linux" && arch === "arm64") return "snagify_Linux_arm64.tar.gz";
  if (platform === "linux" && arch === "x64") return "snagify_Linux_x86_64.tar.gz";
  if (platform === "win32" && arch === "x64") return "snagify_Windows_x86_64.zip";

  throw new Error(`Unsupported platform/arch: ${platform}/${arch}`);
}

function binaryName() {
  return process.platform === "win32" ? "snagify.exe" : "snagify";
}

function run(cmd, args) {
  childProcess.execFileSync(cmd, args, { stdio: "inherit" });
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const req = https.get(url, { headers: { "User-Agent": "snagify-npm-installer" } }, (res) => {
      if ([301, 302, 303, 307, 308].includes(res.statusCode)) {
        file.close();
        fs.unlinkSync(dest);
        return download(res.headers.location, dest).then(resolve, reject);
      }

      if (res.statusCode !== 200) {
        file.close();
        fs.unlinkSync(dest);
        return reject(new Error(`Download failed: ${res.statusCode} ${url}`));
      }

      res.pipe(file);
      file.on("finish", () => file.close(resolve));
    });

    req.on("error", (err) => {
      file.close();
      if (fs.existsSync(dest)) fs.unlinkSync(dest);
      reject(err);
    });
  });
}

function sha256(file) {
  const h = crypto.createHash("sha256");
  h.update(fs.readFileSync(file));
  return h.digest("hex");
}

function parseChecksums(text) {
  const out = {};
  for (const line of text.split(/\r?\n/)) {
    const parts = line.trim().split(/\s+/);
    if (parts.length >= 2) out[parts[1]] = parts[0];
  }
  return out;
}

async function fetchRemote(asset, tmpDir) {
  const base = `https://github.com/${REPO}/releases/download/v${VERSION}`;
  const archive = path.join(tmpDir, asset);
  const checksumsFile = path.join(tmpDir, "checksums.txt");

  await download(`${base}/${asset}`, archive);
  await download(`${base}/checksums.txt`, checksumsFile);

  const sums = parseChecksums(fs.readFileSync(checksumsFile, "utf8"));
  if (!sums[asset]) throw new Error(`No checksum found for ${asset}`);

  const got = sha256(archive);
  if (got !== sums[asset]) {
    throw new Error(`Checksum mismatch for ${asset}. expected=${sums[asset]} got=${got}`);
  }

  return archive;
}

function useLocal(asset) {
  const localDist = process.env.SNAGIFY_LOCAL_DIST;
  if (!localDist) return null;

  const p = path.join(localDist, asset);
  if (!fs.existsSync(p)) throw new Error(`SNAGIFY_LOCAL_DIST set, but asset missing: ${p}`);

  return p;
}

function extract(archive, asset) {
  fs.mkdirSync(VENDOR, { recursive: true });

  if (asset.endsWith(".tar.gz")) {
    run("tar", ["-xzf", archive, "-C", VENDOR]);
  } else if (asset.endsWith(".zip")) {
    if (process.platform === "win32") {
      run("powershell.exe", ["-NoProfile", "-Command", `Expand-Archive -Path '${archive}' -DestinationPath '${VENDOR}' -Force`]);
    } else {
      run("unzip", ["-o", archive, "-d", VENDOR]);
    }
  } else {
    throw new Error(`Unknown archive type: ${asset}`);
  }

  const bin = path.join(VENDOR, binaryName());
  if (!fs.existsSync(bin)) throw new Error(`Binary missing after extraction: ${bin}`);

  if (process.platform !== "win32") fs.chmodSync(bin, 0o755);
}

async function main() {
  fs.mkdirSync(VENDOR, { recursive: true });

  const asset = assetName();
  const local = useLocal(asset);

  if (local) {
    extract(local, asset);
    return;
  }

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "snagify-npm-"));
  const archive = await fetchRemote(asset, tmp);
  extract(archive, asset);
}

main().catch((err) => {
  console.error(`snagify install failed: ${err.message}`);
  process.exit(1);
});
