---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mittwald_systemsoftware Data Source - terraform-provider-mittwald"
subcategory: ""
description: |-
  A data source that selects versions of system components, such as PHP, MySQL, etc.
  This data source should typically be used in conjunction with the mittwald_app
  resource to select the respective versions for the dependencies attribute.
---

# mittwald_systemsoftware (Data Source)

A data source that selects versions of system components, such as PHP, MySQL, etc.

This data source should typically be used in conjunction with the `mittwald_app`
resource to select the respective versions for the `dependencies` attribute.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The system software name

### Optional

- `recommended` (Boolean) Set this to just select the recommended version
- `selector` (String) A version selector, such as `>= 7.4`; if omitted, this will default to `*` (all versions)

### Read-Only

- `version` (String) The selected version
- `version_id` (String) The selected version ID