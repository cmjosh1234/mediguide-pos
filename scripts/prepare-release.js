#!/usr/bin/env node

/*
 * Synchronize the monorepo release version from one requested Git tag.
 * This intentionally avoids package-manager "version" commands because they
 * create component-level commits/tags that are incorrect for this monorepo.
 */

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const root = path.resolve(__dirname, "..");
const args = process.argv.slice(2);
let requested = args[0] || "";
const buildNumberIndex = args.indexOf("--build-number");
const explicitBuildNumber =
  buildNumberIndex === -1 ? null : args[buildNumberIndex + 1];

function fail(message) {
  process.stderr.write(message + "\n");
  process.exit(1);
}

if (["patch", "minor", "major"].includes(requested)) {
  const current = fs.readFileSync(path.join(root, "VERSION"), "utf8").trim();
  const currentMatch = current.match(/^(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z.-]+)?$/);
  if (!currentMatch) fail("Could not parse the root VERSION file.");
  let major = Number(currentMatch[1]);
  let minor = Number(currentMatch[2]);
  let patch = Number(currentMatch[3]);
  if (requested === "major") {
    major += 1;
    minor = 0;
    patch = 0;
  } else if (requested === "minor") {
    minor += 1;
    patch = 0;
  } else {
    patch += 1;
  }
  requested = "v" + major + "." + minor + "." + patch;
}

if (!/^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(requested)) {
  fail(
    "Usage: node scripts/prepare-release.js vMAJOR.MINOR.PATCH|patch|minor|major [--build-number NUMBER]",
  );
}

const expectedArgumentCount = buildNumberIndex === -1 ? 1 : 3;
if (args.length !== expectedArgumentCount || (buildNumberIndex !== -1 && buildNumberIndex !== 1)) {
  fail("Only --build-number NUMBER may follow the requested version or bump type.");
}

const version = requested.replace(/^v/, "");
if (
  explicitBuildNumber !== null &&
  !/^[1-9]\d*$/.test(explicitBuildNumber || "")
) {
  fail("--build-number must be a positive integer.");
}

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), "utf8");
}

function write(relativePath, contents) {
  const target = path.join(root, relativePath);
  const previous = fs.existsSync(target)
    ? fs.readFileSync(target, "utf8")
    : null;
  if (previous !== contents) {
    fs.writeFileSync(target, contents);
    process.stdout.write("updated " + relativePath + "\n");
  }
}

function replaceOnce(relativePath, pattern, replacement) {
  const contents = read(relativePath);
  const matches = contents.match(pattern);
  if (!matches) fail("Could not find the version field in " + relativePath + ".");
  write(relativePath, contents.replace(pattern, replacement));
}

function updatePackage(relativePath, updateLockRoot) {
  const manifest = JSON.parse(read(relativePath));
  manifest.version = version;
  if (updateLockRoot && manifest.packages && manifest.packages[""]) {
    manifest.packages[""].version = version;
  }
  write(relativePath, JSON.stringify(manifest, null, 2) + "\n");
}

const mobileContents = read("user_app/pubspec.yaml");
const mobileMatch = mobileContents.match(
  /^version:\s*(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)\+(\d+)$/m,
);
if (!mobileMatch) fail("Could not parse user_app/pubspec.yaml version.");

const currentMobileVersion = mobileMatch[1];
const currentBuildNumber = Number(mobileMatch[2]);
if (
  explicitBuildNumber !== null &&
  Number(explicitBuildNumber) < currentBuildNumber
) {
  fail(
    "Flutter build numbers cannot decrease (current: " +
      currentBuildNumber +
      ").",
  );
}
const nextBuildNumber =
  explicitBuildNumber !== null
    ? Number(explicitBuildNumber)
    : currentMobileVersion === version
      ? currentBuildNumber
      : currentBuildNumber + 1;

write("VERSION", version + "\n");
updatePackage("dashboard/package.json", false);
updatePackage("guidelines-platform/package.json", false);
updatePackage("guidelines-platform/package-lock.json", true);
replaceOnce(
  "ai-worker/pyproject.toml",
  /^version\s*=\s*"[^"]+"/m,
  'version = "' + version + '"',
);
replaceOnce(
  "ai-worker/uv.lock",
  /(\[\[package\]\]\nname = "mediguide-ai-worker"\nversion = ")[^"]+/,
  "$1" + version,
);
replaceOnce(
  "backend/cmd/api/main.go",
  /^\/\/ @version\s+\S+/m,
  "// @version " + version,
);
replaceOnce(
  "backend/docs/docs.go",
  /Version:\s+"[^"]+",/,
  'Version:          "' + version + '",',
);
replaceOnce(
  "backend/docs/swagger.json",
  /"version":\s*"[^"]+"/,
  '"version": "' + version + '"',
);
replaceOnce(
  "backend/docs/swagger.yaml",
  /^  version:\s*"?[^"\s]+"?/m,
  "  version: " + version,
);
write(
  "user_app/pubspec.yaml",
  mobileContents.replace(
    /^version:\s*.*$/m,
    "version: " + version + "+" + nextBuildNumber,
  ),
);

execFileSync("python3", [path.join(root, "scripts/generate-release-notes.py"), "v" + version], {
  stdio: "inherit",
});

process.stdout.write(
  "Release metadata synchronized: v" +
    version +
    " / mobile " +
    version +
    "+" +
    nextBuildNumber +
    "\n",
);
