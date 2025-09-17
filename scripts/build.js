const fs = require("fs");
const path = require("path");

// Define the assets to be copied from node_modules to the static directory.
// This makes it easy to add or remove frontend dependencies.
const assets = [
  {
    source: "node_modules/htmx.org/dist/htmx.min.js",
    destination: "web/static/js/htmx.min.js",
  },
  {
    source: "node_modules/alpinejs/dist/cdn.min.js",
    destination: "web/static/js/alpine.min.js",
  },
  {
    source: "node_modules/htmx.org/dist/ext/ws.js",
    destination: "web/static/js/ws.js",
  },
];

console.log("Copying frontend assets...");

assets.forEach((asset) => {
  const destDir = path.dirname(asset.destination);

  // Ensure the destination directory exists.
  fs.mkdirSync(destDir, { recursive: true });

  // Copy the file.
  fs.copyFileSync(asset.source, asset.destination);
  console.log(`Copied ${path.basename(asset.source)} to ${asset.destination}`);
});

console.log("Frontend assets copied successfully.");
