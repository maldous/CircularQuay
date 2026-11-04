/** Typed CircularQuay v1 clients. The SDK never discovers credentials or starts a daemon. */
import { spawn, type ChildProcess } from "node:child_process";
import { createHash, timingSafeEqual } from "node:crypto";
import { lstat, open, realpath } from "node:fs/promises";
import { isAbsolute } from "node:path";

const MAX_RESPONSE_BYTES = 1024 * 1024;
const MAX_STDIO_BYTES = 1024 * 1024;
const MAX_EXECUTABLE_BYTES = 256 * 1024 * 1024;
const MAX_SECRET_VALUE_BYTES = 64 * 1024;
const MAX_PROVIDER_VERSION_BYTES = 4096;
const MAX_RUNTIME_VERSION_BYTES = 256;
const MAX_TOOLS = 4;
const MAX_WORKLOADS = 4096;
const MAX_JSON_DEPTH = 32;
const MAX_JSON_NODES = 16_384;
const HANDLE = /^[A-Za-z0-9_.-]{1,128}$/;
const HEX_64 = /^[a-f0-9]{64}$/;
const RFC3339_DATETIME = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/;
const DENIED_ENV_PREFIXES = ["AWS_", "AZURE_", "GOOGLE_", "VAULT_", "OP_", "BW_", "CQUAY_CONFIG_"];
const KNOWN_TOOLS = new Set(["list_workloads", "get_status", "run_workload", "read_secret"]);
const STATES = new Set(["unprovisioned", "ready", "pending", "error", "running", "stopped", "unknown", "not_provisioned", "blocked", "resolved", "snapshotted", "applied", "activated", "apply_failed", "pending_activation", "partial_resolved", "partial_applied", "partial_activated", "partial_apply_failed", "partial_pending_activation"]);

export type ErrorCategory = "invalid" | "denied" | "trust" | "unavailable" | "expired" | "unsupported" | "protocol" | "cancelled" | "integrity" | "process";

const NATIVE_ERROR_CATEGORIES: Readonly<Record<string, ErrorCategory>> = {
  invalid_policy: "invalid",
  access_denied: "denied",
  trust_failure: "trust",
  expired_value: "expired",
  unsupported_capability: "unsupported",
  unavailable: "unavailable",
  missing_value: "unavailable",
  activation_failed: "unavailable"
};

/** A stable, deliberately detail-free SDK error. */
export class CquayError extends Error {
  readonly category: ErrorCategory;

  constructor(category: ErrorCategory) {
    super(`CircularQuay SDK ${category}`);
    this.name = "CquayError";
    this.category = category;
  }
}

export interface CallOptions { readonly signal?: AbortSignal; }
export interface ToolCall { readonly name: string; readonly arguments: Readonly<Record<string, unknown>>; }
/** Caller-owned adapter. Implement it with a connected official MCP client session. */
export interface MCPTransport {
  listTools(options?: CallOptions): Promise<unknown>;
  callTool(call: ToolCall, options?: CallOptions): Promise<unknown>;
}
export interface OfficialMCPClientSession {
  listTools(params?: { readonly cursor?: string }, options?: CallOptions): Promise<unknown>;
  callTool(call: ToolCall, resultSchema: unknown, options?: CallOptions): Promise<unknown>;
}

/**
 * Adapts the v1.32.2 official SDK without owning its connection or transport.
 * `callToolResultSchema` is the caller's `CallToolResultSchema` import.
 */
export function officialMCPAdapter(session: OfficialMCPClientSession, callToolResultSchema: unknown): MCPTransport {
  return {
    async listTools({ signal } = {}) {
      const result = await session.listTools(undefined, signal === undefined ? undefined : { signal });
      const nextCursor = object(result).nextCursor;
      if (nextCursor !== undefined && nextCursor !== "") throw new CquayError("protocol");
      return { pages: [result] };
    },
    callTool(call, { signal } = {}) {
      return session.callTool(call, callToolResultSchema, signal === undefined ? undefined : { signal });
    }
  };
}

export interface Observation {
  readonly action: "discover_tools" | "list_workloads" | "get_status" | "run_workload" | "read_secret" | "local_cli";
  readonly outcome: "success" | ErrorCategory;
  readonly durationMs: number;
}
export type Observer = (observation: Observation) => void | Promise<void>;
export interface CquayClientOptions { readonly observer?: Observer; }
export interface ToolDescriptor { readonly name: string; readonly description?: string; }
export interface WorkloadStatus { readonly workload: string; readonly mode: "offline" | "online" | "hybrid"; readonly state: string; }
export interface RunResult { readonly id: string; readonly status: "completed" | "failed" | "running" | "cancelled" | "unknown"; readonly exitCode: number; }

/** A redacted wrapper around a deliberate `read_secret` disclosure. */
export class SecretDisclosure {
  readonly alias: string;
  readonly version?: string;
  readonly expiresAt?: string;
  #value: string;

