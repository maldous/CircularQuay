import assert from "node:assert/strict";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { CqClient, CqError, LocalCLIClient, officialMCPAdapter } from "../src/index.ts";

test("validates a status result and forwards cancellation", async () => {
  let seen: AbortSignal | undefined;
  const client = new CqClient({
    async listTools() { return { pages: [] }; },
    async callTool(call, options) { seen = options?.signal; assert.equal(call.name, "get_status"); return { structuredContent: { workload: "api", mode: "online", state: "ready" } }; }
  });
  const controller = new AbortController();
  assert.deepEqual(await client.getStatus("api", { signal: controller.signal }), { workload: "api", mode: "online", state: "ready" });
  assert.equal(seen, controller.signal);
});

test("normalizes a successful run result from the public MCP contract", async () => {
  let received: { name: string; arguments: Readonly<Record<string, unknown>> } | undefined;
  const client = new CqClient({
    async listTools() { return { pages: [] }; },
    async callTool(call) {
      received = call;
      return { structuredContent: { id: "run-1", status: "completed", exit_code: 0 } };
    }
  });

  const result = await client.runWorkload("api");
  assert.deepEqual(result, { id: "run-1", status: "completed", exitCode: 0 });
  assert.equal(Object.isFrozen(result), true);
  assert.deepEqual(received, { name: "run_workload", arguments: { workload: "api" } });
});

test("does not expose a malformed or plaintext error response", async () => {
  const client = new CqClient({ async listTools() { return { pages: [] }; }, async callTool() { return { isError: true, content: [{ type: "text", text: "secret" }] }; } });
  await assert.rejects(client.listWorkloads(), error => error instanceof CqError && error.category === "denied" && !error.message.includes("secret"));
});

test("requires an explicit disclosure client and redacts inspection surfaces", async () => {
  const client = new CqClient({ async listTools() { return { pages: [] }; }, async callTool() { return { structuredContent: { alias: "TOKEN", value: "synthetic-value", version: "v1" } }; } });
  const value = await client.disclosure().readSecret("api", "TOKEN");
  assert.equal(value.reveal(), "synthetic-value");
  assert.equal(value.toString().includes("synthetic-value"), false);
  assert.equal(JSON.stringify(value).includes("synthetic-value"), false);
});

test("official adapter rejects a second discovery page", async () => {
  const adapter = officialMCPAdapter({
    async listTools() { return { tools: [{ name: "list_workloads" }], nextCursor: "next" }; },
    async callTool() { return {}; }
  }, {});
  const client = new CqClient(adapter);
  await assert.rejects(client.discoverTools(), error => error instanceof CqError && error.category === "protocol");
});

test("strict discovery accepts only one bounded known-tool catalog", async () => {
  const client = new CqClient({
    async listTools() {
      return { pages: [{ tools: [{ name: "list_workloads" }, { name: "get_status" }] }] };
    },
    async callTool() { return {}; }
  });
  assert.deepEqual(await client.discoverTools(), [{ name: "list_workloads" }, { name: "get_status" }]);
});

test("strict discovery rejects multiple adapter pages", async () => {
  const client = new CqClient({
    async listTools() {
      return { pages: [{ tools: [] }, { tools: [] }] };
    },
    async callTool() { return {}; }
  });
  await assert.rejects(client.discoverTools(), error => error instanceof CqError && error.category === "protocol");
});

test("local client refuses an untrusted relative executable before it can start", async () => {
  await assert.rejects(LocalCLIClient.create({ executablePath: "cq", expectedSHA256: "0".repeat(64), configPath: "/policy.json", stateDirectory: "/state" }), error => error instanceof CqError && error.category === "invalid");
});

test("local update checks remain explicit API surface", () => {
  assert.equal(typeof LocalCLIClient.prototype.updateCheck, "function");
});

test("runtime version DTO categories are stable", async () => {
  const fixture = await readFixture<Array<{ value: unknown; category: "protocol" | "success" | "unsupported" }>>("runtime-version.json");
  for (const { value, category } of fixture) {
    const client = { json: async () => value } as unknown as LocalCLIClient;
    const invoke = () => LocalCLIClient.prototype.version.call(client);
    if (category === "success") {
      await assert.doesNotReject(invoke());
    } else {
      await assert.rejects(invoke(), error => error instanceof CqError && error.category === category);
    }
  }
});

test("remote sources reject literal query and fragment delimiters", async () => {
  const fixture = await readFixture<Array<{ source: string; valid: boolean }>>("remote-source-delimiters.json");
  const directory = await mkdtemp(join(tmpdir(), "cq-sdk-bootstrap-"));
  try {
    for (const { source, valid } of fixture) {
      const client = { json: async (args: string[]) => args } as unknown as LocalCLIClient;
      const input = { configSource: source, configRef: "a".repeat(40), configPath: "policy.json", configSHA256: "b".repeat(64), output: join(directory, "policy.json") };
      if (valid) {
        await assert.doesNotReject(LocalCLIClient.prototype.bootstrap.call(client, input));
      } else {
        await assert.rejects(
          async () => LocalCLIClient.prototype.bootstrap.call(client, input),
          error => error instanceof CqError && error.category === "invalid"
        );
      }
    }
  } finally {
    await rm(directory, { force: true, recursive: true });
  }
});

test("provider versions remain opaque UTF-8 metadata", async () => {
  const fixture = await readFixture<Array<{ version: string }>>("provider-interop.json");
  for (const { version } of fixture) {
    const disclosure = await secretClient({ alias: "TOKEN", value: "synthetic", version }).disclosure().readSecret("api", "TOKEN");
    assert.equal(disclosure.version, version);
  }
});

test("secret values use UTF-8 byte limits and classify expiry", async () => {
  const boundary = "😀".repeat(16_384);
  await secretClient({ alias: "TOKEN", value: boundary }).disclosure().readSecret("api", "TOKEN");
  await assert.rejects(secretClient({ alias: "TOKEN", value: `${boundary}😀` }).disclosure().readSecret("api", "TOKEN"), error => error instanceof CqError && error.category === "protocol");
  await assert.rejects(secretClient({ alias: "TOKEN", value: "synthetic", expires_at: "2000-01-01T00:00:00Z" }).disclosure().readSecret("api", "TOKEN"), error => error instanceof CqError && error.category === "expired");
  await assert.rejects(secretClient({ alias: "TOKEN", value: "synthetic", expires_at: "not-a-timestamp" }).disclosure().readSecret("api", "TOKEN"), error => error instanceof CqError && error.category === "protocol");
  await assert.rejects(secretClient({ alias: "TOKEN", value: "synthetic", expires_at: "2030-01-01" }).disclosure().readSecret("api", "TOKEN"), error => error instanceof CqError && error.category === "protocol");
});

function secretClient(structuredContent: Record<string, unknown>): CqClient {
  return new CqClient({
    async listTools() { return { pages: [] }; },
    async callTool() { return { structuredContent }; }
  });
}

async function readFixture<T>(name: string): Promise<T> {
  const path = fileURLToPath(new URL(`./fixtures/${name}`, import.meta.url));
  return JSON.parse(await readFile(path, "utf8")) as T;
}
