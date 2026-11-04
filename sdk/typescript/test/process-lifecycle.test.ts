import assert from "node:assert/strict";
import childProcess from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { EventEmitter } from "node:events";
import { syncBuiltinESMExports } from "node:module";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { test } from "node:test";

type SDK = typeof import("../src/index.ts");
type ChildPlan = (child: SyntheticChild) => void;

class SyntheticInput extends EventEmitter {
  received: string | undefined;

  end(input?: string): void {
    this.received = input;
    this.emit("end", input);
  }
}

class SyntheticChild extends EventEmitter {
  readonly stdin = new SyntheticInput();
  readonly stdout = new EventEmitter();
  readonly stderr = new EventEmitter();
  readonly signals: string[] = [];
  onKill: (() => void) | undefined;

  kill(signal?: string): boolean {
    this.signals.push(signal ?? "SIGTERM");
    this.onKill?.();
    return true;
  }
}

class ChildHarness {
  readonly children: SyntheticChild[] = [];
  #plans: ChildPlan[] = [];
  #waiters: Array<(child: SyntheticChild) => void> = [];
  #throwOnStart = false;

  enqueue(plan: ChildPlan): void { this.#plans.push(plan); }
  failNextStart(): void { this.#throwOnStart = true; }

  spawn = (): unknown => {
    if (this.#throwOnStart) {
      this.#throwOnStart = false;
      throw new Error("synthetic spawn failure");
    }
    const plan = this.#plans.shift();
    if (plan === undefined) throw new Error("synthetic child plan missing");
    const child = new SyntheticChild();
    this.children.push(child);
    plan(child);
    this.#waiters.shift()?.(child);
    return child;
  };

  latest(): SyntheticChild {
    const child = this.children.at(-1);
    if (child === undefined) throw new Error("synthetic child missing");
    return child;
  }

  waitFor(count: number): Promise<SyntheticChild> {
    const child = this.children[count - 1];
    return child === undefined ? new Promise(resolve => this.#waiters.push(resolve)) : Promise.resolve(child);
  }
}

test("local client safely maps synthetic child lifecycle outcomes", async t => {
  const harness = new ChildHarness();
  const originalSpawn = childProcess.spawn;
  childProcess.spawn = harness.spawn as typeof childProcess.spawn;
  syncBuiltinESMExports();
  const sdk = await import("../src/index.ts") as SDK;
  try {
    await t.test("completes a bounded JSON request and sends JSON only through stdin", async () => {
      const client = await createClient(sdk, harness);
      harness.enqueue(child => {
        child.stdin.once("end", () => queueMicrotask(() => finish(child, '{"accepted":true}', 0)));
      });
      assert.deepEqual(await client.configSet("/workloads/api", { enabled: true }), { accepted: true });
      assert.equal(harness.latest().stdin.received, '{"enabled":true}');
    });

    await t.test("maps bounded native failures without exposing diagnostics", async () => {
      const client = await createClient(sdk, harness);
      await assertProcessCategory(() => client.providers(), harness, '{"error":"access_denied"}', "denied", sdk);
      await assertProcessCategory(() => client.updateCheck(), harness, "malformed", "protocol", sdk);
      await assertProcessCategory(() => client.doctor(), harness, "", "process", sdk);
    });

    await t.test("preserves an explicit effectful nonzero exit result", async () => {
      const client = await createClient(sdk, harness);
      harness.enqueue(child => queueMicrotask(() => child.emit("close", 7)));
      assert.deepEqual(await client.run("api"), { exitCode: 7, stdout: "" });
    });

    await t.test("rejects process start and lifecycle stream failures with safe categories", async () => {
      const client = await createClient(sdk, harness);
      harness.failNextStart();
      await assert.rejects(client.plan(), category(sdk, "process"));
      await assertLifecycleFailure(() => client.configSet("/workloads/api", "value"), harness, child => child.stdin.emit("error", new Error("synthetic stdin")), sdk);
      await assertLifecycleFailure(() => client.plan(), harness, child => child.emit("error", new Error("synthetic child")), sdk);
    });

    await t.test("terminates bounded captured output and rejects before dispatching oversized input", async () => {
      const client = await createClient(sdk, harness);
      await assertLifecycleFailure(() => client.status("api"), harness, child => child.stdout.emit("data", "x".repeat(1024 * 1024 + 1)), sdk, "protocol");
      await assertLifecycleFailure(() => client.configValidate(), harness, child => child.stderr.emit("data", Buffer.alloc(1024 * 1024 + 1)), sdk, "protocol");
      const before = harness.children.length;
      assert.throws(() => client.configReplace("x".repeat(1024 * 1024 + 1)), category(sdk, "invalid"));
      assert.equal(harness.children.length, before);
    });

    await t.test("refuses missing or tampered executables before process start", async () => {
      const before = harness.children.length;
      const options = { expectedSHA256: "0".repeat(64), configPath: resolve("synthetic", "config.json"), stateDirectory: resolve("synthetic", "state") };
      await assert.rejects(sdk.LocalCLIClient.create({ ...options, executablePath: process.execPath }), category(sdk, "integrity"));
      const directory = await mkdtemp(join(tmpdir(), "cquay-sdk-missing-"));
      try {
        await assert.rejects(sdk.LocalCLIClient.create({ ...options, executablePath: join(directory, "missing") }), category(sdk, "integrity"));
      } finally {
        await rm(directory, { force: true, recursive: true });
      }
      assert.equal(harness.children.length, before);
    });

    await t.test("cancellation reaps an unresponsive child through the bounded fallback", async childTest => {
      childTest.mock.timers.enable({ apis: ["setTimeout"] });
      const client = await createClient(sdk, harness);
      const controller = new AbortController();
      harness.enqueue(child => queueMicrotask(() => controller.abort()));
      const expectedChildCount = harness.children.length + 1;
      const pending = client.plan({ signal: controller.signal });
      await harness.waitFor(expectedChildCount);
      childTest.mock.timers.tick(6000);
      await assert.rejects(pending, category(sdk, "cancelled"));
      assert.deepEqual(harness.latest().signals, ["SIGTERM", "SIGKILL"]);
    });
  } finally {
    childProcess.spawn = originalSpawn;
    syncBuiltinESMExports();
  }
});

async function createClient(sdk: SDK, harness: ChildHarness): Promise<InstanceType<SDK["LocalCLIClient"]>> {
  harness.enqueue(child => queueMicrotask(() => finish(child, '{"version":"1.0.0","commit":"synthetic","go":"go1.27.3","maturity":"development"}', 0)));
  const executable = process.execPath;
  const expectedSHA256 = createHash("sha256").update(await readFile(executable)).digest("hex");
  return sdk.LocalCLIClient.create({ executablePath: executable, expectedSHA256, configPath: "/synthetic/config.json", stateDirectory: "/synthetic/state" });
}

async function assertProcessCategory(operation: () => Promise<unknown>, harness: ChildHarness, stderr: string, expected: string, sdk: SDK): Promise<void> {
  harness.enqueue(child => queueMicrotask(() => finish(child, stderr, 1)));
  await assert.rejects(operation(), category(sdk, expected));
}

async function assertLifecycleFailure(operation: () => Promise<unknown>, harness: ChildHarness, emit: (child: SyntheticChild) => void, sdk: SDK, expected = "process"): Promise<void> {
  harness.enqueue(child => {
    child.onKill = () => queueMicrotask(() => child.emit("close", 0));
    queueMicrotask(() => emit(child));
  });
  await assert.rejects(operation(), category(sdk, expected));
}

function finish(child: SyntheticChild, stdoutOrStderr: string, code: number): void {
  if (code === 0) child.stdout.emit("data", stdoutOrStderr);
  else child.stderr.emit("data", stdoutOrStderr);
  child.emit("close", code);
}

function category(sdk: SDK, expected: string): (error: unknown) => boolean {
  return error => error instanceof sdk.CquayError && error.category === expected;
}
