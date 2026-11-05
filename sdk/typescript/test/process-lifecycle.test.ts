import assert from "node:assert/strict";
import childProcess from "node:child_process";
import { createHash } from "node:crypto";
import fsPromises, { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
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
  readonly starts: Array<{ readonly executable: string; readonly args: readonly string[] }> = [];
  #plans: ChildPlan[] = [];
  #waiters: Array<(child: SyntheticChild) => void> = [];
  #throwOnStart = false;

  enqueue(plan: ChildPlan): void { this.#plans.push(plan); }
  failNextStart(): void { this.#throwOnStart = true; }

  spawn = (executable: string, args: readonly string[]): unknown => {
    if (this.#throwOnStart) {
      this.#throwOnStart = false;
      throw new Error("synthetic spawn failure");
    }
    const plan = this.#plans.shift();
    if (plan === undefined) throw new Error("synthetic child plan missing");
    const child = new SyntheticChild();
    this.starts.push({ executable, args });
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
      for (const response of ['{}', '{"error":1}', '{"error":"access_denied","detail":"synthetic"}', '{"error":"invented"}']) {
        await assertProcessCategory(() => client.providers(), harness, response, "protocol", sdk);
      }
    });

    await t.test("rejects nonportable parsed JSON from a successful child", async () => {
      const client = await createClient(sdk, harness);
      for (const response of ['{"number":1e400}', '{"number":9007199254740992}']) {
        harness.enqueue(child => queueMicrotask(() => finish(child, response, 0)));
        await assert.rejects(client.providers(), category(sdk, "protocol"));
      }
    });

    await t.test("preserves an explicit effectful nonzero exit result", async () => {
      const client = await createClient(sdk, harness);
      harness.enqueue(child => queueMicrotask(() => child.emit("close", 7)));
      assert.deepEqual(await client.run("api"), { exitCode: 7, stdout: "" });
      harness.enqueue(child => queueMicrotask(() => child.emit("close", null)));
      assert.deepEqual(await client.run("api"), { exitCode: 1, stdout: "" });
    });

    await t.test("uses public config-init and flow-run contracts", async () => {
      const client = await createClient(sdk, harness);
      harness.enqueue(child => queueMicrotask(() => finish(child, '{"created":true}', 0)));
      assert.deepEqual(await client.configInit("/synthetic/generated.json"), { created: true });
      assert.deepEqual(harness.starts.at(-1)?.args, ["--config", "/synthetic/config.json", "--state-dir", "/synthetic/state", "config", "init", "--output", "/synthetic/generated.json"]);

      harness.enqueue(child => queueMicrotask(() => finish(child, "", 0)));
      assert.deepEqual(await client.flowRun("api", completePin()), { exitCode: 0, stdout: "" });
      assert.deepEqual(harness.starts.at(-1)?.args, ["--config", "/synthetic/config.json", "--state-dir", "/synthetic/state", "flow", "run", "--workload", "api", "--config-source", "https://github.com/acme/policy", "--config-ref", "a".repeat(40), "--config-path", "policy.json", "--config-sha256", "b".repeat(64), "--non-interactive"]);
    });

    await t.test("rejects malformed remote URLs before a local process starts", async () => {
      const client = await createClient(sdk, harness);
      const before = harness.children.length;
      assert.throws(() => client.bootstrap({ ...completePin(), configSource: "https://[", output: "/synthetic/policy.json" }), category(sdk, "invalid"));
      assert.equal(harness.children.length, before);
    });

    await t.test("rejects unsafe explicit environment entries before a local process starts", async () => {
      const options = await trustedOptions();
      const before = harness.children.length;
      for (const environment of [
        { AWS_SECRET_ACCESS_KEY: "synthetic" },
        { "not-valid": "synthetic" },
        { SAFE: "synthetic\0value" },
        { SAFE: "x".repeat(32_769) }
      ]) {
        await assert.rejects(sdk.LocalCLIClient.create({ ...options, environment }), category(sdk, "invalid"));
      }
      assert.equal(harness.children.length, before);
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
      const directory = await mkdtemp(join(tmpdir(), "cq-sdk-missing-"));
      try {
        await assert.rejects(sdk.LocalCLIClient.create({ ...options, executablePath: join(directory, "missing") }), category(sdk, "integrity"));
        await assert.rejects(sdk.LocalCLIClient.create({ ...options, executablePath: directory }), category(sdk, "integrity"));
        const empty = join(directory, "empty");
        await writeFile(empty, "");
        await assert.rejects(sdk.LocalCLIClient.create({ ...options, executablePath: empty }), category(sdk, "integrity"));
      } finally {
        await rm(directory, { force: true, recursive: true });
      }
      assert.equal(harness.children.length, before);
    });

    await t.test("refuses symlinks and unsafe opened-file metadata before process start", async childTest => {
      const options = await trustedOptions();
      const before = harness.children.length;
      const originalLstat = fsPromises.lstat;
      const originalOpen = fsPromises.open;
      childTest.mock.method(fsPromises, "lstat", async (path: string) => {
        const stat = await originalLstat(path);
        childTest.mock.method(stat, "isSymbolicLink", () => true);
        return stat;
      });
      syncBuiltinESMExports();
      try {
        await assert.rejects(sdk.LocalCLIClient.create(options), category(sdk, "integrity"));
      } finally {
        childTest.mock.restoreAll();
        syncBuiltinESMExports();
      }
      for (const metadata of [{ isFile: false, size: 1 }, { isFile: true, size: 256 * 1024 * 1024 + 1 }]) {
        childTest.mock.method(fsPromises, "open", async (...args: Parameters<typeof originalOpen>) => {
          const handle = await originalOpen(...args);
          childTest.mock.method(handle, "stat", async () => ({ isFile: () => metadata.isFile, size: metadata.size }));
          return handle;
        });
        syncBuiltinESMExports();
        try {
          await assert.rejects(sdk.LocalCLIClient.create(options), category(sdk, "integrity"));
        } finally {
          childTest.mock.restoreAll();
          syncBuiltinESMExports();
        }
      }
      assert.equal(harness.children.length, before);
    });

    await t.test("refuses an executable truncated during digest verification", async childTest => {
      const options = await trustedOptions();
      const before = harness.children.length;
      const originalOpen = fsPromises.open;
      childTest.mock.method(fsPromises, "open", async (...args: Parameters<typeof originalOpen>) => {
        const handle = await originalOpen(...args);
        childTest.mock.method(handle, "read", async () => ({ bytesRead: 0 }));
        return handle;
      });
      syncBuiltinESMExports();
      try {
        await assert.rejects(sdk.LocalCLIClient.create(options), category(sdk, "integrity"));
        assert.equal(harness.children.length, before);
      } finally {
        childTest.mock.restoreAll();
        syncBuiltinESMExports();
      }
    });

    await t.test("cancellation during executable verification prevents process creation", async childTest => {
      const client = await createClient(sdk, harness);
      const controller = new AbortController();
      const before = harness.children.length;
      const originalOpen = fsPromises.open;
      childTest.mock.method(fsPromises, "open", async (...args: Parameters<typeof originalOpen>) => {
        const handle = await originalOpen(...args);
        const originalClose = handle.close.bind(handle);
        childTest.mock.method(handle, "close", async () => { await originalClose(); controller.abort(); });
        return handle;
      });
      syncBuiltinESMExports();
      try {
        await assert.rejects(client.plan({ signal: controller.signal }), category(sdk, "cancelled"));
        assert.equal(harness.children.length, before);
      } finally {
        childTest.mock.restoreAll();
        syncBuiltinESMExports();
      }
    });

    await t.test("repeated lifecycle failures preserve the first bounded cleanup", async childTest => {
      childTest.mock.timers.enable({ apis: ["setTimeout"] });
      const client = await createClient(sdk, harness);
      harness.enqueue(child => {
        child.onKill = () => { throw new Error("synthetic kill failure"); };
        queueMicrotask(() => {
          child.stdin.emit("error", new Error("synthetic stdin"));
          child.emit("error", new Error("synthetic child"));
        });
      });
      const expectedChildCount = harness.children.length + 1;
      const pending = client.plan();
      await harness.waitFor(expectedChildCount);
      childTest.mock.timers.tick(6000);
      await assert.rejects(pending, category(sdk, "process"));
      const child = harness.latest();
      assert.deepEqual(child.signals, ["SIGTERM", "SIGKILL"]);
      child.emit("close", 0);
      child.stdout.emit("data", "x".repeat(1024 * 1024 + 1));
      assert.deepEqual(child.signals, ["SIGTERM", "SIGKILL"]);
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
  return sdk.LocalCLIClient.create(await trustedOptions());
}

async function trustedOptions(): Promise<{ readonly executablePath: string; readonly expectedSHA256: string; readonly configPath: string; readonly stateDirectory: string }> {
  const executablePath = process.execPath;
  const expectedSHA256 = createHash("sha256").update(await readFile(executablePath)).digest("hex");
  return { executablePath, expectedSHA256, configPath: "/synthetic/config.json", stateDirectory: "/synthetic/state" };
}

function completePin(): { readonly configSource: string; readonly configRef: string; readonly configPath: string; readonly configSHA256: string } {
  return { configSource: "https://github.com/acme/policy", configRef: "a".repeat(40), configPath: "policy.json", configSHA256: "b".repeat(64) };
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
  return error => error instanceof sdk.CqError && error.category === expected;
}
