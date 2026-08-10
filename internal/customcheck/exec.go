package customcheck

import "os/exec"

// execLookPath is a package-level indirection so tests can fake binary
// presence without touching the real filesystem.
var execLookPath = exec.LookPath
