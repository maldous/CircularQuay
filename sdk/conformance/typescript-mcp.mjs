import assert from "node:assert/strict";
import { createRequire } from "node:module";

const TIMEOUT_MS = 30_000;
const command = process.env.CQ_EXECUTABLE;
const config = process.env.CQ_CONFIG;
const stateDirectory = process.env.CQ_STATE_DIRECTORY;
if (!command || !config || !stateDirectory) throw new Error("missing synthetic conformance paths");

const consumerRequire = createRequire(new URL("../typescript/package.json", import.meta.url));
const { Client } = await import(consumerRequire.resolve("@modelcontextprotocol/sdk/client/index.js"));
const { StdioClientTransport } = await import(consumerRequire.resolve("@modelcontextprotocol/sdk/client/stdio.js"));
const { CallToolResultSchema } = await import(consumerRequire.resolve("@modelcontextprotocol/sdk/types.js"));
const { CqClient, CqError, officialMCPAdapter } = await import(new URL("../typescript/dist/index.js", import.meta.url));

const transport = new StdioClientTransport({
  command,
  args: ["--config", config, "--state-dir", stateDirectory, "mcp", "serve", "--transport", "stdio"],
  env: {}
});
const session = new Client({ name: "cq-sdk-conformance", version: "1.0.0" });
const signal = AbortSignal.timeout(TIMEOUT_MS);

try {
  await bounded("connect", () => session.connect(transport));
  const client = new CqClient(officialMCPAdapter(session, CallToolResultSchema));
  const tools = await bounded("discover", () => client.discoverTools({ signal }));
  assert.deepEqual(tools.map(tool => tool.name).sort(), ["get_status", "list_workloads", "read_secret"]);

  assert.deepEqual(await bounded("list", () => client.listWorkloads({ signal })), ["demo"]);
  const status = await bounded("status", () => client.getStatus("demo", { signal }));
  assert.equal(status.workload, "demo");

  const disclosure = await bounded("read", () => client.disclosure().readSecret("demo", "DEMO_API_KEY", { signal }));
  assert.equal(disclosure.toString().includes("synthetic-only-not-a-credential"), false);
  assert.equal(disclosure.reveal(), "synthetic-only-not-a-credential");

  await assert.rejects(
    bounded("denied-status", () => client.getStatus("not-granted", { signal })),
    error => error instanceof CqError && error.category === "denied"
  );
} finally {
  await bounded("close", () => session.close()).catch(() => undefined);
}

async function bounded(name, operation) {
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`conformance timeout: ${name}`)), TIMEOUT_MS);
  });
  try {
    return await Promise.race([operation(), timeout]);
  } finally {
    clearTimeout(timer);
  }
}
