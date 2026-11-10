import { lstat, open } from "node:fs/promises";
import { isAbsolute, normalize } from "node:path";
import { fileURLToPath } from "node:url";

const MAX_REPORT_BYTES = 4 * 1024 * 1024;
const OWNED_SOURCE = normalize(fileURLToPath(new URL("../src/index.ts", import.meta.url)));

async function main() {
  const reportPath = process.argv[2];
  if (process.argv.length !== 3 || typeof reportPath !== "string" || reportPath.length === 0) {
    fail("expected one coverage report path");
  }

  const text = await readBounded(reportPath);
  validateReport(parseReport(text));
}

async function readBounded(reportPath) {
  const link = await lstat(reportPath);
  if (!link.isFile() || link.isSymbolicLink()) {
    fail("coverage report is not a regular file");
  }

  const handle = await open(reportPath, "r");
  try {
    const stat = await requireStableReport(reportPath, link, handle);
    if (stat.size < 0 || stat.size > MAX_REPORT_BYTES) {
      fail("coverage report exceeds the byte limit");
    }
    const data = await readAtMost(handle);
    const after = await requireStableReport(reportPath, link, handle);
    if (after.size !== data.byteLength || after.size > MAX_REPORT_BYTES) {
      fail("coverage report changed while it was read");
    }
    return decodeUTF8(data);
  } finally {
    await handle.close();
  }
}

async function requireStableReport(reportPath, original, handle) {
  const [opened, current] = await Promise.all([handle.stat(), lstat(reportPath)]);
  if (!sameRegularFile(original, opened) || !sameRegularFile(original, current)) {
    fail("coverage report changed while it was read");
  }
  return opened;
}

function sameRegularFile(left, right) {
  return left.isFile() && right.isFile() && !left.isSymbolicLink() && !right.isSymbolicLink()
    && Number.isSafeInteger(left.dev) && Number.isSafeInteger(left.ino)
    && Number.isSafeInteger(right.dev) && Number.isSafeInteger(right.ino)
    && left.dev === right.dev && left.ino === right.ino;
}

async function readAtMost(handle) {
  const data = Buffer.allocUnsafe(MAX_REPORT_BYTES + 1);
  let offset = 0;
  while (offset < data.length) {
    const { bytesRead } = await handle.read(data, offset, data.length - offset, offset);
    if (bytesRead === 0) {
      break;
    }
    offset += bytesRead;
  }
  if (offset > MAX_REPORT_BYTES) {
    fail("coverage report exceeds the byte limit");
  }
  return data.subarray(0, offset);
}

function decodeUTF8(data) {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(data);
  } catch {
    fail("coverage report is not UTF-8");
  }
}

function parseReport(text) {
  try {
    return JSON.parse(text);
  } catch {
    fail("coverage report is not valid JSON");
  }
}

function validateReport(report) {
  const entries = ownEntries(report, "coverage report");
  if (entries.length !== 1) {
    fail("coverage report must contain exactly one owned source file");
  }

  const [reportedPath, coverage] = entries[0];
  requireOwnedSource(reportedPath);
  validateCoverage(coverage);
}

function validateCoverage(coverage) {
  if (!isRecord(coverage)) {
    fail("coverage entry is invalid");
  }
  requireOwnedSource(coverage.path);

  const statementMap = nonemptyEntries(coverage.statementMap, "statement source map");
  const functionMap = nonemptyEntries(coverage.fnMap, "function source map");
  const branchMap = nonemptyEntries(coverage.branchMap, "branch source map");
  const statements = countEntries(coverage.s, "statement");
  const functions = countEntries(coverage.f, "function");
  const branches = branchEntries(coverage.b);

  requireMatchingKeys(statementMap, statements, "statement");
  requireMatchingKeys(functionMap, functions, "function");
  requireMatchingKeys(branchMap, branches, "branch");
}

function requireOwnedSource(value) {
  if (normalizeCoveragePath(value) !== OWNED_SOURCE) {
    fail("coverage report contains an unexpected source file");
  }
}

function normalizeCoveragePath(value) {
  if (typeof value !== "string") {
    fail("coverage source identity is invalid");
  }
  try {
    if (value.startsWith("file:")) {
      return normalize(fileURLToPath(value));
    }
    if (!isAbsolute(value)) {
      fail("coverage source identity is invalid");
    }
    return normalize(value);
  } catch {
    fail("coverage source identity is invalid");
  }
}

function nonemptyEntries(value, name) {
  const entries = ownEntries(value, name);
  if (entries.length === 0) {
    fail(`${name} must not be empty`);
  }
  return entries;
}

function countEntries(value, name) {
  const entries = nonemptyEntries(value, `${name} map`);
  for (const [, count] of entries) {
    if (!isCount(count)) {
      fail(`${name} map contains an invalid count`);
    }
  }
  return entries;
}

function branchEntries(value) {
  const entries = nonemptyEntries(value, "branch map");
  for (const [, counts] of entries) {
    if (!Array.isArray(counts) || counts.length === 0 || !counts.every(isCount)) {
      fail("branch map contains invalid counts");
    }
  }
  return entries;
}

function requireMatchingKeys(left, right, name) {
  const rightKeys = new Set(right.map(([key]) => key));
  if (left.length !== right.length || left.some(([key]) => !rightKeys.has(key))) {
    fail(`${name} source and count maps do not match`);
  }
}

function ownEntries(value, name) {
  if (!isRecord(value)) {
    fail(`${name} is invalid`);
  }
  return Object.entries(value);
}

function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isCount(value) {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
}

function fail(message) {
  throw new Error(`coverage validation failed: ${message}`);
}

await main();
