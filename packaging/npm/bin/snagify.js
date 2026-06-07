#!/usr/bin/env node

const path = require("path");
const childProcess = require("child_process");

const exe = process.platform === "win32" ? "snagify.exe" : "snagify";
const bin = path.join(__dirname, "..", "vendor", exe);

const child = childProcess.spawn(bin, process.argv.slice(2), { stdio: "inherit" });

child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  else process.exit(code ?? 1);
});

child.on("error", (err) => {
  console.error(`failed to execute snagify: ${err.message}`);
  process.exit(1);
});
