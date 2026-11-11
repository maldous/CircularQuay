import assert from "node:assert/strict";
import test from "node:test";
import { CqClient, CqError, LocalCLIClient } from "../src/index.ts";

test("local configuration CRUD sends bounded JSON through stdin", async () => {
  const calls: Array<{ args: string[]; options: unknown }> = [];
  const client = recordingClient(calls);
  await LocalCLIClient.prototype.configSet.call(client, "/workloads/api", { enabled: true }, "a".repeat(64));
  await LocalCLIClient.prototype.configReplace.call(client, { schema_version: 1 });
  await LocalCLIClient.prototype.configDelete.call(client, "/workloads/api");
  await LocalCLIClient.prototype.configGet.call(client, "/workloads/api");
  await LocalCLIClient.prototype.configList.call(client);
  assert.deepEqual(calls[0]?.args, ["config", "set", "--pointer", "/workloads/api", "--value-file", "-", "--expect-sha256", "a".repeat(64)]);
  assert.equal((calls[0]?.options as { stdin?: string }).stdin, '{"enabled":true}');
  assert.deepEqual(calls[1]?.args, ["config", "replace", "--file", "-"]);
  assert.equal((calls[1]?.options as { stdin?: string }).stdin, '{"schema_version":1}');
  assert.deepEqual(calls.slice(2).map(call => call.args), [["config", "delete", "--pointer", "/workloads/api"], ["config", "get", "--pointer", "/workloads/api"], ["config", "list", "--pointer", ""]]);
});

test("local configuration rejects malformed pointers and nonportable JSON", async () => {
  const client = recordingClient([]);
  assert.throws(() => LocalCLIClient.prototype.configGet.call(client, "workloads/api"), invalid);
  assert.throws(() => LocalCLIClient.prototype.configSet.call(client, "/~2", "value"), invalid);
  assert.throws(() => LocalCLIClient.prototype.configReplace.call(client, Number.POSITIVE_INFINITY), invalid);
  assert.throws(() => LocalCLIClient.prototype.configReplace.call(client, Number.MAX_SAFE_INTEGER + 1), invalid);
  const cyclic: { self?: unknown } = {};
  cyclic.self = cyclic;
  assert.throws(() => LocalCLIClient.prototype.configReplace.call(client, cyclic), invalid);
});

test("local operations validate modes, targets, and remote pins before dispatch", async () => {
  const calls: Array<{ args: string[]; options: unknown }> = [];
  const client = recordingClient(calls);
  await LocalCLIClient.prototype.doctor.call(client, { fix: true });
  await LocalCLIClient.prototype.doctor.call(client, { identityOnly: true });
  await LocalCLIClient.prototype.status.call(client, "api");
  await LocalCLIClient.prototype.snapshot.call(client, "api", "user");
  await LocalCLIClient.prototype.apply.call(client, "api", "user");
  await LocalCLIClient.prototype.sync.call(client, "api");
  await LocalCLIClient.prototype.bootstrap.call(client, validPin("/absolute/policy.json"));
  assert.deepEqual(calls.map(call => call.args), [
    ["doctor", "--fix"], ["doctor", "--identity-only"], ["status", "--workload", "api", "--json"],
    ["snapshot", "--workload", "api", "--target", "user"], ["apply", "--workload", "api", "--target", "user"],
    ["sync", "--workload", "api", "--once"], ["bootstrap", "--config-source", "https://github.com/acme/policy", "--config-ref", "a".repeat(40), "--config-path", "policy.json", "--config-sha256", "b".repeat(64), "--output", "/absolute/policy.json", "--non-interactive"]
  ]);
  assert.throws(() => LocalCLIClient.prototype.doctor.call(client, { fix: true, identityOnly: true }), invalid);
  assert.throws(() => LocalCLIClient.prototype.apply.call(client, "..", "user"), invalid);
  assert.throws(() => LocalCLIClient.prototype.bootstrap.call(client, validPin("relative.json")), invalid);
  assert.throws(() => LocalCLIClient.prototype.bootstrap.call(client, { ...validPin("/absolute/policy.json"), configSource: "https://user:token@example.test/policy" }), invalid);
});

test("MCP results enforce ordering, shape, and safe error categories", async () => {
  await assert.rejects(mcp({ workloads: ["z", "a"] }).listWorkloads(), protocol);
  await assert.rejects(mcp({ workloads: ["api", "api"] }).listWorkloads(), protocol);
  await assert.rejects(mcp({ workloads: ["api"], extra: true }).listWorkloads(), protocol);
  await assert.rejects(mcp({ workload: "api", mode: "offline", state: "invented" }).getStatus("api"), protocol);
  await assert.rejects(mcp({ id: "run", status: "completed", exit_code: 256 }).runWorkload("api"), protocol);
  await assert.rejects(mcp({ alias: "TOKEN", value: "synthetic", version: "bad\u0000version" }).disclosure().readSecret("api", "TOKEN"), protocol);
});

test("MCP observer is metadata-only and cannot change an operation outcome", async () => {
  const observations: Array<{ action: string; outcome: string; durationMs: number }> = [];
  const client = new CqClient(transport({ workloads: ["api"] }), {
    async observer(observation) { observations.push(observation); throw new Error("observer failure"); }
  });
  assert.deepEqual(await client.listWorkloads(), ["api"]);
  assert.deepEqual(observations.map(item => [item.action, item.outcome, item.durationMs >= 0]), [["list_workloads", "success", true]]);
});

test("MCP maps aborted and unavailable transports without retaining detail", async () => {
  const controller = new AbortController();
  controller.abort();
  await assert.rejects(mcp({ workloads: [] }).listWorkloads({ signal: controller.signal }), cancelled);
  const client = new CqClient({ async listTools() { return { pages: [] }; }, async callTool() { throw new Error("provider endpoint synthetic detail"); } });
  await assert.rejects(client.listWorkloads(), unavailable);
});

function recordingClient(calls: Array<{ args: string[]; options: unknown }>): LocalCLIClient {
  return { json: async (args: string[], options: unknown) => { calls.push({ args, options }); return {}; } } as unknown as LocalCLIClient;
}

function validPin(output: string): { configSource: string; configRef: string; configPath: string; configSHA256: string; output: string } {
  return { configSource: "https://github.com/acme/policy", configRef: "a".repeat(40), configPath: "policy.json", configSHA256: "b".repeat(64), output };
}

function mcp(structuredContent: Record<string, unknown>): CqClient {
  return new CqClient(transport(structuredContent));
}

function transport(structuredContent: Record<string, unknown>) {
  return { async listTools() { return { pages: [] }; }, async callTool() { return { structuredContent }; } };
}

function invalid(error: unknown): boolean { return category(error, "invalid"); }
function protocol(error: unknown): boolean { return category(error, "protocol"); }
function cancelled(error: unknown): boolean { return category(error, "cancelled"); }
function unavailable(error: unknown): boolean { return category(error, "unavailable"); }
function category(error: unknown, expected: string): boolean { return error instanceof CqError && error.category === expected; }
