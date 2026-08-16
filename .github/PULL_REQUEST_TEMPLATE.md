## Problem and evidence

Describe the user-visible problem and the evidence supporting this change.

## Verification

- [ ] A failing test existed before the implementation change.
- [ ] `go test ./... -race -count=1` passes when Go code changed.
- [ ] Dashboard tests and build pass when frontend code changed.
- [ ] `./scripts/check-docs.sh`, `./scripts/check-secrets.sh`, and `git diff --check` pass.

## Privacy and compatibility

- [ ] No credential, full conversation, private source, private path, database, or raw header is included.
- [ ] Missing evidence remains unavailable rather than inferred.
- [ ] Client capability claims match `docs/compatibility.md`.
- [ ] New persisted or exported fields have an explicit privacy review.
