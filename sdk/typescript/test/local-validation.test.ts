import assert from "node:assert/strict";
import test from "node:test";
import { CqError, LocalCLIClient } from "../src/index.ts";

test("config replacement rejects nonportable JSON before local dispatch", () => {
  const { client, inputs } = recordingClient();
  const cyclicArray: unknown[] = [];
  cyclicArray.push(cyclicArray);
  const cyclicObject: { self?: unknown } = {};
  cyclicObject.self = cyclicObject;
  const shared = { enabled: true };

  for (const value of [
    cyclicArray,
    cyclicObject,
    { first: shared, second: shared },
    nestedArray(34),
    nestedObject(34),
    Array.from({ length: 16_384 }, () => null),
    Number.NaN,
    Number.MAX_SAFE_INTEGER + 1,
    new Date(),
    undefined,
    { "\0": "disallowed" }
  ]) {
    assert.throws(() => LocalCLIClient.prototype.configReplace.call(client, value), invalid);
  }
  assert.deepEqual(inputs, []);
});

test("config replacement accepts portable JSON at documented boundaries", async () => {
  const { client, inputs } = recordingClient();
  await LocalCLIClient.prototype.configReplace.call(client, { "": null, nested: [{ enabled: false }] });
  await LocalCLIClient.prototype.configReplace.call(client, Array.from({ length: 16_383 }, () => null));

  assert.equal(inputs[0], '{"":null,"nested":[{"enabled":false}]}');
  assert.equal(inputs[1]?.length, 81_916);
});

function recordingClient(): { readonly client: LocalCLIClient; readonly inputs: string[] } {
  const inputs: string[] = [];
  const client = {
    json: async (_args: string[], options: { readonly stdin?: string }) => {
      inputs.push(options.stdin ?? "");
      return {};
    }
  } as unknown as LocalCLIClient;
  return { client, inputs };
}

function nestedArray(depth: number): unknown[] {
  const root: unknown[] = [];
  let current = root;
  for (let index = 0; index < depth; index += 1) {
    const next: unknown[] = [];
    current.push(next);
    current = next;
  }
  return root;
}

function nestedObject(depth: number): Record<string, unknown> {
  const root: Record<string, unknown> = {};
  let current = root;
  for (let index = 0; index < depth; index += 1) {
    const next: Record<string, unknown> = {};
    current.next = next;
    current = next;
  }
  return root;
}

function invalid(error: unknown): boolean {
  return error instanceof CqError && error.category === "invalid";
}

test("local optional flags and preconditions retain their public command contract", async () => {
  const calls: string[][] = [];
  const client = { json: async (args: string[]) => { calls.push(args); return {}; } } as unknown as LocalCLIClient;
  await LocalCLIClient.prototype.configValidate.call(client);
  await LocalCLIClient.prototype.configInit.call(client);
  await LocalCLIClient.prototype.configInit.call(client, "/synthetic/policy.json");
  await LocalCLIClient.prototype.plan.call(client);
  assert.deepEqual(calls, [["config", "validate"], ["config", "init"], ["config", "init", "--output", "/synthetic/policy.json"], ["plan"]]);
  assert.throws(() => LocalCLIClient.prototype.configInit.call(client, "relative.json"), invalid);
  assert.throws(() => LocalCLIClient.prototype.configDelete.call(client, "", "invalid-digest"), invalid);
  assert.throws(() => LocalCLIClient.prototype.configInit.call(client, "/" + "x".repeat(4096)), invalid);
  assert.equal(calls.length, 4);
});

test("remote pins reject missing digest and revision before local dispatch", () => {
  const { client, inputs } = recordingClient();
  const pin = { configSource: "https://github.com/acme/policy", configPath: "policy.json", configRef: "a".repeat(40), configSHA256: "b".repeat(64), output: "/synthetic/policy.json" };
  // Deliberately malformed inputs exercise the boundary seen by untyped JavaScript callers.
  assert.throws(() => LocalCLIClient.prototype.bootstrap.call(client, { ...pin, configRef: undefined } as unknown as Parameters<LocalCLIClient["bootstrap"]>[0]), invalid);
  assert.throws(() => LocalCLIClient.prototype.bootstrap.call(client, { ...pin, configSHA256: undefined } as unknown as Parameters<LocalCLIClient["bootstrap"]>[0]), invalid);
  assert.throws(() => LocalCLIClient.prototype.bootstrap.call(client, { ...pin, configSource: undefined } as unknown as Parameters<LocalCLIClient["bootstrap"]>[0]), invalid);
  assert.deepEqual(inputs, []);
});

test("runtime version parsing rejects oversized and incomplete semantic versions", async () => {
  for (const version of ["v1.0.0+" + "a".repeat(256), "1.0.0+one+two", "1.0.0+", "1.0.0-"]) {
    const client = { json: async () => ({ version, commit: "synthetic", go: "synthetic", maturity: "development" }) } as unknown as LocalCLIClient;
    await assert.rejects(LocalCLIClient.prototype.version.call(client), error => error instanceof CqError && error.category === "protocol");
  }
});

test("remote operations reject absent pins before local dispatch", () => {
  const calls: string[][] = [];
  const client = { json: async (args: string[]) => { calls.push(args); return {}; }, invoke: async (args: string[]) => { calls.push(args); return {}; } } as unknown as LocalCLIClient;
  assert.throws(() => LocalCLIClient.prototype.flowRun.call(client, "api", {} as Parameters<LocalCLIClient["flowRun"]>[1]), invalid);
  assert.throws(() => LocalCLIClient.prototype.bootstrap.call(client, { output: "/synthetic/policy.json" }), invalid);
  assert.deepEqual(calls, []);
});
