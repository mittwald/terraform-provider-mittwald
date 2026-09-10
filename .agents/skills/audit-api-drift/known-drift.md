# Known / accepted mittwald API drift

This file is maintained by the `audit-api-drift` skill (see `SKILL.md`, §4). It
lists API request/response fields that were deliberately judged *not* worth
exposing in the Terraform schema, so future audit runs recognize them as
already-reviewed instead of re-discovering and re-reporting them.

Do not list here fields governed by a blanket policy already documented in
`SKILL.md` (e.g. short IDs for referenced entities) — only case-by-case
judgment calls, typically volatile/informational fields.

| Resource / data source | Field | API type / operation | Reason | Noted |
| --- | --- | --- | --- | --- |
