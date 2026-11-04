import assert from "node:assert/strict";
import test from "node:test";
import { CquayError, LocalCLIClient } from "../src/index.ts";

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
  return error instanceof CquayError && error.category === "invalid";
}
