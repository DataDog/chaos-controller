# Sync and upgrade Go dependencies

Upgrade Go module dependencies and align shared dependencies with a reference repository.
Loop until **both** conditions are met: zero drift AND `go build ./...` passes.
Then regenerate mocks via Linux Docker to match CI.

## Usage

```
/sync-deps [owner/repo]
```

Example: `/sync-deps DataDog/some-reference-repo`

---

## Phase 1 — Upgrade

1. Run `go get -u ./...` to bump all direct deps to their latest minor/patch versions
2. Run `make godeps` (go mod tidy + vendor)
3. Run `go build ./...` — fix any trivial breaking API changes (renamed methods, dropped parameters); if too complex to auto-fix, stop and report

---

## Phase 2 — Alignment loop

Repeat until BOTH conditions pass:
- **Condition A**: `./scripts/sync-dependencies.sh -r <repo> --dry-run` outputs "All shared dependencies are already synchronized!" (use `--yes` to skip interactive prompts when applying changes)
- **Condition B**: `go build ./...` exits 0

### Each iteration

**Step 1 — Detect drift**

```bash
./scripts/sync-dependencies.sh -r <repo> --dry-run
```

Collect the list of drifting packages and their target versions.

**Step 2 — Trace culprits for each drifting dep**

For each drifting package `<pkg>` currently at `<higher-version>`:

```bash
go mod graph 2>/dev/null | grep " <pkg>@<higher-version>" | grep -v "^github.com/DataDog/chaos-controller "
```

- **If empty** → the anchor is a stale explicit entry in `go.mod` (written by a previous `go mod tidy`). Lower it directly.
- **If non-empty** → these packages are pulling `<pkg>` up. Trace them recursively with the same command until you reach entries that are only anchored by `go.mod` directly.

**Step 3 — Cascade trace**

Repeat Step 2 for each newly identified culprit until no further upstream anchors exist.

**Step 4 — Apply all changes at once**

Edit `go.mod` in a single pass: lower all drifting direct deps AND all culprit indirect entries to their target versions simultaneously. Applying all changes at once prevents tidy from re-selecting a higher version from a still-high anchor.

Use Python for reliable in-place edits on macOS (BSD sed doesn't support `-i` without a suffix):

```python
# Example pattern for bulk replacement
with open('go.mod', 'r') as f:
    content = f.read()
for old, new in replacements:
    content = content.replace(old, new)
with open('go.mod', 'w') as f:
    f.write(content)
```

**Step 5 — Reconcile**

```bash
make godeps
```

If `go mod tidy` bumps any dep back above the target version, those deps have new culprits — go to Step 2 for them.

**Step 6 — Build check**

```bash
go build ./...
```

If it fails:
- Fix trivial compilation errors caused by downgraded APIs
- If fixing requires downgrading an additional dep that causes new drift, add it to the set of packages to lower and continue the loop

**Step 7 — Check both conditions**

If either condition still fails, go back to Step 1.

---

## Phase 3 — Regenerate mocks and verify

Mock import ordering is non-deterministic across platforms: macOS mockery produces different output than Linux CI (`validate-codegen` will fail). Always regenerate via Linux Docker.

**Detect current Go version from go.mod:**

```bash
grep "^toolchain" go.mod | awk '{print $2}' | sed 's/go//'
# e.g. 1.26.2
```

**Regenerate mocks in Linux container:**

```bash
docker run --rm -v "$(pwd)":/workspace -w /workspace \
  golang:<go-version> \
  bash -c "go generate ./..."
```

**Apply license headers locally:**

```bash
make header-fix
```

**Final verification:**

```bash
go build ./...
make test
```

Report the outcome.

---

## Key principles

- **Simultaneous pinning**: always lower all anchors in a single `go.mod` edit before running `make godeps`, so tidy sees all constraints at once
- **Stale indirect entries**: `go mod tidy` writes explicit indirect entries; on the next upgrade cycle these become stale anchors. They are the most common culprit when a dep "jumps back" after being lowered
- **Non-shared deps**: if a dep not present in the reference repo keeps pulling a shared dep above the target version, lower it too — even though it won't appear in the drift report
- **go directive**: after `make godeps`, verify the `go` line hasn't been bumped past the project's declared minimum (currently `go 1.25.9` with `toolchain go1.26.2`)