  constructor(alias: string, value: string, version?: string, expiresAt?: string) {
    this.alias = alias;
    this.#value = value;
    if (version !== undefined) this.version = version;
    if (expiresAt !== undefined) this.expiresAt = expiresAt;
  }

  /** Returns plaintext only after the caller deliberately chooses to use it. */
  reveal(): string { return this.#value; }
  toString(): string { return "[CircularQuay secret redacted]"; }
  toJSON(): Record<string, string | undefined> {
    return { alias: this.alias, version: this.version, expiresAt: this.expiresAt, value: "[redacted]" };
  }
  [Symbol.for("nodejs.util.inspect.custom")](): string { return this.toString(); }
}

/** Separate capability so plaintext disclosure cannot happen through the standard client. */
export interface DisclosureClient { readSecret(workload: string, key: string, options?: CallOptions): Promise<SecretDisclosure>; }

export class CquayClient {
  readonly #transport: MCPTransport;
  readonly #observer: Observer | undefined;

  constructor(transport: MCPTransport, options: CquayClientOptions = {}) {
    this.#transport = transport;
    this.#observer = options.observer;
  }

  disclosure(): DisclosureClient {
    return { readSecret: (workload, key, options) => this.readSecret(workload, key, options) };
  }

  async discoverTools(options: CallOptions = {}): Promise<readonly ToolDescriptor[]> {
    return this.observe("discover_tools", options, async () => {
      const pages = object(await this.#transport.listTools(options)).pages;
      if (!Array.isArray(pages) || pages.length !== 1) throw new CquayError("protocol");
      return parseTools(pages[0]);
    });
  }

  listWorkloads(options: CallOptions = {}): Promise<readonly string[]> {
    return this.tool("list_workloads", {}, options, parseWorkloads);
  }

  getStatus(workload: string, options: CallOptions = {}): Promise<WorkloadStatus> {
    validateHandle(workload);
    return this.tool("get_status", { workload }, options, value => parseStatus(value, workload));
  }

  runWorkload(workload: string, options: CallOptions = {}): Promise<RunResult> {
    validateHandle(workload);
    return this.tool("run_workload", { workload }, options, parseRunResult);
  }

  private readSecret(workload: string, key: string, options: CallOptions = {}): Promise<SecretDisclosure> {
    validateHandle(workload);
    validateHandle(key);
    return this.tool("read_secret", { workload, key }, options, value => parseSecret(value, key));
  }

  private async tool<T>(name: Observation["action"], args: Readonly<Record<string, unknown>>, options: CallOptions, parse: (value: unknown) => T): Promise<T> {
    return this.observe(name, options, async () => {
      const response = object(await this.#transport.callTool({ name, arguments: args }, options));
      if (response.isError === true) throw new CquayError("denied");
      return parse(structured(response));
    });
  }

  private observe<T>(action: Observation["action"], options: CallOptions, operation: () => Promise<T>): Promise<T> {
    return observe(this.#observer, action, options.signal, operation);
  }
}

export interface LocalCLIOptions {
  readonly executablePath: string;
  readonly expectedSHA256: string;
  readonly configPath: string;
  readonly stateDirectory: string;
  readonly environment?: Readonly<Record<string, string>>;
  readonly observer?: Observer;
}
export interface LocalRunOptions extends CallOptions { readonly stdin?: string; readonly inheritOutput?: boolean; }
export interface LocalResult { readonly exitCode: number; readonly stdout: string; }
interface RemotePin { readonly configSource?: string; readonly configRef?: string; readonly configPath?: string; readonly configSHA256?: string; }
type CompleteRemotePin = Required<RemotePin>;

/** Explicitly configured local CLI client; it never reads ambient environment or profile state. */
export class LocalCLIClient {
  readonly #executable: string;
  readonly #digest: string;
  readonly #config: string;
  readonly #state: string;
  readonly #environment: Readonly<Record<string, string>>;
  readonly #observer: Observer | undefined;

  private constructor(executable: string, digest: string, config: string, state: string, environment: Readonly<Record<string, string>>, observer: Observer | undefined) {
    this.#executable = executable;
    this.#digest = digest;
    this.#config = config;
    this.#state = state;
    this.#environment = environment;
    this.#observer = observer;
  }

  static async create(options: LocalCLIOptions): Promise<LocalCLIClient> {
    validateLocalOptions(options);
    const executable = await verifyExecutable(options.executablePath, options.expectedSHA256);
    const client = new LocalCLIClient(executable, options.expectedSHA256, options.configPath, options.stateDirectory, cleanEnvironment(options.environment), options.observer);
    await client.version();
    return client;
  }

  async version(options: CallOptions = {}): Promise<Readonly<{ version: string; commit: string; go: string; maturity: string }>> {
    const value = await this.json(["version"], options);
    if (!isVersion(value)) throw new CquayError("protocol");
    if (semanticMajor(value.version) !== 1) throw new CquayError("unsupported");
    return Object.freeze(value);
  }

  providers(options: CallOptions = {}): Promise<unknown> { return this.json(["providers"], options); }

  bootstrap(args: RemotePin & { readonly output: string }, options: CallOptions = {}): Promise<unknown> {
    if (!isAbsolute(args.output)) throw new CquayError("invalid");
    validateRemotePin(args);
    return this.json(["bootstrap", ...flags(args, { ...remoteFlags, output: "--output" }), "--non-interactive"], options);
  }

  configInit(output?: string, options: CallOptions = {}): Promise<unknown> {
    if (output !== undefined && !isAbsolute(output)) throw new CquayError("invalid");
    return this.json(["config", "init", ...optionalFlag("--output", output)], options);
  }
  configValidate(options: CallOptions = {}): Promise<unknown> { return this.json(["config", "validate"], options); }
  configGet(pointer: string, options: CallOptions = {}): Promise<unknown> { validatePointer(pointer); return this.json(["config", "get", "--pointer", pointer], options); }
  configList(pointer = "", options: CallOptions = {}): Promise<unknown> { validatePointer(pointer); return this.json(["config", "list", "--pointer", pointer], options); }

  configSet(pointer: string, value: unknown, expectSHA256?: string, options: CallOptions = {}): Promise<unknown> {
    validatePointer(pointer);
    return this.json(["config", "set", "--pointer", pointer, "--value-file", "-", ...expected(expectSHA256)], { ...options, stdin: strictJSON(value) });
  }
  configDelete(pointer: string, expectSHA256?: string, options: CallOptions = {}): Promise<unknown> {
    validatePointer(pointer);
    return this.json(["config", "delete", "--pointer", pointer, ...expected(expectSHA256)], options);
  }
  configReplace(document: unknown, expectSHA256?: string, options: CallOptions = {}): Promise<unknown> {
    return this.json(["config", "replace", "--file", "-", ...expected(expectSHA256)], { ...options, stdin: strictJSON(document) });
  }
  doctor(options: { readonly fix?: boolean; readonly identityOnly?: boolean; readonly signal?: AbortSignal } = {}): Promise<unknown> {
    if (options.fix && options.identityOnly) throw new CquayError("invalid");
    const mode = doctorMode(options);
    return this.json(["doctor", ...mode], options);
  }
  plan(options: CallOptions = {}): Promise<unknown> { return this.json(["plan"], options); }
  /** Checks available updates only when the caller explicitly requests it. */
  updateCheck(options: CallOptions = {}): Promise<unknown> { return this.json(["update", "check", "--json"], options); }
  status(workload: string, options: CallOptions = {}): Promise<unknown> { validateHandle(workload); return this.json(["status", "--workload", workload, "--json"], options); }
  snapshot(workload: string, target?: string, options: CallOptions = {}): Promise<unknown> { validateHandle(workload); return this.json(["snapshot", "--workload", workload, ...targetFlag(target)], options); }
  apply(workload: string, target: string, options: CallOptions = {}): Promise<unknown> { validateHandle(workload); validateHandle(target); return this.json(["apply", "--workload", workload, "--target", target], options); }
  sync(workload: string, target?: string, options: CallOptions = {}): Promise<unknown> { validateHandle(workload); return this.json(["sync", "--workload", workload, ...targetFlag(target), "--once"], options); }

  run(workload: string, target?: string, options: CallOptions = {}): Promise<LocalResult> {
    validateHandle(workload);
    return this.invoke(["run", "--workload", workload, ...targetFlag(target)], { ...options, inheritOutput: true });
  }

  flowRun(workload: string, pin: CompleteRemotePin, options: CallOptions = {}): Promise<LocalResult> {
    validateHandle(workload);
    validateRemotePin(pin);
    return this.invoke(["flow", "run", "--workload", workload, ...flags(pin, remoteFlags), "--non-interactive"], { ...options, inheritOutput: true });
  }

  private async json(args: string[], options: LocalRunOptions): Promise<unknown> {
    return boundedJSON((await this.invoke(args, options)).stdout);
  }

  private invoke(args: string[], options: LocalRunOptions = {}): Promise<LocalResult> {
    return observe(this.#observer, "local_cli", options.signal, async () => {
      await verifyExecutable(this.#executable, this.#digest);
      return spawnOwned(this.#executable, ["--config", this.#config, "--state-dir", this.#state, ...args], this.#environment, options);
    });
  }
}

const remoteFlags = { configSource: "--config-source", configRef: "--config-ref", configPath: "--config-path", configSHA256: "--config-sha256" };

function doctorMode(options: { readonly fix?: boolean; readonly identityOnly?: boolean }): string[] {
  if (options.fix) return ["--fix"];
  if (options.identityOnly) return ["--identity-only"];
  return [];
}

function object(value: unknown): Record<string, unknown> {
  if (!isPlainObject(value)) throw new CquayError("protocol");
  return value;
}
function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) return false;
  const prototype = Reflect.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}
function structured(root: Record<string, unknown>): unknown {
  if (!Object.hasOwn(root, "structuredContent")) throw new CquayError("protocol");
  const value = root.structuredContent;
  if (!safeJSON(value)) throw new CquayError("protocol");
  const serialized = JSON.stringify(value);
  if (encodedLength(serialized) > MAX_RESPONSE_BYTES) throw new CquayError("protocol");
  return value;
}
function parseTools(page: unknown): readonly ToolDescriptor[] {
  const tools = object(page).tools;
  if (!Array.isArray(tools) || tools.length > MAX_TOOLS) throw new CquayError("protocol");
  const result: ToolDescriptor[] = [];
  for (const item of tools) result.push(parseTool(item, result));
  return Object.freeze(result);
}
function parseTool(item: unknown, existing: readonly ToolDescriptor[]): ToolDescriptor {
  const value = object(item);
  const { name, description } = value;
  if (!validHandle(name) || !KNOWN_TOOLS.has(name) || existing.some(tool => tool.name === name)) throw new CquayError("protocol");
  if (description !== undefined && (typeof description !== "string" || description.length > 4096)) throw new CquayError("protocol");
  return description === undefined ? { name } : { name, description };
}
function parseWorkloads(value: unknown): readonly string[] {
  const result = object(value);
  const workloads = result.workloads;
  const valid = Array.isArray(workloads) && workloads.length <= MAX_WORKLOADS && keysOnly(result, ["workloads"]) && workloads.every(validHandle) && new Set(workloads).size === workloads.length && sorted(workloads);
  if (!valid) throw new CquayError("protocol");
  return Object.freeze([...workloads]);
}
function parseStatus(value: unknown, workload: string): WorkloadStatus {
  const result = object(value);
  const mode = result.mode;
  const state = result.state;
  if (result.workload !== workload || !validMode(mode) || typeof state !== "string" || !STATES.has(state) || !keysOnly(result, ["workload", "mode", "state"])) throw new CquayError("protocol");
  return Object.freeze({ workload, mode, state });
}
function parseRunResult(value: unknown): RunResult {
  const result = object(value);
  const id = result.id;
  const status = result.status;
  const exitCode = result.exit_code;
  if (!validHandle(id) || !validRunStatus(status) || !validExitCode(exitCode) || !keysOnly(result, ["id", "status", "exit_code"])) throw new CquayError("protocol");
  return Object.freeze({ id, status, exitCode });
}
function parseSecret(value: unknown, key: string): SecretDisclosure {
  const result = object(value);
  const secret = result.value;
  const version = result.version;
  const expiresAt = result.expires_at;
  const expiry = parseExpiry(expiresAt);
  if (expiry === "expired") throw new CquayError("expired");
  if (result.alias !== key || !validSecretValue(secret) || !validProviderVersion(version) || expiry !== "valid" || !keysOnly(result, ["alias", "value", "version", "expires_at"])) throw new CquayError("protocol");
  return new SecretDisclosure(key, secret, version, expiryValue(expiresAt));
}
function validMode(value: unknown): value is WorkloadStatus["mode"] {
  return value === "offline" || value === "online" || value === "hybrid";
}
function validRunStatus(value: unknown): value is RunResult["status"] {
  return value === "completed" || value === "failed" || value === "running" || value === "cancelled" || value === "unknown";
}
function validExitCode(value: unknown): value is number {
  return typeof value === "number" && Number.isInteger(value) && value >= 0 && value <= 255;
}
function expiryValue(value: unknown): string | undefined {
  return typeof value === "string" && !isZeroExpiry(value) ? value : undefined;
}
function keysOnly(value: Record<string, unknown>, allowed: readonly string[]): boolean { return Object.keys(value).every(key => allowed.includes(key)); }
function validHandle(value: unknown): value is string { return typeof value === "string" && HANDLE.test(value) && value !== "." && value !== ".."; }
function validateHandle(value: string): void { if (!validHandle(value)) throw new CquayError("invalid"); }
function validateText(value: string): void { if (typeof value !== "string" || value.length > 4096 || value.includes("\0")) throw new CquayError("invalid"); }
function validatePointer(value: string): void {
  validateText(value);
  if (value === "") return;
  if (!value.startsWith("/")) throw new CquayError("invalid");
  for (let index = 0; index < value.length; index += 1) if (value[index] === "~" && value[index + 1] !== "0" && value[index + 1] !== "1") throw new CquayError("invalid");
}
function validSecretValue(value: unknown): value is string { return typeof value === "string" && encodedLength(value) <= MAX_SECRET_VALUE_BYTES; }
function validProviderVersion(value: unknown): value is string | undefined {
  return value === undefined || (typeof value === "string" && encodedLength(value) <= MAX_PROVIDER_VERSION_BYTES && !hasControlCharacter(value));
}
function hasControlCharacter(value: string): boolean { return /[\u0000-\u001f\u007f]/u.test(value); }
function parseExpiry(value: unknown): "valid" | "expired" | "malformed" {
  if (value === undefined || isZeroExpiry(value)) return "valid";
  if (typeof value !== "string" || value.length === 0 || value.length > 128 || !RFC3339_DATETIME.test(value)) return "malformed";
  const milliseconds = Date.parse(value);
  if (!Number.isFinite(milliseconds)) return "malformed";
  return milliseconds > Date.now() ? "valid" : "expired";
}
function isZeroExpiry(value: unknown): boolean { return value === "0001-01-01T00:00:00Z" || value === "0001-01-01T00:00:00.000Z"; }
function sorted(values: readonly string[]): boolean {
  for (let index = 1; index < values.length; index += 1) {
    const previous = values[index - 1];
    const current = values[index];
    if (previous === undefined || current === undefined || previous > current) return false;
  }
  return true;
}
function isAbort(value: unknown): boolean { return typeof value === "object" && value !== null && (value as { name?: unknown }).name === "AbortError"; }
function flags<T extends object>(input: T, names: Readonly<Partial<Record<keyof T, string>>>): string[] {
  const result: string[] = [];
  for (const key of Object.keys(names) as (keyof T)[]) {
    const flag = names[key];
    const value = (input as Record<keyof T, unknown>)[key];
    if (value !== undefined) {
      if (flag === undefined) continue;
      validateFlagValue(value);
      result.push(flag, value);
    }
  }
  return result;
}
function validateFlagValue(value: unknown): asserts value is string {
  if (typeof value !== "string" || value.length === 0 || value.length > 4096 || value.includes("\0")) throw new CquayError("invalid");
}
function optionalFlag(flag: string, value: string | undefined): string[] { return value === undefined ? [] : flags({ value }, { value: flag }); }
function expected(value: string | undefined): string[] {
  if (value === undefined) return [];
  if (!HEX_64.test(value)) throw new CquayError("invalid");
  return ["--expect-sha256", value];
}
function targetFlag(target: string | undefined): string[] {
  if (target === undefined) return [];
  validateHandle(target);
  return ["--target", target];
}
function validateRemotePin(value: RemotePin): void {
  if (!hasRemotePin(value)) return;
  if (!completeRemotePin(value)) throw new CquayError("invalid");
}
function hasRemotePin(value: RemotePin): boolean { return Object.values(value).some(item => item !== undefined); }
function completeRemotePin(value: RemotePin): boolean {
  return validRemoteSource(value.configSource) && validRemotePath(value.configPath) && /^[a-f0-9]{40}$/.test(value.configRef ?? "") && HEX_64.test(value.configSHA256 ?? "");
}
function validRemoteSource(value: string | undefined): boolean {
  try {
    const source = new URL(value ?? "");
    return validRemoteURL(source, value);
  } catch {
    return false;
  }
}
function validRemoteURL(source: URL, original: string | undefined): boolean {
  return source.protocol === "https:" && source.host !== "" && hasHTTPSAuthority(original) && source.username === "" && source.password === "" && source.search === "" && source.hash === "" && !hasRemoteDelimiter(original) && original.length <= 4096;
}
function hasHTTPSAuthority(value: string | undefined): value is string {
  return typeof value === "string" && /^https:\/\//iu.test(value);
}
function hasRemoteDelimiter(value: string | undefined): boolean { return value?.includes("?") === true || value?.includes("#") === true; }
function validRemotePath(value: string | undefined): boolean { return typeof value === "string" && value.length > 0 && value.length <= 1024 && !value.includes("\0"); }
function isVersion(value: unknown): value is { version: string; commit: string; go: string; maturity: string } {
  return isPlainObject(value) && keysOnly(value, ["version", "commit", "go", "maturity"]) && typeof value.version === "string" && semanticMajor(value.version) !== undefined && typeof value.commit === "string" && typeof value.go === "string" && typeof value.maturity === "string";
}
function semanticMajor(value: string): 1 | 2 | undefined {
  if (encodedLength(value) > MAX_RUNTIME_VERSION_BYTES) return undefined;
  const unprefixed = value.startsWith("v") ? value.slice(1) : value;
  const coreAndPrerelease = splitBuild(unprefixed);
  if (coreAndPrerelease === undefined || !validBuild(coreAndPrerelease.optional)) return undefined;
  const core = splitPrerelease(coreAndPrerelease.required);
  if (core === undefined || !validPrerelease(core.optional) || !validVersionCore(core.required)) return undefined;
  return core.required.split(".")[0] === "1" ? 1 : 2;
}
function splitBuild(value: string): { readonly required: string; readonly optional: string | undefined } | undefined {
  const index = value.indexOf("+");
  if (index < 0) return { required: value, optional: undefined };
  if (index !== value.lastIndexOf("+")) return undefined;
  return splitVersionSuffix(value, index);
}
function splitPrerelease(value: string): { readonly required: string; readonly optional: string | undefined } | undefined {
  const index = value.indexOf("-");
  return index < 0 ? { required: value, optional: undefined } : splitVersionSuffix(value, index);
}
function splitVersionSuffix(value: string, index: number): { readonly required: string; readonly optional: string | undefined } | undefined {
  const optional = value.slice(index + 1);
  return optional === "" ? undefined : { required: value.slice(0, index), optional };
}
function validVersionCore(value: string): boolean {
  const components = value.split(".");
  return components.length === 3 && components.every(validCanonicalNumber);
}
function validPrerelease(value: string | undefined): boolean {
  return value === undefined || validVersionIdentifiers(value, true);
}
function validBuild(value: string | undefined): boolean {
  return value === undefined || validVersionIdentifiers(value, false);
}
function validVersionIdentifiers(value: string, rejectPaddedNumbers: boolean): boolean {
  return value.split(".").every(identifier => validVersionIdentifier(identifier, rejectPaddedNumbers));
}
function validVersionIdentifier(value: string, rejectPaddedNumbers: boolean): boolean {
  if (value === "" || !asciiAlphaNumericHyphen(value)) return false;
  return !rejectPaddedNumbers || !allDigits(value) || validCanonicalNumber(value);
}
function validCanonicalNumber(value: string): boolean {
  return value === "0" || (value.length > 0 && value.charCodeAt(0) >= 49 && value.charCodeAt(0) <= 57 && allDigits(value));
}
function allDigits(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index);
    if (code < 48 || code > 57) return false;
  }
  return true;
}
function asciiAlphaNumericHyphen(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index);
    const digit = code >= 48 && code <= 57;
    const uppercase = code >= 65 && code <= 90;
    const lowercase = code >= 97 && code <= 122;
    if (!digit && !uppercase && !lowercase && code !== 45) return false;
  }
  return true;
}
function validateLocalOptions(options: LocalCLIOptions): void {
  if (!isAbsolute(options.executablePath) || !HEX_64.test(options.expectedSHA256) || !isAbsolute(options.configPath) || !isAbsolute(options.stateDirectory)) throw new CquayError("invalid");
}
function cleanEnvironment(source: Readonly<Record<string, string>> | undefined): Readonly<Record<string, string>> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(source ?? {})) { validateEnvironmentEntry(key, value); result[key] = value; }
  return Object.freeze(result);
}
function validateEnvironmentEntry(key: string, value: string): void {
  const valid = /^[_a-zA-Z]\w*$/.test(key) && !DENIED_ENV_PREFIXES.some(prefix => key.startsWith(prefix)) && typeof value === "string" && !value.includes("\0") && value.length <= 32768;
  if (!valid) throw new CquayError("invalid");
}
function safeJSON(value: unknown): boolean { return isSafeJSONValue(value, 0); }
function isSafeJSONValue(value: unknown, depth: number): boolean {
  if (depth > MAX_JSON_DEPTH) return false;
  if (isSafeJSONScalar(value)) return true;
  if (Array.isArray(value)) return value.length <= MAX_JSON_NODES && value.every(item => isSafeJSONValue(item, depth + 1));
  return isPlainObject(value) && Object.values(value).every(item => isSafeJSONValue(item, depth + 1));
}
function isSafeJSONScalar(value: unknown): boolean {
  return value === null || typeof value === "string" || typeof value === "boolean" || (typeof value === "number" && Number.isFinite(value) && (!Number.isInteger(value) || Number.isSafeInteger(value)));
}
interface JSONValidationState { readonly seen: WeakSet<object>; nodes: number; }
function strictJSON(value: unknown): string {
  const state: JSONValidationState = { seen: new WeakSet(), nodes: 0 };
  validateJSONValue(value, 0, state);
  const serialized = JSON.stringify(value);
  if (encodedLength(serialized) > MAX_STDIO_BYTES) throw new CquayError("invalid");
  return serialized;
}
function validateJSONValue(value: unknown, depth: number, state: JSONValidationState): void {
  if (depth > MAX_JSON_DEPTH || ++state.nodes > MAX_JSON_NODES) throw new CquayError("invalid");
  if (value === null || typeof value === "string" || typeof value === "boolean") return;
  if (typeof value === "number") {
    validateJSONNumber(value);
    return;
  }
  if (Array.isArray(value)) {
    validateJSONCollection(value, depth, state);
    return;
  }
  if (isPlainObject(value)) {
    validateJSONObject(value, depth, state);
    return;
  }
  throw new CquayError("invalid");
}
function validateJSONNumber(value: number): void { if (!Number.isFinite(value) || (Number.isInteger(value) && !Number.isSafeInteger(value))) throw new CquayError("invalid"); }
function validateJSONCollection(value: readonly unknown[], depth: number, state: JSONValidationState): void {
  if (state.seen.has(value)) throw new CquayError("invalid");
  state.seen.add(value);
  for (const item of value) validateJSONValue(item, depth + 1, state);
}
function validateJSONObject(value: Record<string, unknown>, depth: number, state: JSONValidationState): void {
  if (state.seen.has(value)) throw new CquayError("invalid");
  state.seen.add(value);
  for (const [key, item] of Object.entries(value)) { validateText(key); validateJSONValue(item, depth + 1, state); }
}
function encodedLength(value: string): number { return new TextEncoder().encode(value).byteLength; }
async function verifyExecutable(path: string, expectedDigest: string): Promise<string> {
  try {
    const link = await lstat(path);
    if (link.isSymbolicLink() || !link.isFile()) throw new CquayError("integrity");
    const canonical = await realpath(path);
    const actualDigest = await hashRegularFile(canonical);
    if (!timingSafeEqual(Buffer.from(actualDigest, "hex"), Buffer.from(expectedDigest, "hex"))) throw new CquayError("integrity");
    return canonical;
  } catch (error) {
    if (error instanceof CquayError) throw error;
    throw new CquayError("integrity");
  }
}
async function hashRegularFile(path: string): Promise<string> {
  const handle = await open(path, "r");
  try {
    const stat = await handle.stat();
    if (!stat.isFile() || stat.size < 1 || stat.size > MAX_EXECUTABLE_BYTES) throw new CquayError("integrity");
    const hash = createHash("sha256");
    const buffer = Buffer.allocUnsafe(64 * 1024);
    for (let position = 0; position < stat.size;) {
      const { bytesRead } = await handle.read(buffer, 0, Math.min(buffer.length, stat.size - position), position);
      if (bytesRead === 0) throw new CquayError("integrity");
      hash.update(buffer.subarray(0, bytesRead));
      position += bytesRead;
    }
    return hash.digest("hex");
  } finally { await handle.close(); }
}
async function observe<T>(observer: Observer | undefined, action: Observation["action"], signal: AbortSignal | undefined, operation: () => Promise<T>): Promise<T> {
  const startedAt = Date.now();
  let outcome: Observation["outcome"] = "unavailable";
  try {
    if (signal?.aborted) throw new CquayError("cancelled");
    const result = await operation();
    outcome = "success";
    return result;
  } catch (error) {
    const safe = observationError(error, signal);
    outcome = safe.category;
    throw safe;
  } finally {
    if (observer !== undefined) try { await observer({ action, outcome, durationMs: Math.max(0, Date.now() - startedAt) }); } catch { /* observer failure cannot affect operation */ }
  }
}
function observationError(error: unknown, signal: AbortSignal | undefined): CquayError {
  if (error instanceof CquayError) return error;
  if (signal?.aborted || isAbort(error)) return new CquayError("cancelled");
  return new CquayError("unavailable");
}
function boundedJSON(text: string): unknown {
  if (encodedLength(text) > MAX_RESPONSE_BYTES) throw new CquayError("protocol");
  const value = JSON.parse(text) as unknown;
  if (!safeJSON(value)) throw new CquayError("protocol");
  return value;
}
function spawnOwned(executable: string, args: string[], environment: Readonly<Record<string, string>>, options: LocalRunOptions): Promise<LocalResult> {
  return new Promise((resolve, reject) => {
    if (options.signal?.aborted) { reject(new CquayError("cancelled")); return; }
    const child = startChild(executable, args, environment, options.inheritOutput === true);
    if (child === undefined) { reject(new CquayError("process")); return; }
    const state = new ProcessState(options.signal, resolve, reject);
    bindProcessLifecycle(child, state, options);
    writeInput(child, options.stdin, () => { state.overflow(child); });
  });
}
class ProcessState {
  #stdout = ""; #stderr = ""; #stderrBytes = 0; #overflow = false; #stdinFailure = false; #childFailure = false; #cancelled = false; #settled = false;
  #killTimer: ReturnType<typeof setTimeout> | undefined; #fallbackTimer: ReturnType<typeof setTimeout> | undefined;
  #abortListener: (() => void) | undefined;
  readonly #signal: AbortSignal | undefined;
  readonly #resolve: (value: LocalResult) => void;
  readonly #reject: (reason: CquayError) => void;

