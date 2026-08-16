# Agent Doctor Public MVP Repository and Launch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish Agent Doctor as a verifiable public beta by NexoToken with a complete GitHub repository, safe feedback loop, bilingual onboarding, and three detailed launch articles for Zhihu, NodeLoc, and NodeSeek.

**Architecture:** Keep Agent Doctor as an independent local-first repository and use NexoToken's official Agent Doctor page as the brand and product bridge. Strengthen the existing privacy boundary with regression tests, package the current tested binary through the existing GitHub Actions/GoReleaser pipeline, and keep every public claim traceable to repository evidence. Remote repository creation and release happen only after GitHub authentication and CI succeed.

**Tech Stack:** Go 1.25, React 19, TypeScript, Vite, Vitest, pnpm 10, SQLite, GitHub Actions, GoReleaser, GitHub CLI, Markdown, SVG.

---

## File map

### Product and safety

- Modify `internal/storage/handoff_test.go` — prove sanitized Snapshot fields and honest empty handoff behavior.
- Modify `internal/server/server_test.go` — prove the authenticated preview API never returns credential-shaped handoff content.
- Modify `dashboard/tests/pages.test.tsx` — prove the Memory page renders only the safe API projection and preserves the empty state.

### Repository presentation

- Modify `README.md` — global landing page, language switch, public-beta status, product proof, NexoToken relationship, install and feedback CTAs.
- Create `README.zh-CN.md` — complete Chinese landing page rather than a shortened translation.
- Create `assets/agent-doctor-flow.svg` — repository-native product flow graphic.
- Create `assets/agent-doctor-social-card.svg` — original social preview source artwork.
- Create `CHANGELOG.md` — public beta release history.
- Create `docs/roadmap.md` — evidence-based near-term roadmap.

### Feedback workflow

- Create `.github/ISSUE_TEMPLATE/bug_report.yml` — safe reproducible bug intake.
- Create `.github/ISSUE_TEMPLATE/feature_request.yml` — user-scenario-first feature intake.
- Create `.github/ISSUE_TEMPLATE/client_support.yml` — client capability feedback.
- Create `.github/ISSUE_TEMPLATE/config.yml` — security and discussion routing.
- Create `.github/PULL_REQUEST_TEMPLATE.md` — privacy, tests and compatibility review.

### Launch and promotion

- Create `docs/launch/public-beta.md` — release positioning and verified boundaries.
- Create `docs/launch/feedback-guide.md` — safe user feedback instructions.
- Create `docs/launch/release-checklist.md` — reproducible local/CI/release gate.
- Create `docs/marketing/zhihu-agent-doctor-public-beta.md` — detailed story-led Zhihu article.
- Create `docs/marketing/nodeloc-agent-doctor-public-beta.md` — detailed technical NodeLoc launch post.
- Create `docs/marketing/nodeseek-agent-doctor-public-beta.md` — concise but complete NodeSeek community post.
- Modify `scripts/check-docs.sh` — require launch assets, bilingual links and platform-specific article contracts.
- Modify `scripts/check-secrets.sh` — include new public Markdown and SVG assets in the existing credential scan.

### Version and publication

- Modify `install.sh` — default to the public beta release version.
- Modify `install.ps1` — use the same public beta version and `setup --all` behavior.
- Modify `docs/install.md` — document beta install, local build, upgrade and rollback.
- Modify `docs/test-reports/v1.0.0.md` — do not rewrite historical evidence; add a link from the new beta report instead.
- Create `docs/test-reports/v0.1.0-beta.1.md` — fresh release evidence.

---

### Task 1: Lock the handoff privacy and empty-state boundary

**Files:**
- Modify: `internal/storage/handoff_test.go`
- Modify: `internal/server/server_test.go`
- Modify: `dashboard/tests/pages.test.tsx`

- [ ] **Step 1: Add a storage regression test for sanitized Snapshot fields**

Add a test that saves a task and active memory containing credential-shaped text, calls `ProjectHandoff`, serializes the returned capsule, and verifies both `Rendered` and structured Snapshot fields are filtered:

