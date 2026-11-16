import assert from "node:assert/strict";
import childProcess from "node:child_process";
import { createHash } from "node:crypto";
import { EventEmitter } from "node:events";
import { readFile, realpath } from "node:fs/promises";
import { syncBuiltinESMExports } from "node:module";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import type { Observation } from "../src/index.ts";

class ContractInput extends EventEmitter {
  end(): void { /* No input is sent by these synthetic plan requests. */ }
}

class ContractChild extends EventEmitter {
  readonly stdin = new ContractInput();
  readonly stdout = new EventEmitter();
  readonly stderr = new EventEmitter();
  readonly signals: string[] = [];

  kill(signal: string): boolean {
    this.signals.push(signal);
    return true;
  }
}

test("termination preserves failure precedence through fallback and late close", async t => {
  const originalSpawn = childProcess.spawn;
  const children: ContractChild[] = [];
  let plan: ((child: ContractChild) => void) | undefined;
  childProcess.spawn = (() => {
    const child = new ContractChild();
    children.push(child);
    const selected = plan;
    assert.notEqual(selected, undefined, "every synthetic process has an explicit plan");
    queueMicrotask(() => { selected?.(child); });
    return child;
  }) as unknown as typeof childProcess.spawn;
  syncBuiltinESMExports();
  const sdk = await import("../src/index.ts");
  try {
    for (const scenario of ["overflow then cancel", "process then overflow", "process only"] as const) {
      await t.test(scenario, async context => {
        context.mock.timers.enable({ apis: ["setTimeout"] });
        plan = child => {
          child.stdout.emit("data", '{"version":"1.0.0","commit":"synthetic","go":"go1.27.4","maturity":"development"}');
          child.emit("close", 0);
        };
        const executablePath = await realpath(process.execPath);
        const expectedSHA256 = createHash("sha256").update(await readFile(executablePath)).digest("hex");
        const root = tmpdir();
        const client = await sdk.LocalCLIClient.create({ executablePath, expectedSHA256, configPath: join(root, "synthetic-cq-policy.json"), stateDirectory: join(root, "synthetic-cq-state"), environment: {} });
        const controller = new AbortController();
        let started: (() => void) | undefined;
        const ready = new Promise<void>(resolve => { started = resolve; });
        plan = child => {
          if (scenario === "overflow then cancel") {
            child.stdout.emit("data", "x".repeat(1024 * 1024 + 1));
            controller.abort();
          } else {
            child.stdin.emit("error", new Error("synthetic-private-pipe-detail"));
            if (scenario === "process then overflow") child.stdout.emit("data", "x".repeat(1024 * 1024 + 1));
          }
          child.emit("error", new Error("synthetic-private-child-detail"));
          started?.();
        };
        const pending = client.plan({ signal: controller.signal });
        await ready;
        context.mock.timers.tick(6000);
        const expected = scenario === "overflow then cancel" ? "cancelled" : scenario === "process then overflow" ? "protocol" : "process";
        await assert.rejects(pending, error => error instanceof sdk.CqError && error.category === expected && !error.message.includes("synthetic-private"));
        const child = children.at(-1);
        assert.notEqual(child, undefined);
        assert.deepEqual(child?.signals, ["SIGTERM", "SIGKILL"]);
        child?.emit("close", null);
      });
    }
  } finally {
    childProcess.spawn = originalSpawn;
    syncBuiltinESMExports();
  }
});

test("observer rejection cannot replace a failed MCP operation or disclose its detail", async () => {
  const sdk = await import("../src/index.ts");
  const original = new sdk.CqError("denied");
  const observations: Observation[] = [];
  let calls = 0;
  const client = new sdk.CqClient({
    async listTools() { return { pages: [] }; },
    async callTool() { calls++; throw original; }
  }, {
    async observer(value) {
      observations.push(value);
      throw new Error("synthetic-private-observer-detail");
    }
  });
  await assert.rejects(client.listWorkloads(), error => error === original);
  assert.equal(calls, 1);
  assert.deepEqual(observations.map(value => ({ action: value.action, outcome: value.outcome })), [{ action: "list_workloads", outcome: "denied" }]);
  assert.equal(observations.every(value => Object.keys(value).sort().join(",") === "action,durationMs,outcome" && value.durationMs >= 0), true);
});
