---
excludeAgent: "coding-agent"
---

When reviewing a pull request, pay special attention to the following aspects:

- For sensitive attributes like passwords and API keys, use write-only attributes with a versioning field whenever possible. By convention, these attribute should be named `<attribute>_wo` and `<attribute>_wo_version`.
- Pay attention to the mapping code between Terraform schema and API models. Ensure that the code follows existing patterns in the codebase for consistency, using `model_api_to.go` for mapping Terraform schema _to_ API models, and `model_api_from.go` for mapping API models _from_ API responses to Terraform schema.
- Ensure that functions (especially mapping functions) are not excessively long and are broken down into smaller, manageable pieces where appropriate.
- Assert that attributes are properly validated and that custom validators are used where necessary.
- Assert that attributes are thouroughly documented with clear and unambiguous descriptions. Descriptions should not contain spelling or grammatical errors.