```go
func TestProjectHandoffSanitizesStructuredPreviewAndRenderedText(t *testing.T) {
    // Create one captured request and one active memory containing
    // "Authorization: Bearer ..." and "sk-..." fixtures.
    capsule, err := database.ProjectHandoff(ctx, []string{"project-safe"}, 800, now)
    if err != nil { t.Fatal(err) }
    encoded, err := json.Marshal(capsule)
    if err != nil { t.Fatal(err) }
    for _, forbidden := range []string{"Bearer private-token", "sk-private-example"} {
        if strings.Contains(string(encoded), forbidden) {
            t.Fatalf("handoff leaked %q: %s", forbidden, encoded)
        }
    }
    for _, expected := range []string{"[REDACTED]", "local-sqlite-cross-client-handoff"} {
        if !strings.Contains(string(encoded), expected) {
            t.Fatalf("handoff missing %q: %s", expected, encoded)
        }
    }
}
```

- [ ] **Step 2: Run the storage tests**

Run:

```bash
go test ./internal/storage -run 'TestProjectHandoff(SanitizesStructuredPreviewAndRenderedText|RejectsProjectWithoutTaskOrConfirmedMemory)' -count=1
```

Expected: both tests pass. If the new test fails, change `internal/handoff/render.go` so `sanitizeSnapshot` remains the only constructor for the exported Capsule Snapshot; do not add a second UI-only sanitizer.

- [ ] **Step 3: Add an authenticated API regression test**

Extend `internal/server/server_test.go` with a capsule fixture containing a credential-shaped value and assert the API body contains only its filtered replacement. Also retain the existing 404 assertion when the store reports no handoff.

```go
for _, forbidden := range []string{"Bearer private-token", "sk-private-example"} {
    if strings.Contains(response.Body.String(), forbidden) {
        t.Fatalf("handoff API leaked %q: %s", forbidden, response.Body.String())
    }
}
```

- [ ] **Step 4: Add a dashboard empty-state regression**

In `dashboard/tests/pages.test.tsx`, mock `loadProjectHandoff` as rejected and navigate to 项目记忆. Assert `当前还没有可交接的任务` is visible and no source/target arrow is rendered.

- [ ] **Step 5: Run the focused safety tests**

Run:

```bash
go test ./internal/storage ./internal/server -count=1
pnpm --dir dashboard test -- --run dashboard/tests/pages.test.tsx
```

Expected: all focused tests pass and no raw credential-shaped fixture appears in test output.

- [ ] **Step 6: Commit the safety boundary**

```bash
git add internal/storage/handoff_test.go internal/server/server_test.go dashboard/tests/pages.test.tsx
git commit -m "test: lock public handoff privacy boundary"
```

---

### Task 2: Build the bilingual repository landing experience

**Files:**
- Modify: `README.md`
- Create: `README.zh-CN.md`
- Create: `assets/agent-doctor-flow.svg`
- Create: `assets/agent-doctor-social-card.svg`

- [ ] **Step 1: Add documentation contract assertions before editing README**

Extend `scripts/check-docs.sh` to require both README files and these exact relationships:

```sh
grep -q 'README.zh-CN.md' README.md
grep -q 'README.md' README.zh-CN.md
grep -q 'Agent Doctor by NexoToken' README.md
grep -q 'https://www.nexotoken.net/official/tools/agent-doctor' README.md
grep -q 'https://www.nexotoken.net/official/tools/agent-doctor' README.zh-CN.md
grep -q 'assets/agent-doctor-flow.svg' README.md
grep -q 'assets/agent-doctor-flow.svg' README.zh-CN.md
```

- [ ] **Step 2: Run the docs contract and confirm RED**

Run `./scripts/check-docs.sh`.

Expected: FAIL because `README.zh-CN.md` and the two SVG assets do not exist yet.

- [ ] **Step 3: Rewrite the English README first screen**

The first screen must contain, in order:

1. `Agent Doctor by NexoToken`;
2. language switch `[简体中文](README.zh-CN.md)`;
3. status badge text `Public beta` without fabricated download/Star counts;
4. one-sentence outcome;
5. flow graphic;
6. two CTAs: install and report feedback;
7. a visible link to `https://www.nexotoken.net/official/tools/agent-doctor`.

Keep the existing architecture, commands, limitations, privacy and compatibility sections. Remove the statement that remote installers only exist “after v1.0.0”; replace it with a beta release verification gate.

- [ ] **Step 4: Create the complete Chinese README**

`README.zh-CN.md` must independently explain:

- what problem the product solves;
- who should and should not use it;
- how Codex MCP guidance differs from Claude Code hooks;
- the one-command local build path and verified Release path;
- dashboard sections;
- exact/estimated/unavailable cost semantics;
- local SQLite and credential boundary;
- known limitations;
- NexoToken relationship and optional usage import;
- links to Issues, privacy, compatibility and troubleshooting.

It must not be a short pointer to the English README.

- [ ] **Step 5: Create original SVG repository artwork**

Create `assets/agent-doctor-flow.svg` as a code-native diagram with four nodes:

```text
Codex / Claude Code → local evidence → Agent Doctor guidance → local dashboard / next AI
```

Create `assets/agent-doctor-social-card.svg` at 1280×640 with the title `Agent Doctor by NexoToken`, subtitle `Local evidence for reliable AI coding`, and the same evergreen/signal-green palette as the dashboard. Use only text, geometric shapes and repository-owned branding; do not imitate third-party artwork.

- [ ] **Step 6: Run documentation and secret checks**

```bash
./scripts/check-docs.sh
./scripts/check-secrets.sh
```

Expected: PASS.

- [ ] **Step 7: Commit the landing experience**

```bash
git add README.md README.zh-CN.md assets/agent-doctor-flow.svg assets/agent-doctor-social-card.svg scripts/check-docs.sh
git commit -m "docs: add bilingual public beta landing"
```

---

### Task 3: Add a safe GitHub feedback workflow

**Files:**
- Create: `.github/ISSUE_TEMPLATE/bug_report.yml`
- Create: `.github/ISSUE_TEMPLATE/feature_request.yml`
- Create: `.github/ISSUE_TEMPLATE/client_support.yml`
- Create: `.github/ISSUE_TEMPLATE/config.yml`
- Create: `.github/PULL_REQUEST_TEMPLATE.md`
- Modify: `CONTRIBUTING.md`

- [ ] **Step 1: Add a repository-structure contract test**

Extend `scripts/check-docs.sh`:

```sh
for file in \
  .github/ISSUE_TEMPLATE/bug_report.yml \
  .github/ISSUE_TEMPLATE/feature_request.yml \
  .github/ISSUE_TEMPLATE/client_support.yml \
  .github/ISSUE_TEMPLATE/config.yml \
  .github/PULL_REQUEST_TEMPLATE.md; do
  test -s "$file" || { echo "missing feedback artifact: $file" >&2; exit 1; }
done
```

- [ ] **Step 2: Run the check and confirm RED**

Run `./scripts/check-docs.sh`.

Expected: FAIL on the first missing Issue template.

- [ ] **Step 3: Create three focused Issue forms**

All forms must include this warning:

```yaml
- type: markdown
  attributes:
    value: >-
      Do not paste API keys, Authorization headers, cookies, private source code,
      complete conversations, or the Agent Doctor SQLite database.
```

Required bug fields: operating system, architecture, Agent Doctor version, client and version, install method, exact reproduction steps, expected result, actual result, sanitized diagnostic output.

Required feature fields: user scenario, current workaround, desired outcome, supported evidence source, privacy impact.

Required client-support fields: client/version, documented extension interface, signals available, signals missing, minimal public reproduction.

- [ ] **Step 4: Create Issue configuration and PR checklist**

Disable blank issues. Route security reports to `SECURITY.md`; route questions to GitHub Discussions after the repository enables Discussions. PR checklist must include tests, docs, privacy, exact/estimated/unavailable evidence, and supported-client boundary.

- [ ] **Step 5: Update CONTRIBUTING**

Replace the sentence that detailed commands “will be added” with the actual verification commands:

```bash
go test ./... -race -count=1
pnpm --dir dashboard test -- --run
pnpm --dir dashboard build
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
```

- [ ] **Step 6: Validate YAML and docs**

Use Ruby's standard YAML parser without adding dependencies:

```bash
ruby -e 'require "yaml"; Dir[".github/ISSUE_TEMPLATE/*.yml"].each { |f| YAML.load_file(f); puts f }'
./scripts/check-docs.sh
```

Expected: each YAML path prints once and docs check passes.

- [ ] **Step 7: Commit the feedback workflow**

```bash
git add .github/ISSUE_TEMPLATE .github/PULL_REQUEST_TEMPLATE.md CONTRIBUTING.md scripts/check-docs.sh
git commit -m "docs: add safe public feedback workflow"
```

