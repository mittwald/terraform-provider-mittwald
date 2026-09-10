---
name: audit-api-drift
description: Audit each Terraform resource/data source against the mittwald API operations it calls, find request/response fields not reflected in the Terraform schema, and either fix straightforward drift with a PR or open an issue for judgment calls.
---

# Audit mittwald API drift

This skill compares what the mittwald API (via `github.com/mittwald/api-client-go`)
actually accepts and returns against what each Terraform resource/data source in
this provider exposes in its schema. Its job is to catch **drift**: fields the API
gained that the schema never picked up, or fields the schema still carries that the
API dropped.

It is meant to run unattended (e.g. from the `audit-api-drift` GitHub Action,
monthly). Do not stop to ask clarifying questions — use the judgment rules
below, and when a call is genuinely ambiguous, leave it alone and write it up
instead of guessing.

## 1. Inventory what to check

Enumerate every resource and data source package:

```bash
ls internal/provider/resource/      # *resource packages
ls internal/provider/datasource/    # *datasource packages
```

For each one, find:

- Its Terraform schema (usually defined inline in `resource.go` / `data_source.go`'s
  `Schema()` method — there usually is no separate `schema.go`).
- Its model struct(s) (`model.go`), and the mapping helpers `model_api_to.go`
  (Terraform → API request) and `model_api_from.go` (API response → Terraform).
- The mittwald API client calls it makes. Grep for the client accessor pattern,
  e.g.:
  ```bash
  grep -rn "r\.client\.\|apiext\.New" internal/provider/resource/<pkg>/*.go
  ```
  This turns up calls like `r.client.Project().CreateProject(...)`,
  `r.client.Project().UpdateProject(...)`, `apiext.NewProjectClient(r.client)`, etc.
  Each call site names the request type (argument) and response type (return value)
  to inspect.

Remember the pairing rule from this repo's `CLAUDE.md`: a resource and a
same-named data source (e.g. `mittwald_project`) should mirror each other and
reuse the same model-mapping helpers. Treat them as one unit for this audit — a
field found missing on one almost always needs to be checked (and likely added)
on the other too.

## 2. Inspect the actual API schemas

Do **not** grep or `sed` through the module cache. Use `go doc`, per this repo's
`CLAUDE.md`:

```bash
MODULE_DIR=$(go list -m -f '{{.Dir}}' github.com/mittwald/api-client-go)
go doc "$MODULE_DIR"/mittwaldv2/generated/schemas/<schema-package> <TypeName>
```

For each API call site found in step 1, look up:

- The **request** type's fields (what the API accepts on create/update).
- The **response**/**resource** type's fields (what the API returns on
  read/create).

Build a simple per-resource list of API fields vs. Terraform schema attributes
(cross-reference `model.go`'s `tfsdk:"..."` tags and the `Schema()` attribute map).
Note:

- Fields present in the API type but absent from both the Terraform schema *and*
  the model struct → **candidate for missing support**.
- Fields present in the Terraform schema/model but no longer present in the
  current API type → **candidate for stale/removed support**.
- Fields that exist on both sides but under different semantics (renamed,
  retyped, changed nullability) → note as drift too, even though no field is
  literally "missing."

## 3. Judge whether drift is worth exposing

Not everything the API returns belongs in the schema. Use judgment, leaning on
these heuristics:

**Usually worth exposing:**
- A field a user would plausibly want to set or read via Terraform (config knobs,
  identifiers, status/state fields useful for other resources to reference).
- A field that has an obvious, low-risk mapping to an existing Terraform type
  (string, bool, int64, a flat list/map of primitives).
- A field that mirrors something already exposed on a sibling resource/data
  source (e.g. other resources already expose a similar `description` or
  `labels` field).

**Usually not worth exposing (skip, but note why in the audit output):**
- Pure implementation/internal fields (caching hints, internal revision
  counters) with no user-facing meaning.
- Fields duplicating information already available through another attribute or
  another resource/data source.
- Deprecated-on-the-API-side fields.
- Fields that would require a new resource or a major schema redesign to model
  correctly (e.g. a deeply nested polymorphic structure) — these belong in an
  issue, not a quick patch, even though they are "worth exposing" eventually.
- **Volatile, purely informational/observational fields** — metrics or usage
  figures that change on their own outside of any Terraform-managed change
  (e.g. current storage/disk usage, request counts, last-seen timestamps).
  Mirroring these into resource state causes permanent drift/plan-diff noise
  on every apply, since Terraform expects state to only change in response to
  configuration or explicit `Read` reconciliation of user-controlled values —
  not to reflect a constantly-moving number. Skip these even though they are
  real API response fields; don't fix, don't file an issue, just record them
  in the known-drift inventory (§4).

**Sensitive fields** (secrets, passwords, API keys, tokens): if genuinely new and
worth exposing, they must use the write-only pattern from this repo's
`CLAUDE.md` — `<attribute>_wo` plus `<attribute>_wo_version` — never a plain
sensitive attribute.

**Short IDs are cosmetic — prefer full IDs.** Several mittwald API types carry
both a full ID and a "short ID" for the same entity (e.g. a project's
`id`/`shortId`, a server's `id`/`shortId`). Treat the short ID as purely
cosmetic for referencing purposes:
- A resource/data source **is allowed** to expose its *own* short ID as a
  `Computed` attribute (e.g. `mittwald_project`'s `short_id`, already present),
  since that's a legitimate generated property of the entity itself.
- A resource/data source must **not** gain a short-ID attribute for an entity
  it merely *references* (e.g. don't add `server_short_id` to a resource that
  already has `server_id`, even if the API's request/response type carries
  both). Only the full ID is used for cross-resource references in this
  provider.
- This is a standing policy, not a case-by-case call — a "missing"
  `<referenced-entity>_short_id` field is never drift to fix or file; it
  doesn't need a known-drift inventory entry either, since this section
  already documents the rule.

## 4. Check and maintain the known-drift inventory

`.agents/skills/audit-api-drift/known-drift.md` lists API fields that a
previous run deliberately decided *not* to expose (per the §3 heuristics —
most commonly volatile/informational fields; short-ID-for-a-referenced-entity
skips don't need an entry, per §3). It exists so this skill doesn't
re-discover, re-analyze, and re-report the same accepted drift every month.

- **Before** treating anything as drift to fix or file, check this inventory
  for the resource/field pair. If it's already listed, skip it silently — it's
  accepted, not a new finding.
- **When** you apply a §3 "usually not worth exposing" judgment call to a
  *specific* field (not a blanket policy like the short-ID rule), add a row
  instead of just letting it drop, so the decision and its reasoning survive
  to the next run. Use the table format already in the file (resource/data
  source, field, API type/operation, reason, date).
- If a field already in the inventory later becomes something that *should*
  be exposed (e.g. the API changed how it behaves, or a user requests it),
  that's a judgment call for a human — remove the row as part of whatever
  issue/PR addresses it, don't do it silently in an audit run.
- If, over the course of a run, you added any new rows, commit them together
  in one small documentation-only branch/PR at the end (e.g. branch
  `audit-api-drift/known-drift-YYYY-MM-DD`, commit message like
  `chore(audit-api-drift): record accepted API drift`), separate from the
  per-resource fix/issue PRs from §6–§7. If no new rows were added, skip this
  — don't open an empty PR.

## 5. Decide: fix directly, or open an issue

**Fix directly** (small, mechanical, low ambiguity) when all of these hold:
- The field maps cleanly onto an existing Terraform type with no new validators,
  plan modifiers, or nested schema design decisions.
- Adding it doesn't change the meaning or defaults of any existing attribute.
- There's a clear existing pattern elsewhere in the codebase to copy (e.g.
  another resource already has an analogous optional/computed field).
- It doesn't need `go generate` output changes beyond the normal docs
  regeneration, and doesn't touch the "adding a resource/data source" checklist
  items (this is an existing resource — no new registration, README entry, or
  bug-report dropdown entry needed).

**Open an issue instead** when:
- Removing/renaming a schema attribute would be a breaking change.
- The field needs new nested schema types, custom validators, or plan modifiers
  design.
- It's ambiguous whether the field is genuinely gone from the API vs. just
  absent from the particular response you inspected (e.g. omitted-when-empty).
- Exposing it well requires a product/API judgment call (e.g. whether a new
  field should be `Optional` vs `Computed`, or whether it implies a new
  sub-resource).
- Multiple resources would need coordinated changes.

When unsure between "fix" and "issue," prefer the issue — a wrong automated PR is
more costly to review than a well-written issue.

## 6. Fixing straightforward drift

1. Work resource-by-resource: one branch/PR per resource (or per resource+its
   mirrored data source, since those change together), not one giant PR for
   everything found in the run. Branch from an up-to-date `master` for each
   resource (e.g. `audit-api-drift/<resourcename>-YYYY-MM-DD`), and switch
   back to `master` after opening each PR before starting the next resource,
   so unrelated resources never end up on the same branch.
2. Add the field to the model struct(s), the `Schema()` attribute map, and the
   relevant `model_api_to.go` / `model_api_from.go` mapping. Mirror the change
   into the paired data source if one exists, reusing the same mapping helpers
   per this repo's conventions.
3. Update or add an example under `examples/resources/<name>/` or
   `examples/data-sources/<name>/` if the new attribute is significant enough to
   demonstrate.
4. Run `go generate ./...` to regenerate `docs/` — never hand-edit generated docs.
5. Verify:
   ```bash
   go build ./...
   go vet ./...
   golangci-lint run
   go test ./...
   ```
   Do not run `make testacc` / `TF_ACC=1` tests in this automated flow — those
   create real billable resources against the live API.
6. Commit using conventional commits, scoped to the resource:
   - `fix(<resourcename>): ...` when correcting drift the schema should already
     have reflected (e.g. a field the API added a while ago, or a response field
     silently going unread).
   - `feat(<resourcename>): ...` when adding genuinely new optional
     capability that extends what the resource can configure.
   Use your judgment per-change; when both a resource and its data source change
   together, one commit covering both is fine.
7. Push the branch and open a PR with `gh pr create`, describing exactly which
   API field(s) were found missing, the API type/operation they came from, and
   why they were judged safe to add directly. Note in the PR body what
   verification (build/vet/lint/test) was run.

## 7. Filing an issue for judgment-call drift

For each such case, open one issue (`gh issue create`) with:
- The resource/data source affected.
- The API operation and type where the field was found.
- The field name, type, and a short description of what it does (pull from the
  API type's doc comments if present).
- Why it needs human judgment (breaking change risk, design ambiguity, new
  nested schema, etc.) rather than a direct fix.
- A concrete suggestion if you have one, framed as a suggestion, not a decision.

Do not open duplicate issues: search existing open issues for the resource name
and field first (`gh issue list --search "..."`).

## 8. Reporting

If a run finds no drift at all, don't create branches, commits, PRs, or issues —
just report that everything is in sync. If it finds drift but every instance was
already fixed by a prior run (check for existing open PRs/issues covering the
same field before acting), skip re-filing and say so. Also mention in your
summary how many fields were newly recorded in the known-drift inventory (§4),
if any.
