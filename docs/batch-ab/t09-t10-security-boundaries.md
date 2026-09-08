# Batch B T09/T10 security/auth/tenant boundaries

Final workstream status: implemented, focused real PostgreSQL/Redis security regressions passed, ready for parent integration. **Not a claim that every T10 scenario or production release has been accepted.** No commits, frontend edits, real SMS deliveries, provider payments/refunds, or production changes were made by this workstream.

## Implemented fixes and evidence

| Boundary | Final implementation | Test evidence |
|---|---|---|
| JWT | Removes fixed source secret; independent config/env secret with startup validation (Zeno). HS256 and issuer fixed; expiry required; issued-at checked; positive tenant/user, valid member/admin type; explicit access/refresh purpose; random 256-bit JTI | TestJWTBoundaries |
| Required/optional auth | Supplied invalid credentials, unavailable Redis and mismatched stored identity reject. Optional authentication permits anonymous only without a credential. Member/admin paths reject the other user type, even overlapping numeric IDs | TestAuthenticationBoundaries |
| Refresh and logout | Separate access/refresh namespaces. Atomic pair persistence; refresh atomically consumes proof and revokes prior access; failed issuance stays revoked. Logout deletes its entire pair and propagates Redis errors | TestOAuthTokenSecurity: 16 simultaneous refresh attempts, exactly one winner; logout prevents refresh resurrection |
| Fixed frontend refresh | Exact admin/member refresh routes validate refresh JWT+Redis directly; stale/expired access header does not override refresh. No general expired-token exception. Header tenant must match refresh tenant; active tenant checked; replay denied | TestRefreshIgnoresExpiredAccessHeader invokes the real admin refresh handler with an expired signed access header |
| Tenant identity | Member tokens use persisted tenant instead of 0. Admin login requires trusted tenant instead of default1. Anonymous tenant comes from registered hostnames; arbitrary header alone cannot set it. Authenticated HTTP checks current user and tenant status | TestTrustedTenantHostBoundary and TestAuthenticatedDisabledUserRejected |
| GORM tenant isolation | SELECT/count/UPDATE/DELETE add qualified tenant filter. INSERT checks/stamps each row. Missing context, foreign inserts, tenant mutation, raw SQL, joins, subqueries/table overrides and conflict upserts reject. Own GORM Save works; cross-tenant fallback rejects. Select(*) updates omit tenant_id, preserving identity even from zero structs | TestTenantDatabaseBoundaries against real PostgreSQL, with A/B final-state assertions and ordinary job context |
| Permission isolation/revocation | Startup Casbin policy qualified by tenant; reviewed role joins require equal tenant, enabled/deleted checks. Runtime permission decisions use live database grants, so grants/revocations do not wait for cache reload. Disabled roles/menus cannot retain access | TestPolicyStartupTenantIsolation, including deliberately corrupt cross-tenant role binding, role disable and restricted scope |
| Admin trade RBAC | Every normal order, after-sale, delivery, config and brokerage route has explicit RequirePermission. New source-derived permission rows are migration6 (Zeno). Provider callback remains a separate C-batch contract | TestTradeRoutesRequireLivePermission walks 53 registered normal trade routes with authenticated admin lacking grants and asserts403 before handlers |
| Anonymous private routes | Sign-in records and kefu messages require Auth. Kefu uses actual userID context key. Admin user/dept/post/role/menu/SMS simple lists require admin Auth. App pay group required auth retained | TestPrivateRoutesRejectAnonymous:23 registered endpoints; nil service dependencies make an escaped handler fail loudly |
| Member payment owner | App Get and Submit validate persisted TradeOrder.PayOrderID+UserID. Existing exposed recharge path validates recharge->wallet user/type. Same-tenant other members cannot read/sync/submit another payment. Member transfer sync explicitly403 unsupported with no service call | TestMemberPayOrderOwnership and TestMemberTransferSyncDisabled; no SDK calls, final status unchanged and no extension rows |
| Payment client cache | GetPayClient(ctx,id) validates trusted tenant, DB channel enabled/ownership and matching enabled payment app before cached lookup. Order direct-factory bypasses and refund/transfer/notify callers updated | TestPaymentClientTenantBoundary: populated cache inaccessible cross-tenant, without context, or after channel/app disable |
| Detached local payment result | SubmitOrder no longer launches a goroutine with incoming Gin context; local NotifyOrder is synchronous and its error propagates. No C-batch provider/state/signature redesign | Source review and compiled/focused package tests; parent owns notification lifecycle integration |
| Password/defaults | No123456 fallback. Shared validator enforces explicit8..72-byte admin password in creation, admin change/reset, and auth reset paths. Offline bootstrap stricter policy remains Zeno-owned | TestAdminPasswordLengthBoundary and platform guard integration's weak update/reset/create rejection |

