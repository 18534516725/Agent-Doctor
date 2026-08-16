# Public release checklist

- [ ] `go test ./... -race -count=1`
- [ ] `pnpm --dir dashboard test -- --run`
- [ ] `pnpm --dir dashboard build`
- [ ] `./scripts/check-docs.sh`
- [ ] `./scripts/check-secrets.sh`
- [ ] `git diff --check`
- [ ] CI passes on Linux, macOS and Windows
- [ ] release archives, SBOMs and `SHA256SUMS.txt` are present
- [ ] a clean machine verifies one archive and first start
- [ ] GitHub, NexoToken product page and installer URLs return successfully
- [ ] known limitations appear in release notes