---

### Task 4: Add beta roadmap, changelog and release documentation

**Files:**
- Create: `CHANGELOG.md`
- Create: `docs/roadmap.md`
- Create: `docs/launch/public-beta.md`
- Create: `docs/launch/feedback-guide.md`
- Create: `docs/launch/release-checklist.md`
- Modify: `docs/install.md`
- Modify: `scripts/check-docs.sh`

- [ ] **Step 1: Add launch-document requirements**

Require all six files in `scripts/check-docs.sh` and assert that `CHANGELOG.md` contains `[0.1.0-beta.1]`, while the beta document contains both `NexoToken` and `public beta`.

- [ ] **Step 2: Run docs check and confirm RED**

Run `./scripts/check-docs.sh`.

Expected: FAIL on `CHANGELOG.md`.

- [ ] **Step 3: Write CHANGELOG and bounded roadmap**

`CHANGELOG.md` records only user-visible beta capabilities already verified. `docs/roadmap.md` contains three horizons:

- now: installer reliability, Codex/Claude guidance, feedback fixes;
- next: more honest adapters and better cost evidence;
- later: optional team workflows only after local privacy remains intact.

Do not include dates or promises that cannot be supported.

- [ ] **Step 4: Write launch and feedback documents**

`public-beta.md` explains audience, proof, limitations, version and NexoToken relationship. `feedback-guide.md` tells users how to obtain sanitized version/doctor output and explicitly rejects full DB/log uploads. `release-checklist.md` mirrors the exact local, CI, tag, artifact download and checksum gates.

- [ ] **Step 5: Update install documentation**

Add separate sections for:

- local source install;
- verified `v0.1.0-beta.1` Release install;
- `AGENT_DOCTOR_VERSION` override;
- rollback to a previous verified release;
- complete uninstall versus local-data deletion.

- [ ] **Step 6: Validate and commit**

```bash
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
git add CHANGELOG.md docs/roadmap.md docs/launch docs/install.md scripts/check-docs.sh
git commit -m "docs: add public beta release package"
```

Expected: all checks pass before commit.

---

### Task 5: Write the detailed Zhihu launch article

**Files:**
- Create: `docs/marketing/zhihu-agent-doctor-public-beta.md`
- Modify: `scripts/check-docs.sh`

- [ ] **Step 1: Add a Zhihu editorial contract**

Require the file and assert it contains these headings/phrases:

```sh
grep -q '^# ' docs/marketing/zhihu-agent-doctor-public-beta.md
grep -q '为什么长任务越来越难复盘' docs/marketing/zhihu-agent-doctor-public-beta.md
grep -q '它实际记录什么' docs/marketing/zhihu-agent-doctor-public-beta.md
grep -q '它不会做什么' docs/marketing/zhihu-agent-doctor-public-beta.md
grep -q '如何开始使用' docs/marketing/zhihu-agent-doctor-public-beta.md
grep -q 'https://github.com/18534516725/Agent-Doctor' docs/marketing/zhihu-agent-doctor-public-beta.md
```

- [ ] **Step 2: Run docs check and confirm RED**

Run `./scripts/check-docs.sh`.

Expected: FAIL because the article is missing.

- [ ] **Step 3: Write a genuinely detailed Zhihu article**

Target 2,000–3,200 Chinese characters with this structure:

1. title focused on a real problem rather than “震撼发布”;
2. story: a long AI coding task becomes slow, repeats work and loses handoff context;
3. why chat history alone does not answer request health, cost precision and validation state;
4. how Agent Doctor captures local evidence and produces bounded guidance;
5. dashboard capabilities with exact/estimated/unavailable examples;
6. Codex MCP versus Claude Code Hook boundary;
7. local SQLite and credential boundary;
8. cross-client handoff example;
9. honest limitations and public-beta status;
10. installation steps;
11. GitHub feedback CTA;
12. relationship to NexoToken without turning the article into an API-price advertisement.

Use these links:

```markdown
[Agent Doctor GitHub](https://github.com/18534516725/Agent-Doctor)
[Agent Doctor by NexoToken](https://www.nexotoken.net/official/tools/agent-doctor?utm_source=zhihu&utm_medium=community&utm_campaign=agent_doctor_beta)
```

Do not claim user counts, saved money, ranking, “zero configuration for every client”, or model superiority.

