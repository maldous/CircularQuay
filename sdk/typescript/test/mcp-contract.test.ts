import assert from "node:assert/strict";
import { inspect } from "node:util";
import test from "node:test";
import { CqClient, CqError } from "../src/index.ts";

function category(expected: string): (error: unknown) => boolean {
  return error => error instanceof CqError && error.category === expected && error.message === `CQ SDK ${expected}`;
}
function client(value: unknown): CqClient {
  return new CqClient({ async listTools() { return value; }, async callTool() { return value; } });
}
function result(value: unknown): CqClient { return client({ structuredContent: value }); }

test("discovery rejects malformed catalogs and retains bounded descriptions", async () => {
  for (const value of [null, { pages: [{}] }, { pages: [{ tools: {} }] },
    ...[Array(5).fill({ name: "list_workloads" }), [{ name: "unknown" }],
      [{ name: "list_workloads" }, { name: "list_workloads" }],
      [{ name: "get_status", description: 1 }], [{ name: "get_status", description: "x".repeat(4097) }]
    ].map(tools => ({ pages: [{ tools }] }))]) {
    await assert.rejects(client(value).discoverTools(), category("protocol"));
  }
  const catalog = await client({ pages: [{ tools: [{ name: "get_status", description: "Status" }] }] }).discoverTools();
  assert.deepEqual(catalog, [{ name: "get_status", description: "Status" }]);
  assert.equal(Object.isFrozen(catalog), true);
});

test("MCP rejects absent, nonportable, deeply nested and oversized structured results", async () => {
  await assert.rejects(client({}).listWorkloads(), category("protocol"));
  let deep: unknown = null;
  for (let index = 0; index < 34; index++) deep = { nested: deep };
  for (const value of [deep, Array(16_385).fill(null), NaN, Infinity, Number.MAX_SAFE_INTEGER + 1,
    new Date(), undefined, null, true, 1.5, [], { payload: "x".repeat(1024 * 1024) }]) {
    await assert.rejects(result(value).listWorkloads(), category("protocol"));
  }
});

test("status and run results accept the complete supported enum contract", async () => {
  for (const mode of ["online", "offline", "hybrid"]) {
    assert.deepEqual(await result({ workload: "api", mode, state: "ready" }).getStatus("api"), { workload: "api", mode, state: "ready" });
  }
  for (const status of ["completed", "failed", "running", "cancelled", "unknown"]) {
    assert.deepEqual(await result({ id: "run", status, exit_code: 255 }).runWorkload("api"), { id: "run", status, exitCode: 255 });
  }
});

test("disclosure validates expiry and redacts Node inspection", async () => {
  const read = (expires_at: unknown) => result({ alias: "TOKEN", value: "synthetic", expires_at }).disclosure().readSecret("api", "TOKEN");
  const future = "9999-01-01T00:00:00Z";
  const disclosure = await read(future);
  assert.equal(disclosure.expiresAt, future);
  assert.equal(inspect(disclosure), "[CQ secret redacted]");
  for (const expiry of ["0001-01-01T00:00:00Z", "0001-01-01T00:00:00.000Z"]) assert.equal((await read(expiry)).expiresAt, undefined);
  await assert.rejects(read("9999-99-99T99:99:99Z"), category("protocol"));
});

test("transport cancellation and observer failures preserve safe error outcomes", async () => {
  const controller = new AbortController();
  for (const abortDuringCall of [false, true]) {
    const failing = new CqClient({ async listTools() { return {}; }, async callTool() {
      if (abortDuringCall) controller.abort();
      throw Object.assign(new Error("synthetic detail"), { name: "AbortError" });
    } });
    await assert.rejects(failing.listWorkloads({ signal: controller.signal }), category("cancelled"));
  }
  const observed: string[] = [];
  const denied = new CqClient({ async listTools() { return {}; }, async callTool() { return { isError: true }; } }, {
    observer(event) { observed.push(event.outcome); throw new Error("observer detail"); }
  });
  await assert.rejects(denied.listWorkloads(), category("denied"));
  assert.deepEqual(observed, ["denied"]);
  const successful = new CqClient({ async listTools() { return {}; }, async callTool() { return { structuredContent: { workloads: [] } }; } }, { observer() {} });
  assert.deepEqual(await successful.listWorkloads(), []);
});
