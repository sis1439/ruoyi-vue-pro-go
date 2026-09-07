# T09 / T10 complementary login security

Status: targeted acceptance passed, 2026-09-07. Shared backend working tree; no commits created. This report covers captcha, the global login limiter and SMS codes, not the complete T09/T10 epic.

## Ownership and integration

Changed captcha service and admin handler, SMS code service, added global login rate limiter and four test files. No edits to auth.go, OAuth/JWT, configuration, database, router.go, CORS or file services were made by this workstream.

Coordinated with Epicurus security agent `01a07c09-399d-7de2-84e8-adf888f2b7e8`. Parent has wired `InjectContext → TrustedTenant(db) → LoginRateLimit(rdb)` in router.go. Trusted tenant comes from server-resolved host mapping for anonymous requests or verified identity for authenticated requests. Neither service reads tenant headers directly.

Signatures:

- `middleware.LoginRateLimit(rdb *redis.Client) gin.HandlerFunc`
- Existing `system.NewCaptchaService(rdb *redis.Client)` preserved.
- Existing `CaptchaService.Verify(ctx, token, x, y) (bool, error)` preserved; challenge is now atomically consumed.
- `CaptchaService.Check(ctx, token, pointJSON) (bool, error)` stores a proof after success.
- `CaptchaService.ConsumeVerification(ctx, verification string) error` consumes that proof once. Epicurus added the optional nonempty-value check in AuthService.Login.
- Existing `SmsCodeService.UseSmsCode(ctx, mobile, scene, code, usedIp) error` preserved. Epicurus switched admin/member SMS login, admin/member password reset and member mobile update to this call. The explicit SMS validation endpoint remains non-consuming.

## Implemented behavior

### Captcha

Challenge keys contain positive trusted tenant IDs and canonical UUID tokens. Redis GETDEL makes each challenge attempt single-use, including wrong attempts; missing context, nil Redis and Redis failure reject the operation. Challenge lifetime remains five minutes. Response images contain no answer coordinate fields. HTTP errors are generic rather than exposing Redis/server errors. Empty get bodies remain supported; malformed or oversized input is rejected.

The pinned Vue reference VerifySlide component constructs `token---pointJson` because secretKey is empty. The backend preserves that exact contract. A successful check stores only its SHA-256 proof under a separate tenant/UUID key for two minutes. Login consumes that key with GETDEL and compares the digest in constant time. Unchecked, changed, expired, cross-tenant and replayed proofs fail. Optional means omitted values remain allowed at the AuthService caller; any supplied value must verify.

### Login limiter

All admin system auth and app member auth paths, plus app member reset-password, share a 10-request/minute tenant/peer-IP budget. Captcha get/check share a separate 30-request/minute budget. Lua INCR plus expiry is atomic and includes failed requests; switching auth endpoints does not replenish the budget. All auth methods are covered, including social, refresh, registration and SMS validation. OPTIONS and unrelated routes are bypassed.

Keys use `pkg/context.TenantID` and parsed `Request.RemoteAddr`, ignoring X-Forwarded-For and X-Real-IP. Missing tenant returns 403, malformed peer 400, oversized known body 413, exhausted budget 429 with Retry-After, Redis absence/error 503. Request bodies are capped at 16 KiB, including unknown-length streams read by handlers. Redis calls have bounded I/O timeouts and contexts; there is no memory fallback.

### SMS codes

Six-digit codes use crypto/rand, replacing math/rand four-digit codes. Active code keys include tenant, phone and scene. Atomic Redis reservations limit a tenant/phone across scenes to one send attempt per minute and ten per UTC calendar day. Keys have finite TTLs. A new reservation invalidates the older code in that scene.

The database code record must be written before delivery is attempted. A code becomes active only after delivery succeeds and Redis confirms activation under the same reservation. Failed DB writes, failed sends and activation failures propagate; none activate a new code. The production wrapper rejects disabled templates/channels and debug/unsupported channels, then requires a persisted successful SMS log with provider status OK before activation. Thus the existing provider adapter behavior of returning a rejection status with nil Go error cannot activate a code.

UseSmsCode atomically consumes the Redis value and then updates exactly the recorded ID under tenant, phone, scene, code and unused guards. Database update failure or zero affected rows rejects authentication; the code remains consumed. Wrong six-digit attempts also burn the code. ValidateSmsCode remains preflight-only. Missing context and unavailable dependencies fail closed.