- [ ] **Step 4: Run content safety checks**

```bash
./scripts/check-docs.sh
./scripts/check-secrets.sh
rg -n '最强|全自动支持所有|节省[0-9]+%|用户超过|排名第一' docs/marketing/zhihu-agent-doctor-public-beta.md
```

Expected: docs and secret checks pass; the prohibited-claim scan returns no matches.

- [ ] **Step 5: Commit the Zhihu article**

```bash
git add docs/marketing/zhihu-agent-doctor-public-beta.md scripts/check-docs.sh
git commit -m "docs: add detailed Zhihu beta launch article"
```

---

### Task 6: Write the detailed NodeLoc technical launch post

**Files:**
- Create: `docs/marketing/nodeloc-agent-doctor-public-beta.md`
- Modify: `scripts/check-docs.sh`

- [ ] **Step 1: Add a NodeLoc editorial contract**

Require headings for architecture, supported clients, install, privacy, known limitations and feedback. Require the GitHub and NodeLoc-specific NexoToken links.

- [ ] **Step 2: Run docs check and confirm RED**

Run `./scripts/check-docs.sh`.

Expected: FAIL on the missing NodeLoc article.

- [ ] **Step 3: Write the technical post**

Target 1,500–2,400 Chinese characters. Lead with the open-source motivation and architecture, then explain:

- Hook/MCP/Skill capability differences;
- local loopback service and SQLite;
- deterministic guidance rather than a hidden second model;
- supported client matrix versus actually connected clients;
- install commands;
- exact/estimated/unavailable cost model;
- current beta limitations;
- issue feedback requirements;
- NexoToken relationship and optional personal usage import.

Use:

```markdown
[项目仓库](https://github.com/18534516725/Agent-Doctor)
[NexoToken 产品说明](https://www.nexotoken.net/official/tools/agent-doctor?utm_source=nodeloc&utm_medium=community&utm_campaign=agent_doctor_beta)
```

- [ ] **Step 4: Validate and commit**

```bash
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
git add docs/marketing/nodeloc-agent-doctor-public-beta.md scripts/check-docs.sh
git commit -m "docs: add NodeLoc technical launch post"
```

---

### Task 7: Write the NodeSeek community launch post

**Files:**
- Create: `docs/marketing/nodeseek-agent-doctor-public-beta.md`
- Modify: `scripts/check-docs.sh`

- [ ] **Step 1: Add a NodeSeek editorial contract**

Require an opening one-sentence use case, a three-item capability list, installation, limitations, feedback, GitHub link and NodeSeek-specific NexoToken link.

- [ ] **Step 2: Run docs check and confirm RED**

Run `./scripts/check-docs.sh`.

Expected: FAIL on the missing NodeSeek article.

- [ ] **Step 3: Write the community post**

Target 900–1,500 Chinese characters. Keep paragraphs short and include:

- what pain it solves;
- three capabilities users can verify immediately;
- one-command source install and post-Release install distinction;
- privacy boundary;
- supported-client boundary;
- known beta gaps;
- request for reproducible feedback;
- clear `by NexoToken` relationship without coupons, prices or forced registration.

Use:

```markdown
[GitHub：Agent Doctor](https://github.com/18534516725/Agent-Doctor)
[产品页：Agent Doctor by NexoToken](https://www.nexotoken.net/official/tools/agent-doctor?utm_source=nodeseek&utm_medium=community&utm_campaign=agent_doctor_beta)
```

- [ ] **Step 4: Validate and commit**

```bash
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
git add docs/marketing/nodeseek-agent-doctor-public-beta.md scripts/check-docs.sh
git commit -m "docs: add NodeSeek community launch post"
```

---

### Task 8: Align beta installers and create fresh release evidence

**Files:**
- Modify: `install.sh`
- Modify: `install.ps1`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Create: `docs/test-reports/v0.1.0-beta.1.md`
- Modify: `scripts/check-docs.sh`

- [ ] **Step 1: Add an installer-version contract test**

Extend `scripts/check-docs.sh`:

```sh
for file in install.sh install.ps1 README.md README.zh-CN.md docs/install.md; do
  grep -q '0.1.0-beta.1' "$file" || { echo "beta version mismatch: $file" >&2; exit 1; }
done
grep -q 'setup --all --yes --json' install.sh
grep -q 'setup --all --yes --json' install.ps1
```

