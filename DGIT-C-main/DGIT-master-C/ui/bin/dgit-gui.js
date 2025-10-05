#!/usr/bin/env node

const { spawn } = require("child_process");
const path = require("path");

// 올바른 electron 경로 찾기
const electronPath = path.join(__dirname, "..", "node_modules", ".bin", "electron");

// 프로젝트 루트 디렉토리로 수정
const child = spawn(electronPath, [path.join(__dirname, "..")], { stdio: "inherit" });

child.on("close", (code) => {
  process.exit(code);
});