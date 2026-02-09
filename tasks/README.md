# Tasks Directory

## Go Tools

### header-checker/

Go CLI tool that validates and inserts Apache 2.0 license headers on all source files.

**Build**: `make build-header-checker`
**Usage**: `./bin/header-checker`
**Exit codes**:

- 0 = success (headers valid or successfully fixed)
- 1 = error occurred

**Supported file types**: `.go`, `.yaml`, `.yml`, `.py`

**Features**:

- Handles auto-generated files with special header offsets
- Skips specified files (generated code, CRDs, protobuf files)
- Uses exact Datadog-style Apache 2.0 header format

### license-checker/

Go CLI tool that generates `LICENSE-3rdparty.csv` from vendor/ directory.

**Build**: `make build-license-checker`

**Usage**:

- Interactive mode: `./bin/license-checker`
- Non-interactive mode (CI): `./bin/license-checker --no-prompt`

**Exit codes**:

- 0 = success
- 1 = error occurred

**Dependencies**: Uses `github.com/google/licenseclassifier/v2` for SPDX license detection

**Features**:

- Parses `vendor/modules.txt` to extract Go module hierarchy
- Finds and analyzes LICENSE files for each vendor module
- Automatic SPDX license detection with 80% confidence threshold
- Interactive prompts for unrecognized licenses (with common license menu)
- Manual license entry option
- Intelligent caching (preserves manually entered licenses)
- Non-interactive mode for CI/CD pipelines (`--no-prompt`)
- Generates CSV with columns: From, Package, License
- Warns when license types change between runs

## Scripts

### release.sh

Bash script that creates git tags for releases.

**Usage**: `VERSION=1.0.0 make release`