- [ ] **Step 2: Run docs check and confirm RED**

Run `./scripts/check-docs.sh`.

Expected: FAIL because installers still default to `1.0.0` and Windows setup is not aligned.

- [ ] **Step 3: Align installer defaults**

Set the shell and PowerShell default version to `0.1.0-beta.1`. Change Windows to `setup --all --yes --json` so Codex and Claude Code owned assets match the documented one-command path. Keep checksum verification and do not introduce unattended client restarts.

- [ ] **Step 4: Run the complete release verification**

```bash
go test ./... -race -count=1
pnpm --dir dashboard test -- --run
pnpm --dir dashboard build
./scripts/embed-dashboard.sh
go build -trimpath -o ./bin/agent-doctor ./cmd/agent-doctor
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
```

Expected: all commands pass.

- [ ] **Step 5: Write the fresh beta test report**

Record exact commands, pass counts, dashboard bundle names, supported build matrix, limitations, local runtime smoke result and the fact that the public GitHub Release is still pending until its workflow completes. Do not copy historical `v1.0.0` counts if the fresh run differs.

- [ ] **Step 6: Commit the release candidate**

```bash
git add install.sh install.ps1 README.md README.zh-CN.md docs/install.md docs/test-reports/v0.1.0-beta.1.md scripts/check-docs.sh internal/server/web
git commit -m "build: prepare Agent Doctor public beta"
```

---

### Task 9: Create and configure the public GitHub repository

**Files:**
- Git remote metadata only; no source file is created in this task.

- [ ] **Step 1: Verify the local release candidate is clean**

```bash
git status --short
git log -1 --oneline --name-only
git remote -v
```

Expected: only the pre-existing untracked `.superpowers/` directory may remain; all release files are committed; no Agent Doctor remote exists yet.

- [ ] **Step 2: Restore GitHub CLI authentication without storing a token in Git**

Because the inherited `GH_TOKEN` is invalid, use an authenticated device/browser flow outside source control:

```bash
env -u GH_TOKEN gh auth login --hostname github.com --git-protocol ssh --web
env -u GH_TOKEN gh auth status
```

Expected: active account `18534516725`. If authentication is unavailable, stop this task; do not create a substitute repository under another account.

- [ ] **Step 3: Create the public repository and push main**

```bash
env -u GH_TOKEN gh repo create 18534516725/Agent-Doctor \
  --public \
  --source=. \
  --remote=origin \
  --description='Local-first diagnostics, context handoff, and cost evidence for AI coding agents — by NexoToken.' \
  --push
```

Expected: `origin` points to `git@github.com:18534516725/Agent-Doctor.git`, and anonymous repository access returns HTTP 200.

- [ ] **Step 4: Configure public metadata**

```bash
env -u GH_TOKEN gh repo edit 18534516725/Agent-Doctor \
  --homepage='https://www.nexotoken.net/official/tools/agent-doctor' \
  --enable-issues \
  --enable-discussions
env -u GH_TOKEN gh api -X PUT repos/18534516725/Agent-Doctor/topics \
  -H 'Accept: application/vnd.github+json' \
  -f names[]='ai-coding' -f names[]='codex' -f names[]='claude-code' \
  -f names[]='developer-tools' -f names[]='local-first' -f names[]='observability' \
  -f names[]='mcp' -f names[]='sqlite'
```

Expected: About metadata, Issues, Discussions and eight topics are visible.

- [ ] **Step 5: Wait for CI rather than assuming success**

```bash
env -u GH_TOKEN gh run list --repo 18534516725/Agent-Doctor --branch main --limit 5
env -u GH_TOKEN gh run watch --repo 18534516725/Agent-Doctor --exit-status
```

Expected: matrix CI and release snapshot jobs pass. If a job fails, fix locally with TDD, commit and push; never edit generated source directly on GitHub.

---

### Task 10: Publish and independently verify v0.1.0-beta.1

**Files:**
- Modify: `docs/test-reports/v0.1.0-beta.1.md` only if published evidence adds new verifiable facts.

- [ ] **Step 1: Re-run pre-tag checks**

```bash
git status --short
go test ./... -race -count=1
pnpm --dir dashboard test -- --run
./scripts/check-docs.sh
./scripts/check-secrets.sh
git diff --check
```

