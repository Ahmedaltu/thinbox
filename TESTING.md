# thinbox — Robot Framework Tests & CI

System-level tests for the [thinbox](https://github.com/Ahmedaltu/thinbox) container runtime, written in Robot Framework and integrated into GitHub Actions CI/CD.

## Structure

```
thinbox/
├── tests/
│   └── container_lifecycle.robot   # All test cases
├── resources/
│   └── thinbox_keywords.resource   # Reusable keywords
├── testdata/
│   └── rootfs/                     # Alpine minirootfs (auto-downloaded in CI)
└── .github/
    └── workflows/
        └── ci.yml                  # Build → rootfs → RF tests pipeline
```

## Test Coverage

| Tag         | What it covers                                      |
|-------------|-----------------------------------------------------|
| `smoke`     | Binary exists, basic run, missing args              |
| `lifecycle` | Exit code propagation, clean container exit         |
| `isolation` | PID / UTS / NET namespace isolation                 |
| `filesystem`| Fresh `/proc` mount, pivot_root rootfs isolation    |
| `negative`  | Invalid commands, missing arguments                 |

## Running Locally

```bash
# 1. Build thinbox
go build -o thinbox .

# 2. Download Alpine rootfs
mkdir -p testdata/rootfs
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz
sudo tar -xzf alpine-minirootfs-3.19.1-x86_64.tar.gz -C testdata/rootfs

# 3. Install Robot Framework
pip install robotframework

# 4. Run tests (root required for namespace syscalls)
sudo python -m robot --outputdir results tests/
```

Open `results/report.html` in your browser for the full test report.

## CI Pipeline

On every push/PR to `main`:

1. **Build** — `go build` compiles the binary
2. **Rootfs** — downloads Alpine minirootfs in parallel
3. **RF Tests** — Robot Framework runs all suites; results uploaded as artifacts
4. **Annotations** — Failed tests are annotated directly on the PR via `dorny/test-reporter`
