# Release Process

EyvesCloud installation and upgrades depend on GitHub Release artifacts. Use semantic version tags such as `v1.1.29` when releasing.

## Version Numbers

The version must be kept in sync in:

- `backend/internal/version/version.go`
- `frontend/package.json`
- The Release tag.

## Release Artifacts

The install script downloads the Linux AMD64 or ARM64 artifact according to the host architecture:

```text
clicd-linux-amd64.tar.gz
clicd-linux-arm64.tar.gz
```

In some scenarios it also tries to download the standalone binary:

```text
clicd-linux-amd64
clicd-linux-arm64
```

## Install Script Behavior

- `CLICD_VERSION=latest`: uses GitHub `releases/latest`.
- `CLICD_VERSION=vX.Y.Z`: downloads the Release artifact for the specified tag.

Example:

```bash
CLICD_VERSION=v1.1.29 sh install.sh
```

## Post-release Verification

- The install script can download the new version.
- `systemctl status clicd` is healthy.
- `/api/version` returns the new version.
- The web panel can load frontend assets.
- The container list, task queue, and API key pages open correctly.
