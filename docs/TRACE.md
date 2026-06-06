# Future: trace mode (design note)

> Status: design only. Not implemented in v0.3. No tracing code ships yet.

Snagify's diagnostics are non-invasive: it captures passive state and runs
read-only probes. Some "works on my machine" failures only surface at runtime —
a missing shared library, an unexpected config path, a certificate the app
can't find. A future opt-in `trace` mode would observe a single command run and
report the files, libraries, and endpoints it tried to access but couldn't.

## Proposed command

```sh
snagify trace -- <command> [args...]
```

Run the given command under OS-specific tracing, then summarize missing or
surprising accesses (files, libs, certs, config paths, network endpoints).

## Platform approach (sketch)

- **Linux** — `strace -f -e trace=file,network` (or eBPF where available).
  Parse failed `open`/`stat`/`connect` calls.
- **macOS** — `fs_usage`/`dtruss` are constrained by System Integrity
  Protection and require elevated privileges; coverage will be partial and may
  require the user to disable SIP for the traced process. Document the
  limitations honestly rather than overpromising.
- **Windows** — ETW or a ProcMon export (`.PML`/CSV) parsed after the fact;
  live ETW capture needs additional privileges.

## Hard constraints (carried over from v0.3)

- Opt-in only; never traces without an explicit `trace` invocation.
- Show exactly what will be run before running it.
- Never capture or transmit secret values or `.env` contents.
- No uploading of traces anywhere.
- Keep the honest promise: report *likely* missing dependencies, never claim a
  proven universal root cause.

## Why it is deferred

Tracing is invasive, platform-fragmented, and privilege-sensitive. It belongs in
its own release with a dedicated security review, not bolted onto the passive
diagnostics that make up v0.1–v0.3.
