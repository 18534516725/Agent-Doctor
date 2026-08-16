# Public release checklist

- [x] `go test ./... -race -count=1`
- [x] `pnpm --dir dashboard test -- --run`
- [x] `pnpm --dir dashboard build`
- [x] `./scripts/check-docs.sh`
- [x] `./scripts/check-secrets.sh`
- [x] `git diff --check`
- [x] CI passes on Linux, macOS and Windows
- [x] release archives, SBOMs and `SHA256SUMS.txt` are present
- [x] published archives pass an independent checksum and manifest verification
- [x] GitHub release and installer URLs return successfully
- [x] known limitations appear in release notes

The NexoToken product page is maintained separately from this repository and is
checked as part of the platform release process.
