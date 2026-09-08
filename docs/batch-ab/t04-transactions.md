# T04 local transaction consistency — batch B

Scope: shared Go backend, local PostgreSQL order and product mutations. No frontend changes, commits, payment channels, real payments, or refunds were executed by this work.

## Implementation

- `internal/repo/transaction.go` is tracked, handwritten source outside ignored generated DAO files. `InTransaction` joins an explicitly propagated local transaction; `QueryFromContext` resolves it without mutating shared service state. Callers must propagate returned errors; transaction contexts must not escape to goroutines or remote payment calls.
- `TradeOrderUpdateService.CreateOrder` commits the order, items, conditional SKU decrement, SPU aggregate, coupon claim, point balance and ledger, local payment row, payment linkage, selected-cart cleanup, and creation log together. Critical handler failures now propagate. Removed unused alternative creation/payment helpers which retained the old unsafe boundary.
- SKU decrements retain the conditional stock predicate and affected-row check. Whole multi-SKU changes also form a transaction when called independently. SPU aggregate failures abort the same transaction.
- Coupon consumption is a tenant/user/status/validity conditional update with an affected-row check. Cancellation returns only the coupon held by that exact order.
- Point records lock and read the member inside the propagated transaction, validate bounds, change the balance, and write the ledger atomically. Order debits and unpaid cancellation credits use this service. The remaining bare point mutation method cannot overdraw or overflow int32 balances.
- Local payment creation resolves the same transaction and propagates lookup errors. Trade payment linkage checks the affected row. Payment application and trade configuration reads within creation also resolve the transaction.
- Order numbers use the already installed UUID library, encoded to 32 characters, instead of timestamp plus a six-digit random suffix.
- Empty/invalid quantities, nonzero unsupported marketing activity IDs, and zero-price orders are rejected before creating payable orders.
- Member cancellation and system timeout cancellation share an unpaid-order state compare-and-set. Only the winner restores resources; an error in any return or cancellation-log insertion rolls back the state and all resources.
- Parent-requested SPU CRUD extension: SPU create/update/delete/status use the same helper; SKU insert/update/delete join it. SKU inserts remain batched; existing SKU rows use explicit SPU/ID-scoped updates, with selected writable columns so zero values persist while audit/sales fields remain untouched. Foreign SKU IDs are rejected. SPU selected updates likewise persist empty/zero/false values. Removed SKUs are soft-deleted in the same transaction. No upsert exemption was added to tenant security.
- Database-agent request: member CSV tag membership now uses PostgreSQL `ANY(string_to_array(...))`, with exact token regression checks.

## Evidence

Final verification: PASS on the final local source with real PostgreSQL and `go test -race` for trade/product/promotion/member/pay/repo. `git diff --check` also passed. Full repository generation/build/HTTP smoke remains parent-owned.

Tests use PostgreSQL 16 at the isolated local port 55439, with a fresh schema and `search_path` on every pooled connection. `internal/testutil.PostgreSQL` applies the database agent's versioned migrations, then tests enable the actual tenant plugin. Assertions inspect real persisted rows, not only HTTP responses. A fixed test discount calculator isolates resource consistency; real SKU/SPU reads, order orchestration, stock/coupon/member services and local payment service execute. No payment client factory or channels are instantiated.

`internal/service/mall/trade/order_transaction_test.go` covers:

- Main CreateOrder: late SKU shortage after pricing, expired coupon, insufficient points, injected point-ledger write failure, local payment insert failure, payment linkage failure, cart deletion failure, order-log insertion failure, and a PostgreSQL deferred constraint-trigger failure at COMMIT.
- Concurrent last-stock, coupon and point claims: both requests complete pricing before competing, exactly one succeeds. Coupon and point contenders use different SKU and SPU rows so stock locks do not mask their concurrency.
- Assertions check SKU stock/sales, every SPU aggregate, coupon status/order/timestamp, balance and ledger totals/business order IDs, order count, all items, local payment count/link/amount/status, cart deletion state, exact creation/cancellation log counts and ownership, and absence of payment extensions/notification tasks/refunds.
- Unpaid cancellation: late ledger/log failure restores the original unpaid state and all resources; concurrent duplicate cancellation produces one winner; repeated system cancellation cannot return again.
- Real subprocess SIGKILL after local payment INSERT and before COMMIT. No Go cleanup runs; all local resources are rolled back when PostgreSQL loses the connection.
- Zero-price rejection, unsupported input, exact CSV tag membership, and successful order creation with one available database connection.

