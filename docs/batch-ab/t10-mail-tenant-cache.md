# T10 MailService tenant-scoped resolution

2026-09-07. Targeted acceptance passed. Production changes are confined to `internal/service/system/mail_service.go`; added `mail_service_tenant_test.go` and this report/evidence. No dependencies, commits, schema changes, SMTP connections or mail sends.

## Confirmed problem

MailService constructor and account/template CRUD methods called RefreshCache. It executed unscoped account/template full-table Find queries, ignored both errors, and built global maps keyed by account ID and template code. TenantPlugin rejected those reads, resulting in silently empty caches. If tenant filtering were bypassed, same-code templates from multiple tenants would instead compete for one global cache entry.

The initial real-PostgreSQL A/B send test failed with `code: 1002024000, msg: 邮件模版不存在`, despite both tenants having the template. Raw failure evidence is in `evidence/t10-mail-tenant-red.log`.

## Minimal change

Removed both cache maps, mutex, RefreshCache and all refresh calls. Constructor now only stores the database handle and performs no reads. No replacement cache, refresh job, global fallback or dependency was introduced.

SendSingleMail requires a positive trusted tenant, queries the template code with `db.WithContext(ctx)` and an explicit tenant predicate, and obtains the referenced account through GetMailAccount with the same context and explicit tenant predicate. Therefore a template cannot select another tenant's SMTP configuration, even when the test deliberately omits TenantPlugin. The audit row is also explicitly assigned the trusted tenant.

Template/account database errors are propagated; not-found business errors retain their existing mapping. Account configuration and template changes are visible on the next call without refreshing anything. Disabled-template behavior remains unchanged.

Related ignored errors within this file were removed: account/template relationship count, JSON parameter serialization, recipient lookup, and the final audit status update. The final send result joins transport and audit-update errors. Admin recipient resolution uses a typed SystemUser query with context and explicit tenant filtering instead of a schema-less scalar Table/Scan that TenantPlugin denies. The member model has no email field; without explicit recipients it now returns the existing missing-mail error directly instead of attempting an invalid query and swallowing its error.

## Verification

Three tests pass with -race against real PostgreSQL 16.10 on localhost:55439, database ruoyi_ab, unique migrated schemas. No selected test was skipped and no race was reported.

```sh
rtk proxy env GOCACHE=/private/tmp/ruoyi-ab-go-build GOPATH=/private/tmp/ruoyi-ab-gopath TEST_POSTGRES_DSN='postgres://macmini@127.0.0.1:55439/ruoyi_ab?sslmode=disable' go test -race ./internal/service/system -run 'TestMailSameCodeTenantSelection|TestMailExplicitTenantAndDependencyFailures|TestMailRecipientLookupUsesTenantModel' -count=1 -v
```

Raw passing output: `evidence/t10-mail-tenant-race.log`.

- Same template code in tenants A and B selects the correct template, rendered title, sender, account ID, host, username, password, port and SSL/STARTTLS settings. Alternating A/B/A cannot reuse another tenant's data. Account host and template-title updates are immediately observed; B remains independent.
- A separate test intentionally has no TenantPlugin. Constructor performs zero reads; tenantless requests perform zero queries; a B-only template and a cross-tenant account reference are rejected. Template/account query errors, relationship count errors, JSON errors, nil database and a physically closed PostgreSQL connection propagate.
- With TenantPlugin enabled, admin fallback recipient is read from the correct tenant's typed user model. A foreign user ID, unsupported member fallback, and recipient-query failure are rejected.

The tests execute the public SendSingleMail method and real SQL lookups. A test-only GORM callback captures the impending mail-audit record and returns a sentinel error before it can be persisted. Since production only invokes doSend after a successful audit insert, SMTP cannot be reached. The selected account configuration is captured from the actual query result; test output does not print credentials. No production transport injection seam was added solely for testing.

## Coordination and scope

The parent owns NotifyService and LoginLog changes. Neither was edited here.

During inspection, MailHandler.SendMail was found to construct a request without binding its JSON body. This separate handler issue was sent to parent; no handler was modified here. These tests establish service-level tenant/config selection, not an assertion that the complete HTTP test-send route or an SMTP server was exercised. Existing SMTP transport/TLS behavior was not redesigned or acceptance-tested. Post-send audit error propagation is implemented but no live-send path was executed to test it.
