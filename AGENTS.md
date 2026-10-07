# AGENTS.md — AI Agent Instructions for openshift/oadp-must-gather

## Project Overview
OADP Must-Gather is a diagnostic data collection tool for the OADP operator and Velero components running on OpenShift clusters. It extends the OpenShift must-gather framework to collect OADP-specific resources, logs, and configuration for troubleshooting. Includes a Markdown UI for summarized results and supports both Velero, Restic, and Kopia backup engines.

- **Primary Language**: Go (CLI tool) + Shell (collection scripts)
- **Module**: `github.com/openshift/oadp-must-gather`
- **Default Branch**: `oadp-dev`

## Build Instructions
```bash
# Build the Go CLI tool
go build -o oadp-must-gather ./cmd/...

# Build container images
# See Dockerfile and Dockerfile.oadp
```

## Test Instructions
```bash
# Run Go tests
go test ./...

# Run specific tests
go test ./pkg/... -run TestName
go test ./cmd/... -run TestName

# Vet code
go vet ./...
```

## Dependency Management
```bash
# Update dependencies
make update-deps

# Update version
make update-version

# Full release preparation
make prepare-release
```

## Code Conventions
- Go CLI tool in `cmd/` and `pkg/`
- Collection scripts organized by component
- Velero-specific collection in `velero/`
- Restic-specific collection in `restic/`
- Kopia-specific collection in `kopia/`
- Deprecated scripts in `deprecated/`

## Project Structure
```
cmd/           - CLI tool entry point and commands
pkg/           - Shared Go packages for data collection and analysis
velero/        - Velero-specific must-gather scripts and templates
restic/        - Restic-specific data collection
kopia/         - Kopia-specific data collection
deprecated/    - Legacy collection scripts (kept for backward compatibility)
hack/          - Development and build scripts
Dockerfile     - Standard container image
Dockerfile.oadp - OADP-specific container image
```

## CI/CD
- Prow CI for OpenShift (primary)
- Container builds via Dockerfiles
- Reproduce CI locally:
  ```bash
  go vet ./...
  go test ./...
  go build ./cmd/...
  ```

## Common Tasks

### Adding collection for a new OADP component
1. Create a new collection script or Go package
2. Add to the appropriate directory (velero/, kopia/, etc.)
3. Update the main collection entry point
4. Test data collection against a cluster

### Updating the Markdown summary UI
- Summary generation is in `pkg/`
- Follow existing patterns for resource rendering

### Working with Dockerfiles
- `Dockerfile` — Standard image
- `Dockerfile.oadp` — OADP-specific image with additional tools
