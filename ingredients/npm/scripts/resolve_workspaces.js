// resolve_workspaces.js
//
// Replaces pnpm workspace:* dependency references in package.json with concrete
// version numbers resolved from sibling packages. Runs before npm publish so the
// published package has valid dependency specifiers.
//
// Usage: node resolve_workspaces.js
// Must be run from the package directory (e.g., packages/hooks/).

const fs = require("fs");
const path = require("path");

const pkgPath = path.join(process.cwd(), "package.json");
const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
let changed = false;

const depTypes = [
  "dependencies",
  "devDependencies",
  "peerDependencies",
  "optionalDependencies",
];

for (const depType of depTypes) {
  const deps = pkg[depType];
  if (!deps) continue;

  for (const [name, version] of Object.entries(deps)) {
    if (typeof version !== "string" || !version.startsWith("workspace:"))
      continue;

    // Find sibling package by scanning ../*/package.json
    const parentDir = path.dirname(process.cwd());
    const dirs = fs
      .readdirSync(parentDir, { withFileTypes: true })
      .filter((d) => d.isDirectory())
      .map((d) => d.name);

    let resolved = false;
    for (const dir of dirs) {
      const siblingPkg = path.join(parentDir, dir, "package.json");
      if (!fs.existsSync(siblingPkg)) continue;
      const sibling = JSON.parse(fs.readFileSync(siblingPkg, "utf8"));
      if (sibling.name === name) {
        const prefix =
          version === "workspace:*" ? "^" : version.replace("workspace:", "");
        deps[name] = prefix + sibling.version;
        console.log("Resolved " + name + ": " + version + " -> " + deps[name]);
        resolved = true;
        changed = true;
        break;
      }
    }

    if (!resolved) {
      console.error(
        "Warning: could not resolve " + name + " (" + version + ")"
      );
    }
  }
}

if (changed) {
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n");
  console.log("Updated package.json with resolved workspace dependencies");
} else {
  console.log("No workspace dependencies to resolve");
}