`internal/service/mall/product/transaction_test.go` covers:

- Actual SQL trigger rejects the second SKU of a batch, and a deferred SPU trigger rejects COMMIT; no SPU/SKU residue survives.
- Second SKU update failure restores the exact pre-call SPU/SKU snapshots.
- Numeric zero, empty strings/lists, and false fields persist through real service updates.
- Updating/adding/removing SKUs is atomic; a late deletion error restores all rows.
- Failed product deletion restores both SPU and SKU rows; successful deletion marks all children deleted.
- Cross-tenant SKU IDs cannot mutate or attach another tenant's row; own product changes roll back.
- Nested product creation followed by update joins the caller transaction and its rollback leaves no SPU or SKU rows.

Evidence files:

- `evidence/t04-postgres.log`: final verbose order and product integration tests.
- `evidence/t04-product-postgres.log`: earlier focused product integration run; the combined log includes the final nested-transaction coverage.
- `evidence/t04-focused-race.log`: complete relevant package tests with Go race detection.
- `evidence/t04-original-boundary-fails.log`: counterfactual run replacing only `order_update.go` with the original Git HEAD file via `go test -overlay`, keeping fixed infrastructure and other services. The original boundary returns two successes under contention and leaves a local payment row after link/commit failure. This isolates the boundary defect; it is not a claim that untouched original source supported PostgreSQL.

Re-run (shell commands prefixed with rtk):

```sh
rtk proxy env GOCACHE=/private/tmp/ruoyi-ab-go-build GOPATH=/private/tmp/ruoyi-ab-gopath TEST_POSTGRES_DSN='host=127.0.0.1 port=55439 user=macmini dbname=ruoyi_ab sslmode=disable' go test -race ./internal/service/mall/trade ./internal/service/mall/product ./internal/service/mall/promotion ./internal/service/member ./internal/service/pay ./internal/repo -count=1
```

Without `TEST_POSTGRES_DSN`, integration tests explicitly skip; a skipped run is not acceptance evidence. The subprocess interruption test currently requires the documented key/value DSN form.

## Limits and coordination

- This proves local creation consistency and the tested unpaid cancellation/resource-return behavior. It does not certify the batch-C channel lifecycle, cancellation versus remote payment/callback races, paid cancellation/refunds, partial after-sale returns, duplicate payment notification delivery, or restart recovery after external channel requests. The existing local payment row remains waiting after unpaid cancellation; closing/reconciling it belongs to T07 and must be handled before live payment use.
- Legacy paid/refund and generic handler-dispatch paths were not converted wholesale. They still require batch-C transaction/state-machine review; do not infer correctness from the new helper's existence.
- All creation-side effects are local transactional writes, including cart cleanup and the order log; no best-effort post-commit branch or retry queue remains. The caller can retry a failed creation after rollback. Network-level request idempotency after a committed-but-lost response is not introduced by this change.
- No migration/schema/security ownership was taken. Zeno supplied migrations and fixed real PG blockers uncovered by these tests (duplicate generated columns, BitBool zero defaults, and migration connection ownership). Epicurus supplied tenant enforcement and payment-client boundary changes. Parent owns whole-repository generation/build/HTTP smoke.
- Marketing price formulas are not validated by the fixed-discount transaction fixtures. External merchant credentials and callback delivery were not needed or used.

## Final parent acceptance supplement

`TestOrderTransactionUnavailableProduct` uses the actual pricing service SKU/SPU validation and proves initial out-of-stock, off-shelf and recycled products cannot create an order or consume local resources. All three PostgreSQL cases passed in `evidence/unavailable-product.log`; they also run in the final full-suite check. Final commits and consolidated acceptance supersede workstream-local status wording above; see verification-report.md.
