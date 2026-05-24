# Linux Namespaces

## Overview

A namespace wraps a global kernel resource and gives a process its own private
instance of it. Processes in different namespaces see different values of the
same resource and cannot interfere with each other.

thinbox uses four namespaces. All four are created in a single `clone()` call
when the container process forks.

---

## Namespaces used

### PID — `CLONE_NEWPID`

Isolates the process ID number space.

- The first process inside the namespace becomes PID 1
- Cannot see processes outside the namespace
- Signals and `/proc` entries are scoped to the namespace

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID,
}
```

Verification inside container:
```sh
$ ps aux
PID   USER   COMMAND
1     root   /bin/sh    ← container sees itself as PID 1
```

---

### MNT — `CLONE_NEWNS`

Isolates the mount table.

- Mounts made inside the container do not propagate to the host
- Required for `pivot_root` to work correctly
- `/proc`, `/dev`, `/sys` are mounted fresh inside the container

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWNS,
}
```

Note: `CLONE_NEWNS` is the original namespace flag — it predates the naming
convention, hence `NS` instead of `MNT`.

---

### UTS — `CLONE_NEWUTS`

Isolates hostname and domain name.

- Container gets its own hostname independent of the host
- `sethostname()` inside the container does not affect the host
- UTS = UNIX Time-sharing System (legacy name from the kernel struct)

```go
// set hostname inside child process after fork
syscall.Sethostname([]byte("thinbox-" + id[:8]))
```

Verification:
```sh
# host
$ hostname
raspberrypi

# inside container
$ hostname
thinbox-1a2b3c4d
```

---

### NET — `CLONE_NEWNET`

Isolates the network stack.

- Container gets its own network interfaces, routing table, and iptables rules
- Starts with only loopback (`lo`) — no external connectivity by default
- Prevents container from binding to host ports or intercepting host traffic

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWNET,
}
```

Verification:
```sh
# inside container
$ ip link
1: lo: <LOOPBACK> mtu 65536    ← only loopback
```

---

## Creating all four at once

All four flags are OR'd together in a single `clone()` call:

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNS  |
                syscall.CLONE_NEWUTS |
                syscall.CLONE_NEWNET,
}
```

This is more efficient than creating them sequentially — one syscall, one fork,
all namespaces active from the first instruction of the child process.

---

## The re-exec pattern

**Problem:** Go's runtime starts multiple OS threads before `main()` runs.
`clone()` with namespace flags requires a single-threaded process at the
moment of the call. If called after Go's runtime initialises, it fails.

**Solution:** thinbox re-execs itself. The parent calls:

```go
cmd := exec.Command("/proc/self/exe", "child", id, rootfs, command)
```

`/proc/self/exe` is a Linux symlink to the currently running binary. The
child process starts fresh — Go runtime has not yet initialised threads —
and the kernel creates the namespaces at `clone()` time before any threads
exist.

**Flow:**

```
parent: thinbox run
    │
    │  clone() with CLONE_NEWPID|NEWNS|NEWUTS|NEWNET
    │  exec /proc/self/exe child <id> <rootfs> <cmd>
    │
    └─► child: thinbox child        ← born inside new namespaces
            │
            ├── sethostname()
            ├── pivot_root()
            ├── mount /proc /dev /sys
            └── exec <cmd>          ← replaces child process
```

The child subcommand is internal — not exposed in the help text, not meant
to be called directly by the user.

---

## Kernel interfaces

| Operation | Syscall | Go package |
|---|---|---|
| Create namespaces at fork | `clone(2)` | `syscall.SysProcAttr.Cloneflags` |
| Set hostname | `sethostname(2)` | `syscall.Sethostname()` |
| Unshare after fork | `unshare(2)` | `syscall.Unshare()` |
| Re-exec self | `/proc/self/exe` | `exec.Command()` |

---

## References

- `man 7 namespaces`
- `man 2 clone`
- `man 2 unshare`