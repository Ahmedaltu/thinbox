# thinbox — Environment Setup & Verification Notes

Before writing any code, we verified that the development environment meets
the requirements for building a Linux container runtime. Here is what we
checked and why each check matters.

---

## 1. Linux kernel version

**Command:**
```bash
uname -r
```

**Our result:**
```
6.6.87.2-microsoft-standard-WSL2
```

**Why we checked this:**

thinbox uses Linux namespaces and cgroup v2 — both are features of the Linux
kernel itself, not userspace libraries. They are not available on Windows or
macOS natively, which is why we use WSL2 (Windows Subsystem for Linux 2).

The minimum kernel version required is 5.x because:
- Namespace support (`CLONE_NEWPID`, `CLONE_NEWNS`, `CLONE_NEWUTS`, `CLONE_NEWNET`)
  has been stable since kernel 3.x, but the full feature set we need is solid at 5.x
- cgroup v2 (the unified hierarchy) was introduced in kernel 4.5 but became the
  default in most distributions at kernel 5.x+

Our kernel is 6.6 — well above the minimum. All features are available.

---

## 2. cgroup v2

**Command:**
```bash
cat /sys/fs/cgroup/cgroup.controllers
```

**Our result:**
```
cpuset cpu io memory hugetlb pids rdma
```

**Why we checked this:**

cgroup v2 (also called the "unified hierarchy") is how thinbox enforces resource
limits on containers — capping CPU usage, memory, and the number of processes a
container can spawn.

There are two versions of cgroups in Linux:
- cgroup v1 — older, each resource controller had its own hierarchy
- cgroup v2 — newer, single unified hierarchy, simpler API

thinbox uses cgroup v2 because it is the modern standard and what LXD, containerd,
and systemd all use on current Linux systems.

The file `/sys/fs/cgroup/cgroup.controllers` only exists if cgroup v2 is active.
The controllers we care about are:

| Controller | What thinbox uses it for |
|---|---|
| `memory` | Sets `memory.max` — max RAM the container can use |
| `cpu` | Sets `cpu.max` — max CPU fraction the container can use |
| `pids` | Sets `pids.max` — max number of processes inside the container |

All three are present in our output. cgroup v2 is active and ready.

---

## 3. Go version

**Command:**
```bash
go version
```

**Our result:**
```
go version go1.26.3 linux/amd64
```

**Why we checked this:**

thinbox is written in Go. Go 1.22+ is required because:
- We use `os.WriteFile` and other standard library functions that require modern Go
- The `syscall` package behaviour for namespace flags is stable and well-tested at 1.22+
- Go modules (`go.mod`) syntax we use requires 1.16+ minimum

Go was not pre-installed in WSL — we installed it via:
```bash
sudo snap install go --classic
```

Our version is 1.26.3 — well above the minimum.

---

## 4. Working directory

**Location:**
```
/mnt/c/Users/tuwai/Documents/GitHub/thinbox
```

The thinbox repo lives on the Windows filesystem, accessed through WSL2 via
`/mnt/c/`. This means:
- Code is edited in VS Code on Windows
- Code is built and run inside WSL2 (Linux)
- Git operations can be done from either side

This is a standard WSL2 development workflow.

---

## 5. Go module

**Command:**
```bash
go mod init github.com/Ahmedaltu/thinbox
```

**Result:**
```
module github.com/Ahmedaltu/thinbox
go 1.26.3
```

The Go module file (`go.mod`) tells the Go toolchain:
- What this module is called (`github.com/Ahmedaltu/thinbox`)
- What Go version it targets
- What external dependencies it has (none yet)

This allows internal packages to import each other using full paths, for example:
```go
import "github.com/Ahmedaltu/thinbox/internal/container"
```

---

## Summary

| Check | Required | Our result | Status |
|---|---|---|---|
| Linux kernel version | 5.x+ | 6.6.87.2 | ✓ |
| cgroup v2 active | memory, cpu, pids | all present | ✓ |
| Go version | 1.22+ | 1.26.3 | ✓ |
| Go module initialized | github.com/Ahmedaltu/thinbox | confirmed | ✓ |

Environment is fully ready. Next step: write `cmd/thinbox/main.go`.
