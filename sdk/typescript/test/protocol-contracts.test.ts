import assert from "node:assert/strict";
import test from "node:test";
import { CqClient, CqError, LocalCLIClient, officialMCPAdapter } from "../src/index.ts";

const protocol = (error: unknown) => error instanceof CqError && error.category === "protocol";
const invalid = (error: unknown) => error instanceof CqError && error.category === "invalid";
function client(value: unknown): CqClient {
  return new CqClient({ async listTools() { return value; }, async callTool() { return value; } });
}
function response(value: unknown): CqClient { return client({ structuredContent: value }); }

test("MCP rejects missing, unsafe, oversized and sparse structured results", async () => {
  for (const result of [null, {}, { structuredContent: Number.NaN },
    { structuredContent: { workloads: Array(1) } },
    { structuredContent: "x".repeat(1024 * 1024 + 1) },
    { structuredContent: Array.from({ length: 34 }).reduce<unknown>(value => [value], null) }]) {
    await assert.rejects(client(result).listWorkloads(), protocol);
  }
});

test("discovery preserves descriptions and rejects malformed or duplicate capabilities", async () => {
  for (const tools of [null, Array(5).fill({ name: "get_status" }),
    [{ name: "invented" }], [{ name: "get_status" }, { name: "get_status" }],
    [{ name: "get_status", description: false }],
    [{ name: "get_status", description: "x".repeat(4097) }]]) {
    await assert.rejects(client({ pages: [{ tools }] }).discoverTools(), protocol);
  }
  assert.deepEqual(await client({ pages: [{ tools: [{ name: "get_status", description: "Status" }] }] }).discoverTools(),
    [{ name: "get_status", description: "Status" }]);
  const adapter = officialMCPAdapter({
    async listTools(params, options) { assert.equal(params, undefined); assert.equal(options, undefined); return { tools: [] }; },
    async callTool() { return { structuredContent: { workloads: [] } }; }
  }, {});
  assert.deepEqual(await adapter.listTools(), { pages: [{ tools: [] }] });
  await adapter.callTool({ name: "list_workloads", arguments: {} });
});

test("all published run statuses and expiry forms retain their typed meaning", async () => {
  for (const mode of ["offline", "hybrid"]) {
    assert.deepEqual(await response({ workload: "api", mode, state: "ready" }).getStatus("api"),
      { workload: "api", mode, state: "ready" });
  }
  for (const status of ["failed", "running", "cancelled", "unknown"]) {
    assert.deepEqual(await response({ id: "job", status, exit_code: 1 }).runWorkload("api"),
      { id: "job", status, exitCode: 1 });
  }
  await assert.rejects(response({ id: "job", status: "invented", exit_code: 0 }).runWorkload("api"), protocol);
  const future = "9999-12-31T23:59:59Z";
  const disclosure = await response({ alias: "TOKEN", value: "synthetic", expires_at: future }).disclosure().readSecret("api", "TOKEN");
  assert.equal(disclosure.expiresAt, future);
  assert.equal(disclosure.version, undefined);
  for (const expires_at of ["0001-01-01T00:00:00Z", "0001-01-01T00:00:00.000Z"]) {
    const value = await response({ alias: "TOKEN", value: "synthetic", expires_at }).disclosure().readSecret("api", "TOKEN");
    assert.equal(value.expiresAt, undefined);
  }
  await assert.rejects(response({ alias: "TOKEN", value: "synthetic", expires_at: "9999-99-99T00:00:00Z" }).disclosure().readSecret("api", "TOKEN"), protocol);
});

test("runtime version parsing rejects malformed suffixes and oversized identities", async () => {
  for (const version of ["1.0.0+one+two", "1.0.0+", "1.0.0-", "1.0.0+invalid!", "x".repeat(257)]) {
    const local = { json: async () => ({ version, commit: "synthetic", go: "go1.27.4", maturity: "development" }) } as unknown as LocalCLIClient;
    await assert.rejects(LocalCLIClient.prototype.version.call(local), protocol);
  }
});

test("local mutations reject invalid preconditions before dispatch", () => {
  let dispatched = false;
  const local = { json: async () => { dispatched = true; return {}; } } as unknown as LocalCLIClient;
  assert.throws(() => LocalCLIClient.prototype.configInit.call(local, "relative.json"), invalid);
  assert.throws(() => LocalCLIClient.prototype.configDelete.call(local, "/workloads", "invalid"), invalid);
  const pin = { output: "/synthetic/policy.json", configSource: "https://example.test/config", configPath: "policy.json", configRef: "a".repeat(40), configSHA256: "b".repeat(64) };
  for (const entry of [{ ...pin, configRef: undefined }, { ...pin, configSHA256: undefined },
    { ...pin, configSource: undefined }, { ...pin, configPath: undefined }]) {
    assert.throws(() => LocalCLIClient.prototype.bootstrap.call(local, entry), invalid);
  }
  assert.throws(() => LocalCLIClient.prototype.bootstrap.call(local, { ...pin, output: "/synthetic/\0policy" }), invalid);
  assert.equal(dispatched, false);
});

test("config init without an output uses the explicitly configured policy", async () => {
  const calls: string[][] = [];
  const local = { json: async (args: string[]) => { calls.push(args); return {}; } } as unknown as LocalCLIClient;
  await LocalCLIClient.prototype.configInit.call(local);
  assert.deepEqual(calls, [["config", "init"]]);
});


test("transport cancellation and successful observers preserve safe outcomes", async () => {
  for (const duringCall of [false, true]) {
    const controller = new AbortController();
    const value = new CqClient({
      async listTools() { return {}; },
      async callTool() {
        if (duringCall) controller.abort();
        throw duringCall ? new Error("synthetic transport detail") : { name: "AbortError" };
      }
    });
    await assert.rejects(value.listWorkloads({ signal: controller.signal }),
      error => error instanceof CqError && error.category === "cancelled");
  }
  const observations: unknown[] = [];
  const value = new CqClient({ async listTools() { return {}; }, async callTool() { return { structuredContent: { workloads: [] } }; } },
    { observer: observation => { observations.push(observation); } });
  assert.deepEqual(await value.listWorkloads(), []);
  assert.equal(observations.length, 1);
});


test("bootstrap without remote pins stays local while flow requires complete pins", async () => {
  const calls: string[][] = [];
  const local = { json: async (args: string[]) => { calls.push(args); return {}; } } as unknown as LocalCLIClient;
  await LocalCLIClient.prototype.bootstrap.call(local, { output: "/synthetic/policy.json" });
  assert.deepEqual(calls, [["bootstrap", "--output", "/synthetic/policy.json", "--non-interactive"]]);
  assert.throws(() => LocalCLIClient.prototype.flowRun.call(local, "api", {} as never), invalid);
  assert.equal(calls.length, 1);
});

test("failure observers receive only safe categories and cannot replace errors", async () => {
  const observations: Array<{ action: string; outcome: string; durationMs: number }> = [];
  const value = new CqClient({
    async listTools() { return {}; },
    async callTool() { throw new Error("synthetic provider detail"); }
  }, { observer: async observation => { observations.push(observation); throw new Error("synthetic observer detail"); } });
  await assert.rejects(value.listWorkloads(), error => error instanceof CqError && error.category === "unavailable" && !error.message.includes("detail"));
  assert.deepEqual(observations.map(({ action, outcome }) => ({ action, outcome })),
    [{ action: "list_workloads", outcome: "unavailable" }]);
});