  constructor(
    signal: AbortSignal | undefined,
    resolve: (value: LocalResult) => void,
    reject: (reason: CquayError) => void
  ) {
    this.#signal = signal;
    this.#resolve = resolve;
    this.#reject = reject;
  }
  appendStdout(chunk: Buffer | string, child: ChildProcess): void {
    const next = appendBounded(this.#stdout, chunk);
    if (next === undefined) {
      this.overflow(child);
      return;
    }
    this.#stdout = next;
  }
  appendStderr(chunk: Buffer | string, child: ChildProcess): void {
    this.#stderrBytes += Buffer.byteLength(chunk);
    if (this.#stderrBytes > MAX_STDIO_BYTES) {
      this.overflow(child);
      return;
    }
    this.#stderr += chunk.toString();
  }
  cancel(child: ChildProcess): void { this.#cancelled = true; this.terminate(child); }
  overflow(child: ChildProcess): void { this.#overflow = true; this.terminate(child); }
  stdinError(child: ChildProcess): void { this.#stdinFailure = true; this.terminate(child); }
  childError(child: ChildProcess): void { this.#childFailure = true; this.terminate(child); }
  bindAbort(child: ChildProcess): void {
    this.#abortListener = () => { this.cancel(child); };
    this.#signal?.addEventListener("abort", this.#abortListener, { once: true });
  }
  close(code: number | null, allowNonZero: boolean): void {
    const result = { exitCode: code ?? 1, stdout: this.#stdout };
    this.finish(this.closeError(code, allowNonZero), result);
  }
  terminate(child: ChildProcess): void {
    if (this.#settled || this.#killTimer !== undefined) return;
    try { child.kill("SIGTERM"); } catch { /* wait for close */ }
    this.#killTimer = setTimeout(() => { killChild(child, "SIGKILL"); }, 5000);
    this.#fallbackTimer = setTimeout(() => { this.finish(this.terminalError() ?? new CquayError("process")); }, 6000);
  }
  finish(error?: CquayError, result?: LocalResult): void {
    if (this.#settled) return;
    this.#settled = true;
    if (this.#killTimer !== undefined) clearTimeout(this.#killTimer);
    if (this.#fallbackTimer !== undefined) clearTimeout(this.#fallbackTimer);
    if (this.#abortListener !== undefined) this.#signal?.removeEventListener("abort", this.#abortListener);
    if (error !== undefined) {
      this.#reject(error);
      return;
    }
    if (result === undefined) {
      this.#reject(new CquayError("process"));
      return;
    }
    this.#resolve(result);
  }

  private closeError(code: number | null, allowNonZero: boolean): CquayError | undefined {
    const terminal = this.terminalError();
    if (terminal !== undefined) return terminal;
    return code !== 0 && !allowNonZero ? nativeFailure(this.#stderr) : undefined;
  }

  private terminalError(): CquayError | undefined {
    if (this.#cancelled) return new CquayError("cancelled");
    if (this.#overflow) return new CquayError("protocol");
    if (this.#stdinFailure || this.#childFailure) return new CquayError("process");
    return undefined;
  }
}
function bindProcessLifecycle(child: ChildProcess, state: ProcessState, options: LocalRunOptions): void {
  state.bindAbort(child);
  if (!options.inheritOutput) bindCapturedOutput(child, state);
  child.stdin?.once("error", () => { state.stdinError(child); });
  child.once("error", () => { state.childError(child); });
  child.once("close", code => { state.close(code, options.inheritOutput === true); });
}
function bindCapturedOutput(child: ChildProcess, state: ProcessState): void {
  child.stdout?.on("data", (chunk: Buffer | string) => { state.appendStdout(chunk, child); });
  child.stderr?.on("data", (chunk: Buffer | string) => { state.appendStderr(chunk, child); });
}
function killChild(child: ChildProcess, signal: NodeJS.Signals): void {
  try { child.kill(signal); } catch { /* close owns normal reaping */ }
}
function startChild(executable: string, args: string[], environment: Readonly<Record<string, string>>, inheritOutput: boolean): ChildProcess | undefined {
  try {
    return spawn(executable, args, {
      shell: false,
      windowsHide: true,
      env: environment,
      stdio: inheritOutput ? ["pipe", "inherit", "inherit"] : ["pipe", "pipe", "pipe"]
    });
  } catch {
    return undefined;
  }
}
function appendBounded(current: string, chunk: Buffer | string): string | undefined {
  const next = current + chunk.toString();
  return encodedLength(next) <= MAX_STDIO_BYTES ? next : undefined;
}
function nativeFailure(stderr: string): CquayError {
  if (stderr === "") return new CquayError("process");
  const category = nativeErrorCategory(stderr);
  return category === undefined ? new CquayError("protocol") : new CquayError(category);
}
function nativeErrorCategory(stderr: string): ErrorCategory | undefined {
  try {
    const value = object(JSON.parse(stderr));
    if (!keysOnly(value, ["error"]) || Object.keys(value).length !== 1 || typeof value.error !== "string") return undefined;
    return NATIVE_ERROR_CATEGORIES[value.error];
  } catch { return undefined; }
}
function writeInput(child: ChildProcess, input: string | undefined, onOverflow: () => void): void {
  if (input !== undefined && encodedLength(input) > MAX_STDIO_BYTES) {
    onOverflow();
    return;
  }
  child.stdin?.end(input);
}
