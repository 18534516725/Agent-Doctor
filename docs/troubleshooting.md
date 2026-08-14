# Troubleshooting

## The dashboard URL stops working

The listener uses a random port and stops with the process. Run:

```bash
agent-doctor start
```

Open the newly printed URL. Do not bookmark the old session URL.

## Codex is found but the connection count stays at zero

`Found` only means the client configuration exists. After Agent Doctor writes or
updates its managed MCP block, fully quit and reopen Codex once. A live
`agent-doctor mcp serve` process then records a local heartbeat and the dashboard
changes Codex to `connected` within the 2.5-second polling interval. Agent Doctor
never force-quits or restarts an editor.

MCP connectivity proves that Codex can call Agent Doctor's read-only evidence
tools. It does not grant Agent Doctor access to the private Codex Desktop
transcript. Complete message capture still requires a supported local proxy or
an explicit client event interface.

## `doctor --json` reports read-only recovery

A migration failed and Agent Doctor protected the original SQLite file. Keep
the backup path private, stop writes, and update to a fixed version. Do not
delete the database to hide a migration defect.

## A cost is unavailable

Check whether the client reported token usage and whether a matching signed
catalog exists. Agent Doctor intentionally refuses to guess a model price or
currency rate.

## Setup does not modify another client

Current automatic setup owns only the Codex MCP block. Use the matching tested
asset under `adapters/<client>/`, review it, then install it through that
client's documented extension or MCP interface.

## Replay is refused

Generate a fresh preview. Consent is tied to the exact plan hash; changing the
model, command, base SHA, call limit, or cost limit invalidates prior consent.

## Remove everything

Run `agent-doctor uninstall --yes --json` to remove the owned integration, then
`agent-doctor forget --yes --json` to delete local history. Remove the binary
last. Neither command closes or restarts AI clients.
