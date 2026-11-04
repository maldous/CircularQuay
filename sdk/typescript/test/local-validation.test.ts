import assert from "node:assert/strict";
import test from "node:test";
import { CquayError, LocalCLIClient } from "../src/index.ts";

test("config replacement rejects nonportable JSON before local dispatch", () => {
  let dispatches = 0;
  const client = {
    json: async () => { dispatches += 1; return {}; }
  } as unknown as LocalCLIClient;
  const cyclic: unknown[] = [];
  cyclic.push(cyclic);

  assert.throws(() => LocalCLIClient.prototype.configReplace.call(client, cyclic), invalid);
  assert.throws(() => LocalCLIClient.prototype.configReplace.call(client, new Date()), invalid);
  assert.equal(dispatches, 0);
});

function invalid(error: unknown): boolean {
  return error instanceof CquayError && error.category === "invalid";
}
