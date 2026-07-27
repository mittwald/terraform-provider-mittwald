---
name: Bug report
about: Report unexpected behaviour of the mittwald Terraform provider
title: ""
labels: bug
assignees: ""
---

**Describe the bug**

_A clear and concise description of what the bug is._

**Affected resources and data sources**

_Which resources, data sources or actions are affected? E.g. `mittwald_app`, `data.mittwald_project`._

-

**Terraform configuration**

_A minimal configuration that reproduces the problem. Please remove anything unrelated to the bug, and redact secrets (API tokens, passwords, certificates)._

```hcl
resource "mittwald_..." "example" {
  # ...
}
```

**Steps to reproduce**

1. `terraform apply` the configuration above
2. ...
3. See error

**Expected behavior**

_A clear and concise description of what you expected to happen (e.g. "the plan should be empty on the second run")._

**Actual behavior**

_What happened instead? Include the error message, the unexpected plan diff, or the state that was written._

**Debug output**

_If applicable, add Terraform output to help explain your problem. Please double-check the output for credentials before posting it — `TF_LOG=debug` may include API request and response bodies._

<!--
<details>

<summary>Full output with <code>TF_LOG=debug</code></summary>

```
ENTER FULL DEBUG OUTPUT HERE
```

</details>
-->

**Panic output**

_If Terraform produced a panic, please paste the contents of `crash.log` here (or attach it as a file)._

**Environment:**

_Please complete the following information:_

- Terraform / OpenTofu version: [e.g. terraform 1.14.2, or opentofu 1.10.5] (`terraform version`)
- Provider version: [e.g. 1.10.9]
- Operating system: [e.g. macOS 15.5, Ubuntu 24.04]

**Additional context**

_Add any other context about the problem here: does it happen consistently or intermittently? Did it work with an earlier provider version? Links to related issues, pull requests or documentation._
