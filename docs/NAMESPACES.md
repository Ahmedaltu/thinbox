# Linux Namespaces — How thinbox Uses Them

## What is a namespace?

A namespace is a kernel feature that wraps a global system resource and makes
a process see its own private version of it instead of the shared one.

Without namespaces, every process on a Linux machine lives in the same world:
- Same process list
- Same filesystem
- Same hostname
- Same network interfaces

With namespaces, a process can be given its own private version of any of these.
It sees only what is inside its namespace. The host and other containers are
invisible to it.

The key point: **the kernel is still shared**. Namespaces do not create a new
kernel or emulate hardware. They just change what each process can see. This is
why containers are much faster and lighter than virtual machines.

---

## The hotel analogy

Think of a Linux machine as a hotel:
- The building, electricity, and plumbing are the kernel
- Each room is a namespace
- Guests in room 101 cannot see into room 102
- But they all share the same building

The hotel does not build a new power grid for each room. It just puts walls
between them. That is what namespaces do.

---

## The four namespaces thinbox uses

### 1. PID namespace — `CLONE_NEWPID`

**What it does:**
Gives the container its own process ID space. The first process inside the
container becomes PID 1. It cannot see any processes running on the host.

**Without it:**
```
# inside container — sees all host processes
$ ps aux
root         1  systemd
root       312  sshd
root       891  thinbox
root       892  /bin/sh      ← our container process
```

**With CLONE_NEWPID:**
```
# inside container — only sees its own processes
$ ps aux
root         1  /bin/sh      ← container thinks it is PID 1
```

**Why PID 1 matters:**
In Linux, PID 1 is special — it is the init process, responsible for reaping
orphaned child processes. When a container process thinks it is PID 1, it
behaves correctly as a standalone system. This is exactly what Docker and
LXD do.

**In Go:**
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID,
}
```

---

### 2. Mount namespace — `CLONE_NEWNS`

**What it does:**
Gives the container its own view of the filesystem mount table. Mounts made
inside the container do not affect the host. Combined with pivot_root, this
is how the container gets its own root filesystem.

**Without it:**
Any filesystem mount inside the container would be visible on the host — a
serious security problem.

**With CLONE_NEWNS:**
The container can mount /proc, /dev, /sys inside itself without touching
the host's mount table at all.

**In Go:**
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWNS,
}
```

---

### 3. UTS namespace — `CLONE_NEWUTS`

**What it does:**
Gives the container its own hostname and domain name. UTS stands for
"UNIX Time-sharing System" — a legacy name from the original Unix struct
that stored hostname information.

**Without it:**
If you run `hostname mycontainer` inside the container, it changes the
host's hostname too — breaking every other process on the machine.

**With CLONE_NEWUTS:**
```
# on host
$ hostname
raspberrypi

# inside container
$ hostname
thinbox-1a2b3c4d
```

They are completely independent.

**In Go:**
```go
// inside the child process, after fork:
syscall.Sethostname([]byte("thinbox-" + id))
```

---

### 4. Network namespace — `CLONE_NEWNET`

**What it does:**
Gives the container its own network stack — its own interfaces, routing
table, iptables rules, and ports. The container starts with only a loopback
interface (lo).

**Without it:**
The container shares the host's network. A process inside could bind to
port 80 and intercept the host's traffic — a serious security problem.

**With CLONE_NEWNET:**
```
# on host
$ ip link
eth0, wlan0, lo ...

# inside container
$ ip link
lo    ← only loopback, completely isolated
```

**In Go:**
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWNET,
}
```

---

## How thinbox creates namespaces

thinbox uses the re-exec pattern — the same pattern used by runc and Docker.

**Why re-exec?**

Go's runtime starts multiple threads before `main()` runs. Linux namespaces
must be created in a single-threaded process. If you try to call `unshare()`
after the Go runtime has started its threads, it fails.

The solution: thinbox re-execs itself. The parent process forks a child using
`exec.Command("/proc/self/exe", "child", ...)` with namespace clone flags.
The child process starts fresh — single-threaded at the moment the kernel
creates the namespaces — then the Go runtime starts inside the new namespaces.

**The flow:**

```
parent process (thinbox run)
    │
    │  fork with CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWUTS | CLONE_NEWNET
    │
    └─► child process (thinbox child)      ← born inside new namespaces
            │
            ├── set hostname (UTS namespace)
            ├── pivot_root into Alpine rootfs (MNT namespace)
            ├── mount /proc /dev /sys
            └── exec /bin/sh               ← replaces this process
```

**In Go:**
```go
cmd := exec.Command("/proc/self/exe", "child", id, rootfs, "/bin/sh")
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNS  |
                syscall.CLONE_NEWUTS |
                syscall.CLONE_NEWNET,
}
cmd.Run()
```

`/proc/self/exe` is a Linux symlink that always points to the currently
running binary. So thinbox re-execs itself — the child is the same binary
but started with the `child` subcommand instead of `run`.

---

## What we are NOT doing (yet)

- **User namespace (`CLONE_NEWUSER`)** — maps container root to an unprivileged
  host user. Important for rootless containers. Not in thinbox v1.
- **IPC namespace (`CLONE_NEWIPC`)** — isolates System V IPC and POSIX message
  queues. Not needed for our use case.
- **Time namespace (`CLONE_NEWTIME`)** — isolates system clocks. Very new,
  kernel 5.6+. Not in thinbox v1.

---

## Connection to LXD

LXD uses exactly these same namespace flags at its core. Every LXD container
is a process born with `CLONE_NEWPID | CLONE_NEWNS | CLONE_NEWUTS | CLONE_NEWNET`
(and more). thinbox strips away everything else and shows just this foundation.

When a Canonical engineer asks "how does LXD isolate containers?" — this
document is the answer.

---

## Further reading

- `man 7 namespaces` — Linux man page for namespaces
- `man 2 clone` — the clone() syscall that creates namespaces
- `man 2 unshare` — unshare namespaces in an existing process
- [Linux kernel docs on namespaces](https://www.kernel.org/doc/html/latest/admin-guide/namespaces/index.html)
- [Liz Rice — Containers from Scratch](https://www.youtube.com/watch?v=8fi7uSYlOdc) — the best 30-minute talk on this topic
