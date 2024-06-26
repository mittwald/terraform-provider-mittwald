---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mittwald_virtualhost Resource - terraform-provider-mittwald"
subcategory: ""
description: |-
  This resource models a virtualhost.
---

# mittwald_virtualhost (Resource)

This resource models a virtualhost.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `hostname` (String) The desired hostname for the virtualhost.
- `paths` (Attributes Map) The desired paths for the virtualhost. (see [below for nested schema](#nestedatt--paths))
- `project_id` (String) The ID of the project the virtualhost belongs to

### Read-Only

- `id` (String) The generated virtualhost ID

<a id="nestedatt--paths"></a>
### Nested Schema for `paths`

Optional:

- `app` (String) The ID of an app installation that this path should point to.
- `redirect` (String) The URL to redirect to.