Captcha proof consumption when supplied, atomic SMS consumption in admin/member login/reset/mobile change, and their failure cases integrate the complementary agent's work. See `t09-login-security.md`. Parent owns CORS/request-log redaction, rate-limit wiring, file/cache, WeChat cache, scheduler and global integration evidence.

## Real, bounded platform administrator workflow

`GET /admin-api/system/security/tenant-inspect?targetTenantId=<positive ID>` requires ordinary authenticated admin context plus a current enabled **system-type reserved platform_admin role in the actor's source tenant**. It returns only target tenant ID, product count and order count. It never returns a target context, SQL capability, records, credentials, or a generic cross-tenant bypass.

The service validates actor user, source tenant, reserved role and target existence. It creates a fresh, cancellable ordinary context containing only the trusted target tenant, preserving the original actor/Gin identity unchanged. All target reads still pass TenantPlugin. Caller cancellation propagates into this bounded read. `visit-tenant-id` does not activate this capability.

Every inspection attempt writes structured `security_audit` with actorUserId, sourceTenantId, targetTenantId, operation=`platform.tenant.inspect`, outcome=success/denied. Tests capture the actual audit sink and verify these fields. Operational retention/collection of that sink belongs to parent deployment logging.

Provisioning is explicit offline `RUOYI_BOOTSTRAP_PLATFORM_ADMIN=true` in `cmd/bootstrap` (Zeno); no default platform grant, shared menu grant or web grant. Bootstrap only provisions an explicitly new user and refuses to reset an existing user. The reserved role and super_admin cannot be created/renamed/assigned through role, permission or UserSaveReq.RoleIDs alternate paths. Existing reserved bindings cannot be removed through ordinary assignment. Reserved actors cannot be taken over by admin profile edits, password/reset/status/delete/bulk-delete methods; public auth password reset also rejects them.

`TestPlatformInspectionAndReservedRoleGuards` uses A/B tenants plus platform and normal actors in real PostgreSQL: A/B count success, normal actor denial, actor context unchanged, required audit fields, role/user alternate-path escalation rejection and immutable database state. The endpoint is a real platform-admin workflow; startup RBAC loading is a separate read-only audited exception and is not presented as the business workflow.

Platform tenant/menu/dictionary management **writes remain NOT ENABLED**. There is no blanket platform write scope or generic cross-tenant HTTP header.

## First-phase data-scope contract: NOT ENABLED beyond tenant-wide RBAC

Departmental, custom-department and self-only data-scope semantics are **NOT ENABLED** in this first phase. The existing unregistered `internal/pkg/datascope` plugin is not claimed or activated; blanket registration would affect authentication/metadata and would not provide reliable restrictions.

- RoleService.UpdateRoleDataScope rejects non-All values and nonempty custom department IDs.
- TrustedTenant validates every authenticated administrator's active role scopes before any handler, including Auth-only/simple-list endpoints and existing tokens. Any active restricted role, including mixed All+restricted roles, rejects rather than broadening silently.
- IsSuperAdmin, HasPermission and platform inspection also fail closed for unsupported scopes and database errors; no superadmin early return evades the check.
- Supported All scope remains writable. Tests assert unsupported changes leave persisted role scope unchanged; mixed-role HTTP denial and All restored success are covered.

Future departmental/self permission semantics require a separately tested data-access design, not removal of these rejection guards. Member ownership checks remain independent of administrator data scopes.

## Context, table and query policy

Order: `InjectContext -> TrustedTenant(db) -> LoginRateLimit -> route Auth/RequirePermission`.

`pkg/context.WithTenant(ctx,id)` is only for trustworthy server-side host registrations, persisted records and jobs. `WithLoginUser` copies identity into ordinary context; no detached job must rely on a live Gin pointer. Tenant IDs must be positive. Platform inspection deliberately uses a fresh context, because ordinary WithTenant on an actor context does not replace LoginUser.TenantID.

Anonymous resolution uses Request.Host and registered SystemTenant.Websites, ignores X-Forwarded-Host, checks status/expiry, rejects ambiguous registrations unless a matching tenant-id selects one. Public tenant-discovery and non-API liveness/root paths are explicit exceptions. Refresh is an exact-path credential exception as described above.

