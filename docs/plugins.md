# Plugins

distill can hand off formats it doesn't natively support to **out-of-process
plugins**. A plugin is any executable that speaks a tiny stdin/stdout protocol —
so you can write one in shell, Python, Rust, or anything that can read stdin and
write stdout. There is no Go `plugin` dependency, no cgo, and no recompiling
distill.

Plugins are **off by default**. Because they run arbitrary executables with the
privileges of the distill process, you must opt in per invocation (`--use-plugins`)
or per environment (`DISTILL_USE_PLUGINS`).

## The protocol

Your executable must support two invocations.

### 1. Capability discovery

distill runs your command with `--distill-capabilities` as the **first
argument**. You print one line of JSON to stdout and exit 0:

```json
{"name":"rtf","version":"0.1.0","extensions":[".rtf"],"mimetypes":["application/rtf"]}
```

| Field | Required | Notes |
|-------|----------|-------|
| `name` | no | Defaults to the manifest `name`. |
| `version` | no | Shown by `--list-plugins`. |
| `extensions` | one of these two | Normalized to lowercase, dot-prefixed (`foo` → `.foo`). |
| `mimetypes` | one of these two | Matched exactly against the detected MIME type. |

A plugin that declares neither `extensions` nor `mimetypes` is rejected.
Discovery has a 5-second timeout.

### 2. Conversion

distill runs your command with the manifest `args` (no capability flag), pipes
the document bytes to **stdin**, and reads **Markdown from stdout**. A non-zero
exit code is treated as a conversion error and stderr is surfaced in the message.
Conversion has a 120-second timeout.

> A plugin that forks a long-running child is force-terminated shortly after its
> timeout — it cannot wedge distill open by holding the output pipe.

## Configuring plugins

distill reads manifests from two locations (global first, then workspace):

- **Global:** `~/.config/distill/plugins.json` (or your OS config dir)
- **Workspace:** `./.distill/plugins.json`

```jsonc
{
  "plugins": [
    { "name": "rtf", "command": "/usr/local/bin/distill-plugin-rtf" },
    { "name": "doc-legacy", "command": "python3", "args": ["/opt/distill/plugins/doc.py"] }
  ]
}
```

`command` is the executable; `args` (optional) are passed on **every**
invocation (discovery and conversion). A missing file is not an error; a
malformed one is.

## CLI

```bash
distill --list-plugins                 # discover + list configured plugins, then exit
distill --use-plugins report.rtf       # enable plugins for this conversion
DISTILL_USE_PLUGINS=1 distill x.rtf    # enable via environment
```

When enabled, plugins are registered **ahead of** the built-in converters, so a
plugin can override a format distill already handles. `batch`, `serve`, and `mcp`
honor `DISTILL_USE_PLUGINS`.

## Example: a 6-line shell plugin

This plugin claims `.up` files and upper-cases them:

```sh
#!/bin/sh
if [ "$1" = "--distill-capabilities" ]; then
  echo '{"name":"upcase","version":"0.1.0","extensions":[".up"]}'
  exit 0
fi
printf '# upcased\n\n'
tr '[:lower:]' '[:upper:]'
```

```bash
chmod +x up.sh
mkdir -p .distill
echo '{"plugins":[{"name":"upcase","command":"'"$PWD"'/up.sh"}]}' > .distill/plugins.json

printf 'hello world\n' > sample.up
distill --use-plugins sample.up
# # upcased
#
# HELLO WORLD
```

## Security

- Plugins run as **arbitrary executables** with your user's privileges. There is
  no sandbox. Only enable plugins you trust.
- Default **off**: distill never executes a configured plugin unless you pass
  `--use-plugins` or set `DISTILL_USE_PLUGINS`.
- Discovery and conversion are time-bounded; a hung or forking plugin is killed.
