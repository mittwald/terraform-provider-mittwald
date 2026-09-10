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
| `mittwald_project` | `backup_storage_usage_in_bytes` (+`_set_at`) | `projectv2.Project` (GetProject) | Volatile usage metrics | 2026-09-10 |
| `mittwald_project` | `web_storage_usage_in_bytes` (+`_set_at`) | `projectv2.Project` (GetProject) | Volatile usage metrics | 2026-09-10 |
| `mittwald_project` | `cluster_domain`, `cluster_id` | `projectv2.Project` (GetProject) | Internal cluster identifiers | 2026-09-10 |
| `mittwald_project` | `deletion_requested` | `projectv2.Project` (GetProject) | Transient; deletion handled by provider Delete | 2026-09-10 |
| `mittwald_project` | `disabled_at`, `disable_reason` | `projectv2.Project` (GetProject) | Disabled state not settable via this client | 2026-09-10 |
| `mittwald_project` | `enabled` | `projectv2.Project` (GetProject) | No setter endpoint; always true for managed projects | 2026-09-10 |
| `mittwald_project` | `features`, `supported_features` | `projectv2.Project` (GetProject) | Plan-derived capability lists (informational) | 2026-09-10 |
| `mittwald_project` | `image_ref_id` | `projectv2.Project` (GetProject) | Internal image reference | 2026-09-10 |
| `mittwald_project` | `is_ready` | `projectv2.Project` (GetProject) | Transient; `status` already exposed | 2026-09-10 |
| `mittwald_project` | `project_hosting_id` | `projectv2.Project` (GetProject) | Internal identifier | 2026-09-10 |
| `mittwald_project` | `readiness` | `projectv2.Project` (GetProject) | Deprecated type; superseded by `status` | 2026-09-10 |
| `mittwald_project` | `server_group_id` | `projectv2.Project` (GetProject) | Internal identifier | 2026-09-10 |
| `mittwald_project` | `statistics_base_domain` | `projectv2.Project` (GetProject) | Informational statistics domain | 2026-09-10 |
| `mittwald_project` | `status` | `projectv2.Project` (GetProject) | Status fields (and related metadata like creation+update timestamps) are not exposed as resource state on purpose (see #473) | 2026-09-10 |
| `mittwald_project` | `created_at` | `projectv2.Project` (GetProject) | Status fields (and related metadata like creation+update timestamps) are not exposed as resource state on purpose (see #473) | 2026-09-10 |
| `mittwald_project` | `status_set_at` | `projectv2.Project` (GetProject) | Volatile timestamp (moves with `status`) | 2026-09-10 |
| `mittwald_server` | `disabled_reason` | `projectv2.Server` (GetServer) | Disabled state not managed by provider | 2026-09-10 |
| `mittwald_server` | `group_id` | `projectv2.Server` (GetServer) | Internal identifier | 2026-09-10 |
| `mittwald_server` | `image_ref_id` | `projectv2.Server` (GetServer) | Internal image reference | 2026-09-10 |
| `mittwald_server` | `is_ready` | `projectv2.Server` (GetServer) | Transient; `status` already exposed | 2026-09-10 |
| `mittwald_server` | `readiness` | `projectv2.Server` (GetServer) | Deprecated type; superseded by `status` | 2026-09-10 |
| `mittwald_server` | `statistics_base_domain` | `projectv2.Server` (GetServer) | Informational statistics domain | 2026-09-10 |
| `mittwald_server` | `machine_type.cpu`, `machine_type.memory` | `projectv2.Server` (GetServer) | Descriptive plan specs; `machine_type` name exposed | 2026-09-10 |
| `mittwald_app` | `auto_updates_activated` | `appv2.AppInstallation` (GetAppinstallation) | Read-only; no setter endpoint | 2026-09-10 |
| `mittwald_app` | `deletion_requested` | `appv2.AppInstallation` (GetAppinstallation) | Transient flag | 2026-09-10 |
| `mittwald_app` | `disabled` | `appv2.AppInstallation` (GetAppinstallation) | Platform-managed; no setter | 2026-09-10 |
| `mittwald_app` | `hostname` | `appv2.AppInstallation` (GetAppinstallation) | Platform-assigned, not a config knob | 2026-09-10 |
| `mittwald_app` | `last_error` | `appv2.AppInstallation` (GetAppinstallation) | Volatile error state | 2026-09-10 |
| `mittwald_app` | `locked_by` | `appv2.AppInstallation` (GetAppinstallation) | Transient lock state | 2026-09-10 |
| `mittwald_app` | `ports` | `appv2.AppInstallation` (GetAppinstallation) | Runtime port descriptors (informational) | 2026-09-10 |
| `mittwald_app` | `project_description` | `appv2.AppInstallation` (GetAppinstallation) | Duplicates `mittwald_project.description` | 2026-09-10 |
| `mittwald_app` | `screenshot_id`, `screenshot_ref` | `appv2.AppInstallation` (GetAppinstallation) | Media references | 2026-09-10 |
| `mittwald_app` | `source_app_installation_id` | `appv2.AppInstallation` (GetAppinstallation) | Staging source ref (see issue #476) | 2026-09-10 |
| `mittwald_app` | `system_software` | `appv2.AppInstallation` (GetAppinstallation) | Duplicates `dependencies` attribute | 2026-09-10 |
| `mittwald_app` | `update_available` | `appv2.AppInstallation` (GetAppinstallation) | Volatile flag | 2026-09-10 |
| `mittwald_app` | `app_external_version` | `appv2.AppInstallation` (GetAppinstallation) | Ambiguous vs `version`/`version_current` | 2026-09-10 |
| `mittwald_app` | `app_id`, `app_name` | `appv2.AppInstallation` (GetAppinstallation) | `app` name is the reference key | 2026-09-10 |
| `mittwald_app` | `phase` | `appv2.AppInstallation` (GetAppinstallation) | Status fields (and related metadata like creation+update timestamps) are not exposed as resource state on purpose (see #474) | 2026-09-10 |
| `mittwald_app` | `created_at` | `appv2.AppInstallation` (GetAppinstallation) | Status fields (and related metadata like creation+update timestamps) are not exposed as resource state on purpose (see #474) | 2026-09-10 |
| `mittwald_app` (data source) | `id`, `tags`, `action_capabilities` | `appv2.App` (GetApp) | Name identifies the app; catalog/action metadata | 2026-09-10 |
| `mittwald_app` (data source) | `recommended` (status) | `appv2.AppVersion` (ListAppversions) | Name collision with `recommended` selector flag | 2026-09-10 |
| `mittwald_app` (data source) | `doc_root`, `databases`, `default_cronjobs`, `system_software_dependencies`, `user_inputs`, `request_handler`, `breaking_note`, `backend_path_template` | `appv2.AppVersion` (ListAppversions) | Nested catalog/dependency schema; resolved by `mittwald_app` resource | 2026-09-10 |
| `mittwald_container_stack` | `disabled` | `containerv2.StackResponse` (GetStack) | No setter endpoint | 2026-09-10 |
| `mittwald_container_stack` | `prefix` | `containerv2.StackResponse` (GetStack) | Derived from stack name | 2026-09-10 |
| `mittwald_container_stack` | `template_id` | `containerv2.StackResponse` (GetStack) | Internal template reference | 2026-09-10 |
| `mittwald_container_stack` | `status`, `status_set_at` | `containerv2.ServiceResponse` (GetStack) | Runtime state + volatile timestamp | 2026-09-10 |
| `mittwald_container_stack` | `message` | `containerv2.ServiceResponse` (GetStack) | Transient status message | 2026-09-10 |
| `mittwald_container_stack` | `requires_recreate` | `containerv2.ServiceResponse` (GetStack) | Transient flag | 2026-09-10 |
| `mittwald_container_stack` | `deployed_state`, `pending_state` | `containerv2.ServiceResponse` (GetStack) | Duplicates config attributes | 2026-09-10 |
| `mittwald_container_stack` | `deploy.image_digest` | `containerv2.ServiceResponse` (GetStack) | Volatile image digest | 2026-09-10 |
| `mittwald_container_stack` | `project_id`, `stack_id` | `containerv2.ServiceResponse` (GetStack) | Redundant with parent/stack-level IDs | 2026-09-10 |
| `mittwald_container_stack` | volume `id` | `containerv2.VolumeResponse` (GetStack) | Internal; name is the reference key | 2026-09-10 |
| `mittwald_container_stack` | volume `linked_services` | `containerv2.VolumeResponse` (GetStack) | Derived relationship | 2026-09-10 |
| `mittwald_container_stack` | volume `orphaned` | `containerv2.VolumeResponse` (GetStack) | Transient GC state | 2026-09-10 |
| `mittwald_container_stack` | volume `stack_id` | `containerv2.VolumeResponse` (GetStack) | Redundant | 2026-09-10 |
| `mittwald_container_stack` | volume `storage_usage_in_bytes` (+`_set_at`) | `containerv2.VolumeResponse` (GetStack) | Volatile usage metrics | 2026-09-10 |
| `mittwald_container_stack` | container `restart_policy` | `containerv2.ServiceDeclareRequest`/`ServiceRequest`/`ServiceResponse` (declare/update/GetStack) | Restart policy support is currently flaky at best and not advertised widely; put on the back burner for now (see #477) | 2026-09-10 |
| `mittwald_mysql_database` | `finalizers` | `databasev2.MySqlDatabase` (GetMysqlDatabase) | Internal k8s finalizers | 2026-09-10 |
| `mittwald_mysql_database` | `is_ready` | `databasev2.MySqlDatabase` (GetMysqlDatabase) | Transient; `status` already exposed | 2026-09-10 |
| `mittwald_mysql_database` | `is_shared` | `databasev2.MySqlDatabase` (GetMysqlDatabase) | Platform-managed flag | 2026-09-10 |
| `mittwald_mysql_database` | `main_user` | `databasev2.MySqlDatabase` (GetMysqlDatabase) | Duplicates `user` block | 2026-09-10 |
| `mittwald_mysql_database` | `status_set_at`, `updated_at` | `databasev2.MySqlDatabase` (GetMysqlDatabase) | Volatile timestamps | 2026-09-10 |
| `mittwald_mysql_database` | `storage_usage_in_bytes` (+`_set_at`) | `databasev2.MySqlDatabase` (GetMysqlDatabase) | Volatile usage metrics | 2026-09-10 |
| `mittwald_mysql_database` | `user.created_at` | `databasev2.MySqlUser` (GetMysqlDatabase) | Low value on nested block | 2026-09-10 |
| `mittwald_mysql_database` | `user.database_id` | `databasev2.MySqlUser` (GetMysqlDatabase) | Redundant with parent | 2026-09-10 |
| `mittwald_mysql_database` | `user.disabled` | `databasev2.MySqlUser` (GetMysqlDatabase) | No setter endpoint | 2026-09-10 |
| `mittwald_mysql_database` | `user.main_user` | `databasev2.MySqlUser` (GetMysqlDatabase) | Derived from `user` block | 2026-09-10 |
| `mittwald_mysql_database` | `user.password_updated_at`, `user.updated_at`, `user.status_set_at` | `databasev2.MySqlUser` (GetMysqlDatabase) | Volatile timestamps | 2026-09-10 |
| `mittwald_mysql_database` | `user.status` | `databasev2.MySqlUser` (GetMysqlDatabase) | Transient user state | 2026-09-10 |
| `mittwald_mysql_database` | `user.description` | `databasev2.MySqlUser` (GetMysqlDatabase) | Not settable on create; low value | 2026-09-10 |
| `mittwald_redis_database` | `finalizers` | `databasev2.RedisDatabase` (GetRedisDatabase) | Internal k8s finalizers | 2026-09-10 |
| `mittwald_redis_database` | `status_set_at`, `updated_at` | `databasev2.RedisDatabase` (GetRedisDatabase) | Volatile timestamps | 2026-09-10 |
| `mittwald_redis_database` | `storage_usage_in_bytes` (+`_set_at`) | `databasev2.RedisDatabase` (GetRedisDatabase) | Volatile usage metrics | 2026-09-10 |
| `mittwald_cronjob` | `failed_execution_alert_threshold` | `cronjobv2.Cronjob` (GetCronjob) | Semantics not documented in client | 2026-09-10 |
| `mittwald_cronjob` | `latest_execution` | `cronjobv2.Cronjob` (GetCronjob) | Volatile execution history | 2026-09-10 |
| `mittwald_cronjob` | `next_execution_time` | `cronjobv2.Cronjob` (GetCronjob) | Volatile schedule state | 2026-09-10 |
| `mittwald_cronjob` | `updated_at` | `cronjobv2.Cronjob` (GetCronjob) | Volatile timestamp | 2026-09-10 |
| `mittwald_cronjob` | `app_installation_id` | `cronjobv2.Cronjob` (GetCronjob) | Deprecated fallback; `target` covers it | 2026-09-10 |
| `mittwald_email_outbox` | `authentication_enabled`, `sending_enabled` | `mailv2.Deliverybox` (GetDeliverybox) | Platform-managed flags; no setter | 2026-09-10 |
| `mittwald_email_outbox` | `password_updated_at`, `updated_at` | `mailv2.Deliverybox` (GetDeliverybox) | Volatile timestamps | 2026-09-10 |
| `mittwald_virtualhost` | `ips` | `ingressv2.Ingress` (GetIngress) | Informational/volatile; project exposes `default_ips` | 2026-09-10 |
| `mittwald_virtualhost` | `is_enabled` | `ingressv2.Ingress` (GetIngress) | Platform-managed; not a config knob | 2026-09-10 |
| `mittwald_virtualhost` | `is_domain` | `ingressv2.Ingress` (GetIngress) | Classification flag | 2026-09-10 |
| `mittwald_virtualhost` | `ownership` (txt_record/verified) | `ingressv2.Ingress` (GetIngress) | DNS verification outside resource scope | 2026-09-10 |
| `mittwald_virtualhost` | `tls` | `ingressv2.Ingress` (GetIngress) | Managed by `mittwald_tls_certificate` | 2026-09-10 |
| `mittwald_virtualhost` | `dns_validation_errors` | `ingressv2.Ingress` (GetIngress) | Volatile validation state | 2026-09-10 |
| `mittwald_ssh_user` | `auth_updated_at` | `sshuserv2.SshUser` (GetSSHUser) | Volatile timestamp | 2026-09-10 |
| `mittwald_ssh_user` | `updated_at` | `sshuserv2.SshUser` (GetSSHUser) | Volatile timestamp | 2026-09-10 |
| `mittwald_ssh_user` | `has_password` | `sshuserv2.SshUser` (GetSSHUser) | A practitioner managing this resource via Terraform already knows whether the SSH user has a password set (see #485) | 2026-09-10 |
| `mittwald_tls_certificate` | `is_expired` | `sslv2.Certificate` (GetCertificate) | Volatile; derivable from `valid_to` | 2026-09-10 |
| `mittwald_tls_certificate` | `certificate_type` | `sslv2.Certificate` (GetCertificate) | Derived from creation mode | 2026-09-10 |
| `mittwald_tls_certificate` | `certificate_order_id` | `sslv2.Certificate` (GetCertificate) | Internal order reference | 2026-09-10 |
| `mittwald_tls_certificate` | `contact` | `sslv2.Certificate` (GetCertificate) | Billing domain | 2026-09-10 |
| `mittwald_tls_certificate` | `dns_cert_spec` | `sslv2.Certificate` (GetCertificate) | Internal DNS-01 challenge spec | 2026-09-10 |
| `mittwald_tls_certificate` | `last_expiration_threshold_hit` | `sslv2.Certificate` (GetCertificate) | Internal metric | 2026-09-10 |
| `mittwald_ai` | `contract_number`, `termination`, contract item details | `contractv2.Contract` (GetDetailOfContractByAIHosting) | Billing domain; only `contract_id`/`article_id`/`customer_id` exposed (consistent with project/server) | 2026-09-10 |
| `mittwald_ai_api_key` | `is_blocked` | `aihostingv2.Key` (CustomerGetKey) | Operational state; not settable via this client | 2026-09-10 |
| `mittwald_ai_api_key` | `models` | `aihostingv2.Key` (CustomerGetKey) | Plan-derived | 2026-09-10 |
| `mittwald_ai_api_key` | `plan_id`, `profile_id` | `aihostingv2.Key` (CustomerGetKey) | Opaque references to unmodeled entities | 2026-09-10 |
| `mittwald_ai_api_key` | `rate_limit` | `aihostingv2.Key` (CustomerGetKey) | Plan-derived; shared across project | 2026-09-10 |
| `mittwald_ai_api_key` | `token_usage` | `aihostingv2.Key` (CustomerGetKey) | Volatile usage metrics | 2026-09-10 |
| `mittwald_ai_api_key` | `create_webui_container` | `aihostingv2.CustomerCreateKeyRequestBody`/`CustomerUpdateKeyRequestBody` (create/update) | Convenience feature for the GUI only; not included in the Terraform provider on purpose (see #484) | 2026-09-10 |
| `mittwald_ai_api_key` | `container_meta` | `aihostingv2.Key` (CustomerGetKey) | Convenience feature for the GUI only; not included in the Terraform provider on purpose (see #484) | 2026-09-10 |
| `mittwald_user` (data source) | `avatar_ref` | `userv2.User` (GetUser) | Media reference | 2026-09-10 |
| `mittwald_user` (data source) | `customer_memberships`, `project_memberships` | `userv2.User` (GetUser) | Nested maps of referenced entities | 2026-09-10 |
| `mittwald_user` (data source) | `employee_information`, `is_employee` | `userv2.User` (GetUser) | Platform-internal | 2026-09-10 |
| `mittwald_user` (data source) | `mfa` | `userv2.User` (GetUser) | Security state outside resource scope | 2026-09-10 |
| `mittwald_user` (data source) | `password_updated_at` | `userv2.User` (GetUser) | Volatile timestamp | 2026-09-10 |
| `mittwald_user` (data source) | `phone_number` | `userv2.User` (GetUser) | PII | 2026-09-10 |
| `mittwald_article` (data source) | `addons`, `modifier_articles`, `possible_article_changes`, `template` | `articlev2.ReadableArticle` (ListArticles/GetArticle) | Nested option catalogs | 2026-09-10 |
| `mittwald_article` (data source) | `balance_addon_key`, `forced_invoicing_period_in_month`, `has_independent_contract_period`, `hide_on_invoice` | `articlev2.ReadableArticle` (ListArticles/GetArticle) | Billing internals | 2026-09-10 |
| `mittwald_container_image` (data source) | `is_ai_available`, `has_ai_generated_data` | `containerv2.ContainerImageConfig` (GetContainerImageConfig) | Platform catalog internals | 2026-09-10 |
| `mittwald_container_image` (data source) | `overwriting_user` | `containerv2.ContainerImageConfig` (GetContainerImageConfig) | Undocumented semantics | 2026-09-10 |
| `mittwald_systemsoftware` (data source) | `fee` | `appv2.SystemSoftwareVersion` (ListSystemsoftwareversions) | Billing | 2026-09-10 |
| `mittwald_systemsoftware` (data source) | `system_software_dependencies` | `appv2.SystemSoftwareVersion` (ListSystemsoftwareversions) | Auto-resolved by `mittwald_app.dependencies` | 2026-09-10 |
| `mittwald_systemsoftware` (data source) | `user_inputs` | `appv2.SystemSoftwareVersion` (ListSystemsoftwareversions) | Managed via `mittwald_app.user_inputs` | 2026-09-10 |
| `mittwald_systemsoftware` (data source) | `recommended` (status) | `appv2.SystemSoftwareVersion` (ListSystemsoftwareversions) | Name collision with `recommended` selector flag | 2026-09-10 |