## Verification

Initial reproduction: the new tenant-scoped Generate/Verify test failed on the old implementation with `captcha_security_test.go:22: redis: nil`, since only the legacy tenantless key was written. After the fix the test passes and concurrent acceptance is exactly one of 32 attempts.

Final command, run from the backend directory:

```sh
rtk proxy env GOCACHE=/private/tmp/ruoyi-ab-go-build GOPATH=/private/tmp/ruoyi-ab-gopath T09_REDIS_ADDR=127.0.0.1:56397 TEST_POSTGRES_DSN='postgres://macmini@127.0.0.1:55439/ruoyi_ab?sslmode=disable' go test -race ./internal/service/system ./internal/middleware ./internal/api/handler/admin/system -run 'TestCaptcha|TestSmsCode|TestLoginRateLimit|TestSecurityActualRedisServerOutage' -count=1 -v
```

All 10 tests pass, with no race reports or skipped selected tests. Raw output: `evidence/t09-login-security-race.log`.

- Captcha challenge and proof concurrent replay: exactly one of 32 accepted, tenant isolation, wrong/changed proofs, expiry, nil/failed cache.
- Actual dedicated redis-server is started on a private Unix socket, seeded, then killed; subsequent captcha/SMS calls fail closed within the asserted bound. The first fixture attempt hit the macOS Unix-socket path limit; shortening the fixture directory fixed it without changing application logic.
- Rate limiter: 40 concurrent requests spread over login/SMS/reset routes yield exactly 10 allowed and 30 limited despite forged forwarding headers; other tenants and peer IPs are independent. Captcha 30-request budget, expiry, missing tenant, invalid peer, 16 KiB known/chunked bodies, nil/failed cache and unaffected public routes are checked.
- SMS: real PostgreSQL in unique migrated schemas; tenant/scene isolation, exactly one of 32 concurrent uses, durable used-state, failed delivery, DB create/update errors, closed database, Redis failure, exactly one of 24 concurrent sends, ten-attempt daily budget across scenes, disabled/debug channels, wrong/expired code, and post-delivery Redis activation failure.
- HTTP captcha: exact frontend field shape, empty get body, successful check → consumable login proof, replay rejection, oversized request and generic outage response.

No tests send an SMS or contact external providers. SMS delivery is a local function stub except disabled/debug rejection tests, which use the real constructor with an unreachable provider factory.

## Environment and operational limits

Private shared Redis: `127.0.0.1:56397`, Redis 8.2.1, no password, loopback only, persistence disabled. Its PID/log/env files are in `/private/tmp/ruoyi-t09-redis/`; it is intentionally left running for parent/Epicurus acceptance. Tests do not flush shared Redis. DB0 is used here with tenant/UUID-isolated keys; Epicurus was advised to use DB1. PostgreSQL is the parent-provided localhost:55439 database ruoyi_ab. Go cache and GOPATH match the task instructions.

GETDEL requires Redis 6.2 or newer. Old tenantless captcha/SMS cache values are not accepted. Behind a reverse proxy the peer-IP limiter intentionally groups clients by the proxy address until an explicitly trusted proxy policy is introduced. The fixed budgets are implementation defaults, not dynamically configurable settings.

Failed SMS attempts consume send quota; users may need to wait after dependency failures. Successful delivery followed by Redis/DB failure can reject a legitimate attempt and require a new code; this is the fail-closed policy. Limits are per tenant/phone and tenant/IP, not a global distributed-bot or per-account lockout defense. Server transport read/write deadlines remain outside this workstream.

The initial plaintext SMS audit finding was resolved by the targeted follow-up documented in `t09-sms-audit-security.md`: new code audit rows store a nonsecret marker, authentication SMS logs are redacted on write and read, and raw send/debug diagnostics are removed. Historical stored data is not rewritten by a code deployment. Provider SDK transport behavior was not redesigned. No real Aliyun/Tencent delivery acceptance was performed. Other SMS sending endpoints remain under their existing implementation; the added confirmation guard applies to authentication codes. These limits must not be presented as full production SMS readiness or full T09/T10 acceptance.