Shared read-only table whitelist: system_tenant, system_tenant_package, system_menu, system_dict_type, system_dict_data, system_area. Unknown models without TenantID reject. System config, OAuth clients, notify templates, SMS channels/templates/logs/codes and mail accounts/templates now have TenantBaseDO; Zeno included columns in migrations. Existing member level/config/point, trade config and access/error logs were checked; parent owns file/job conversions. Unreviewed IoT models are not implicitly shared.

Reviewed startup policy scope is read-only, logged and limited to role/user/menu table queries; its write attempts reject. Unreviewed raw SQL, joins, schema-less scans and conflict upserts reject instead of being rewritten. Trade config/payment notification task gen.Save was replaced with scoped Updates preserving zero values; product/SKU equivalents belong to transaction agent. This ORM guard is not PostgreSQL RLS and does not sandbox future arbitrary database/sql code.

## Verification command and final evidence

Environment: PostgreSQL16 at127.0.0.1:55439; private Redis at127.0.0.1:56397 DB1; shared recorded Go toolchain. PostgreSQL tests create independent schemas with search_path on every pool connection. Redis tests create random token keys and clean only those keys; no FLUSHDB. Missing integration environment variables cause explicit skips and are not counted as acceptance.

```sh
rtk proxy env GOCACHE=/private/tmp/ruoyi-ab-go-build GOPATH=/private/tmp/ruoyi-ab-gopath \
 TEST_POSTGRES_DSN='host=127.0.0.1 port=55439 user=macmini dbname=ruoyi_ab sslmode=disable' \
 T09_REDIS_ADDR=127.0.0.1:56397 \
 go test -race ./pkg/utils ./pkg/context ./pkg/database ./internal/pkg/permission \
 ./internal/middleware ./internal/service/system ./internal/service/member ./internal/service/pay \
 ./internal/api/handler/app/pay ./internal/api/router \
 -run 'TestJWTBoundaries|TestTenantDatabaseBoundaries|TestPolicyStartupTenantIsolation|TestAuthenticationBoundaries|TestTrustedTenantHostBoundary|TestAuthenticatedDisabledUserRejected|TestOAuthTokenSecurity|TestLoginRejectsInvalidCaptchaBeforeDatabase|TestPaymentClientTenantBoundary|TestTradeRoutesRequireLivePermission|TestPrivateRoutesRejectAnonymous|TestRefreshIgnoresExpiredAccessHeader|TestPlatformInspectionAndReservedRoleGuards|TestMemberPayOrderOwnership|TestMemberTransferSyncDisabled|TestAdminPasswordLengthBoundary' \
 -count=1 -v
```

Final raw evidence: **evidence/security-final-all-race.log**,16 top-level security tests, real PG/Redis, race detector. Earlier narrower evidence remains in security-focused-race.log, security-final-boundaries-race.log and security-route-boundaries.log. Parent owns generated DAO/Wire, full-suite/build/HTTP smoke; this workstream did not regenerate DAO or commit anything. Original defects were source-confirmed; shared worktree was not reverted to run new regression tests against the old insecure source.

## Remaining limits and acceptance classification

| Handoff case | This workstream | Consolidated acceptance still needed |
|---|---|---|
| A24 JWT/refresh/revocation/roles | Focused regression passed, including original expired-header refresh shape | Parent original frontend smoke and full enabled-interface matrix |
| A25 Redis/default configuration | Token failure tests passed; config/startup and actual Redis process-kill evidence owned cooperating agents | Parent consolidated report |
| A26 HTTP/Raw/cache/platform | Focused A/B isolation, member ownership, real audited platform read and cache-boundary tests passed | Full enabled-route and all-cache/file matrix; platform writes remain NOT ENABLED |
| A27 detached context/jobs/payment/files | Ordinary-context isolation and payment-cache ownership passed; captured Gin goroutine removed | Parent scheduler/file/lifecycle tests and C-batch callbacks/provider trust recovery |

Remaining limits: shared callback hostname ambiguous across tenants rejects; no claimed completion of C-batch provider signature validation, payment-channel creation/signing, notification recovery or real provider tests. Payment-channel config hot replacement/versioning is still channel-lifecycle work; lookup rechecks ownership/status. Logout revokes one pair, not every session on account password reset; current disabled users reject at HTTP and refresh. Departmental/custom/self scopes and member transfer-sync are explicitly not enabled. No production readiness or complete T10 acceptance follows from these focused results alone.