Expected: all pass; `.superpowers/` remains untracked and is not included in the tag.

- [ ] **Step 2: Create and push the annotated pre-release tag**

```bash
git tag -a v0.1.0-beta.1 -m 'Agent Doctor public beta 1'
git push origin v0.1.0-beta.1
```

- [ ] **Step 3: Wait for the Release workflow**

```bash
env -u GH_TOKEN gh run watch --repo 18534516725/Agent-Doctor --exit-status
env -u GH_TOKEN gh release view v0.1.0-beta.1 --repo 18534516725/Agent-Doctor
```

Expected: archives for macOS/Linux/Windows amd64/arm64, `SHA256SUMS.txt`, SBOMs and `release-manifest.json` are present.

- [ ] **Step 4: Independently verify downloaded public artifacts**

```bash
release_dir=$(mktemp -d)
env -u GH_TOKEN gh release download v0.1.0-beta.1 \
  --repo 18534516725/Agent-Doctor \
  --dir "$release_dir"
./scripts/verify-release.sh "$release_dir"
```

Expected: verification passes for every published asset.

- [ ] **Step 5: Update the beta report with published evidence**

Add the tag, workflow run URL, artifact count and independent checksum result. Do not add download counts, user counts or ranking claims.

- [ ] **Step 6: Commit and push the final evidence update**

```bash
git add docs/test-reports/v0.1.0-beta.1.md
git commit -m "docs: record verified public beta release"
git push origin main
```

---

### Task 11: Verify the NexoToken ↔ Agent Doctor bridge and hand off the posts

**Files:**
- Read-only verification in `payment-platform` unless a test reveals a broken link.
- Final deliverables: the three files under `docs/marketing/`.

- [ ] **Step 1: Verify Agent Doctor links back to NexoToken**

```bash
rg -n 'https://www.nexotoken.net/official/tools/agent-doctor' \
  README.md README.zh-CN.md docs/marketing docs/integrations/nexotoken.md
```

Expected: README files, integration docs and all three posts contain the appropriate official-page link.

- [ ] **Step 2: Verify NexoToken links to the public repository**

From `/Users/wangqi/work/payment-platform` run:

```bash
rg -n 'https://github.com/18534516725/Agent-Doctor' \
  frontend/src/public-site/pages/AgentDoctorPage.tsx frontend/tests/publicSiteMetadata.test.ts
cd frontend && pnpm exec tsx --test tests/publicSiteMetadata.test.ts
```

Expected: product page and test both reference the public repository; metadata test passes. Do not deploy payment-platform unless this verification reveals and fixes an actual broken link.

- [ ] **Step 3: Check every public URL anonymously**

```bash
curl -fsSI https://github.com/18534516725/Agent-Doctor
curl -fsSI https://github.com/18534516725/Agent-Doctor/releases/tag/v0.1.0-beta.1
curl -fsSI https://www.nexotoken.net/official/tools/agent-doctor
```

Expected: HTTP 200 or a single canonical redirect followed by HTTP 200.

- [ ] **Step 4: Deliver the platform-specific Markdown files**

Provide clickable paths for:

- `docs/marketing/zhihu-agent-doctor-public-beta.md`
- `docs/marketing/nodeloc-agent-doctor-public-beta.md`
- `docs/marketing/nodeseek-agent-doctor-public-beta.md`

State that the user should publish only after the public repository and release URLs pass Step 3.

---

## Final verification matrix

Run from `/Users/wangqi/work/Agent-Doctor`:

```bash
go test ./... -race -count=1
pnpm --dir dashboard test -- --run
pnpm --dir dashboard build
./scripts/embed-dashboard.sh
go build -trimpath -o ./bin/agent-doctor ./cmd/agent-doctor
./scripts/check-docs.sh
./scripts/check-secrets.sh
ruby -e 'require "yaml"; Dir[".github/ISSUE_TEMPLATE/*.yml"].each { |f| YAML.load_file(f) }'
git diff --check
git status --short
```

Expected final state:

- all commands pass;
- no SQLite database, credentials, private logs or generated `.superpowers/` content is committed;
- `main` is pushed to `origin` without force;
- GitHub CI is green;
- public beta artifacts pass independent verification;
- NexoToken and Agent Doctor link to each other;
- all three promotional Markdown files are detailed, platform-specific and ready to publish.
