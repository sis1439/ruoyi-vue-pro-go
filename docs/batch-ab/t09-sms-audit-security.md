# T09 SMS audit exposure follow-up

2026-09-07. Targeted acceptance passed. No commits, dependencies, schema changes, external SMS sends or historical data rewrites.

## Concrete findings

- SystemSmsCode.Code held the six-digit OTP in a varchar(6) SQL audit column. Repository-wide call-site search found no handler/admin page for SystemSmsCode; its only runtime use was SMS code creation and the redundant Code predicate in the consumed-code audit update. This was a database-at-rest exposure, not a confirmed direct API exposure.
- SystemSmsLog.TemplateContent held the rendered OTP message and TemplateParams included the raw code. SmsLogService.convertResp passed both into the admin SMS-log page response. ExportSmsLogExcel called the same page service and copied both fields into spreadsheet cells. Therefore an authorized SMS-log reader/exporter could read still-valid authentication codes.
- SMS send failures persisted err.Error() into ApiSendMsg; provider success responses also persisted raw messages/request identifiers. Those values were included in page/export responses and could contain mobile numbers or echoed credentials. The create-log failure logger used zap.Error(err).
- The DEBUG SMS client explicitly logged mobile, template ID and all template parameters. Client initialization also logged raw errors. Authentication code delivery already rejects DEBUG, but the direct administrative SMS-send route could still invoke it.

## Changes and preserved behavior

SystemSmsCode.Code now stores `[OTP]`, which fits the existing column. A full digest does not fit varchar(6); truncating a hash would not provide a meaningful secret-storage guarantee. No schema expansion or new key/configuration was introduced.

Redis remains authoritative for code verification and atomically consumes the `auditID:code` value. The SQL update then guards the immutable audit ID, tenant, phone, scene and unused state. The obsolete SQL Code equality predicate was removed. The real-Redis/PostgreSQL test now proves that consumption succeeds with the nonsecret SQL marker and exactly one of 32 concurrent attempts succeeds. Wrong/expired code, database-failure and replay tests remain passing.

Auth SMS audit records are recognized by the existing scene-template list or a code parameter. New log writes replace message content, all parameters and mobile with redaction values. Raw provider messages/identifiers are removed; only canonical OK/FAILED send status remains, preserving the authentication code activation guard. Redaction uses copies so the actual provider parameter map retains the original OTP. The production sender still receives the proper code.

The same redaction runs when converting records to page/export responses, including historical unredacted records. This prevents legacy records from exposing their payloads through these interfaces. Full-mobile filtering of newly redacted auth-SMS log rows is no longer available; log IDs, tenant, template/channel, user, time and status remain available for audit.

Send failures return generic delivery/client/audit errors; raw failure errors are not persisted or emitted by explicit SMS log calls. DEBUG logs emit only a safe simulation event. Client initialization logs only a numeric channel ID. No raw error, mobile, template parameter or provider token is included in these explicit events. This does not claim inspection of every third-party SDK internal log.

## Evidence

Before changes, the two new assertions failed:

- `audit row contains OTP instead of marker`
- `audit leaks 739281` (synthetic fixture)

Saved output: `evidence/t09-sms-audit-red.log`.

After changes, seven selected tests pass with the race detector, real Redis and PostgreSQL, no skips or race reports:

```sh
rtk proxy env GOCACHE=/private/tmp/ruoyi-ab-go-build GOPATH=/private/tmp/ruoyi-ab-gopath T09_REDIS_ADDR=127.0.0.1:56397 TEST_POSTGRES_DSN='postgres://macmini@127.0.0.1:55439/ruoyi_ab?sslmode=disable' go test -race ./internal/service/system ./internal/service/system/sms/client/debug ./internal/api/handler/admin/system -run 'TestSms|TestDebugSMS' -count=1 -v
```

Saved raw output: `evidence/t09-sms-audit-race.log`.

The tests check new SQL audit rows, failed and successful provider diagnostic updates, preservation of the provider input map and OK status, legacy service responses, the actual HTTP page, decoded Excel export cells, captured DEBUG zap events, and all previous SMS replay/quota/failure cases. No fixture contacts a real SMS provider.

The parent owns GORM parameterized logging and hidden HTTP query logging. Those files were not changed. This follow-up also makes no migration or destructive cleanup of historical rows/backups; API/export read redaction is effective immediately, while historical database-at-rest cleanup remains a separate data operation if such rows exist.

Read-only inspection of the shared `ruoyi_ab.public` acceptance schema after the fix returned zero rows where SystemSmsCode.Code matches a raw 4–6 digit code, and zero SMS-log rows whose code parameter is present and not redacted. Only counts were queried; no code or phone values were printed, and no cleanup was performed. These counts describe the local acceptance database, not any external deployment or backup.
