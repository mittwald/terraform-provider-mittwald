# [1.7.0](https://github.com/mittwald/terraform-provider-mittwald/compare/v1.6.0...v1.7.0) (2026-02-19)


### Bug Fixes

* **container_stack:** use plan volumes for disregardUnknown check ([#324](https://github.com/mittwald/terraform-provider-mittwald/issues/324)) ([180ba7a](https://github.com/mittwald/terraform-provider-mittwald/commit/180ba7a202089d7f2184f0f4f814b8de87994ac0))
* **mysql_database:** fix Update to correctly read state and persist changes ([#322](https://github.com/mittwald/terraform-provider-mittwald/issues/322)) ([f404a08](https://github.com/mittwald/terraform-provider-mittwald/commit/f404a08eecf7c74109c1b2f0f9a6615bddd00462))
* **ux:** prevent inadvertent usage of short IDs where full UUIDs are expected ([#327](https://github.com/mittwald/terraform-provider-mittwald/issues/327)) ([e672c36](https://github.com/mittwald/terraform-provider-mittwald/commit/e672c363e722dcb60984d6b65d0e6bc591f17f32)), closes [mittwald/terraform-provider-mittwald#276](https://github.com/mittwald/terraform-provider-mittwald/issues/276) [mittwald/terraform-provider-mittwald#291](https://github.com/mittwald/terraform-provider-mittwald/issues/291)


### Features

* **ssh_user:** add mittwald_ssh_user resource and read_ssh_publickey function ([#323](https://github.com/mittwald/terraform-provider-mittwald/issues/323)) ([c44a6e9](https://github.com/mittwald/terraform-provider-mittwald/commit/c44a6e99af898e2ed8823828cb9aeebc2afb7397)), closes [#286](https://github.com/mittwald/terraform-provider-mittwald/issues/286)

# [1.6.0](https://github.com/mittwald/terraform-provider-mittwald/compare/v1.5.3...v1.6.0) (2026-02-18)


### Features

* **ai:** add resources for AI hosting ([#296](https://github.com/mittwald/terraform-provider-mittwald/issues/296)) ([0284847](https://github.com/mittwald/terraform-provider-mittwald/commit/0284847f873179d238d7de6310b360f2e196a2c8))
* **article:** create mittwald_article data source ([#295](https://github.com/mittwald/terraform-provider-mittwald/issues/295)) ([b5973d6](https://github.com/mittwald/terraform-provider-mittwald/commit/b5973d61d8d45cb745751688c888fb04b7f34510))
* **container_stack:** add CPU and memory limits support ([#320](https://github.com/mittwald/terraform-provider-mittwald/issues/320)) ([3b1c276](https://github.com/mittwald/terraform-provider-mittwald/commit/3b1c276f837410f0b16a544afc9fce777c04fcd5)), closes [mittwald/terraform-provider-mittwald#319](https://github.com/mittwald/terraform-provider-mittwald/issues/319)
