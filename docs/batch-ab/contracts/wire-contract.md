# T01 Batch A contract inventory

This is the Batch A inventory and test baseline, **not T01 final acceptance or Batch D E2E**. All Go evidence is read from `4a240b8b15d54f77d439b631417d35630c27ad4a` using `git show`, even while other owners change the shared checkout. Java and frontend evidence likewise use the commits in `references.json`. No frontend business code has been changed.

## Scope and interpretation

`../api-compatibility.csv` inventories 216 literal API call sites (215 distinct method/path pairs; settlement-product has two client wrappers). It covers the selected API modules in `inventory.py`, not every API in either frontend. Candidate scope includes member authentication/profile/address, product/SPU/SKU detail, category, cart, settlement, order, payment order/channel, express and pickup, after-sale, file upload, coupon and point records; admin login/captcha/permissions/menu, product management, delivery and pickup, after-sale, member, payment configuration and file management.

Import locations are evidence that a view/component uses an API module, **not proof that every exported function is called or the page is enabled**. Runtime admin menus, tenant configuration, DIY pages, feature switches and terminal compile branches remain unresolved. No row is marked `不启用` without server-side disable evidence. Auxiliary calls included by a selected module (e.g. OAuth authorization or product history) do not expand the accepted first-phase scope. Seckill, combination, bargain, brokerage, wallet recharge and live streaming are outside this inventory; this does not prove they are disabled.

The CSV vocabulary is interpreted narrowly:

- `部分对齐`: the pinned Go router declares the exact method/path, or a separately stated static behavioral gap exists. It does **not** mean DTOs or business behavior passed.
- `未实现`: no exact method/path registration was found in the pinned router files, including a method mismatch. Handler/service code may still exist. Search/runtime verification is required before implementation.
- `已对齐`: deliberately unused; no HTTP/server acceptance has run.
- `不启用`: deliberately unused; no server disable evidence has been established.

`evidence.json` links each CSV row to its pinned frontend location, Go registration/group middleware, Java Controller method source and referenced VO files. `fields.json` retains Java declarations/annotations and Go contract field/tag declarations. These are source evidence, **not generated JSON Schema**: inherited Java fields, validation groups, nested types, custom serializers and service constraints require manual reconciliation. Same-named Java VO matches need package verification. The lightweight route extractor handles literal group registrations and Java single/value/array mappings; it is not a Go AST or Spring runtime route dump. Dynamic URL paths and calls outside selected modules require a later discovery pass.

## Wire rules established from fixed sources

| Dimension | Established observation | Remaining validation |
|---|---|---|
| Prefix | Go app `/app-api`, admin `/admin-api`; Java `WebProperties` uses those defaults. uni-app uses `baseUrl + apiPath`, with `apiPath` from `SHOPRO_API_PATH`. | Confirm actual isolated runtime configuration and reverse proxy; no hard-coded production host. |
| App auth | `sheep/request/index.js` sends stored token as `Authorization`; refresh retry adds `Bearer `; file upload also adds `Bearer `. `sheep/store/user.js` stores returned accessToken unchanged. | Capture login response/token format and first request vs refresh; do not silently normalize this inconsistency. |
| Admin auth | Axios prepends `Bearer `; login/refresh are allowlisted. `tenant-id` is conditional on tenant config; `visit-tenant-id` can accompany authorized admin requests. | Missing/expired/wrong token type, refresh replay/revocation and actual permission/tenant denial must be tested against server. |
| App tenant/terminal | Request interceptor adds `tenant-id` and `terminal`; upload adds tenant but not terminal. | Missing tenant, cross-tenant and async isolation remain Batch B/D tests. |
| Envelope/errors | Client expects numeric `code`, `msg`, `data`; code 0 is success, 401 triggers refresh. Go `pkg/response/response.go` uses HTTP 200 for success/business errors. | Middleware HTTP statuses and complete Java business-code mapping are not established by a route match. |
| Captcha exception | Fixed Java admin captcha returns Anji `ResponseModel`, and fixed admin calls `postOriginal` with relative `system/captcha/get` and `system/captcha/check`. | Never mechanically wrap captcha into CommonResult; validate its separate response/error fields. |
| Request binding | Java method signature and `@RequestBody` vs `@RequestParam`/model binding are in evidence. Upload uses multipart field `file`, optional `directory`. | Required vs missing/null/empty, validation groups and Go handler binding need individual tests. Multipart boundary is terminal-generated. |
| Arrays | Settlement encodes indexed keys `items[0].skuId/count/cartId`; cart/SPU arrays reach request wrapper as arrays. Admin Axios uses `qs.stringify(params, {allowDots:true})`; some wrappers join IDs with commas. | Node tests capture wrapper input, not luch-request/qs final network encoding. Do not claim one universal CSV/brackets/repeated-key policy. |
| Pagination | `pageNo/pageSize`, `data.list/data.total` are the reviewed common names; Java method/VO and Go tags retained. | Defaults, maximum sizes, null vs empty list and total precision are endpoint-specific. |
| Amount | Preserve integer fen for fields explicitly documented in 分; `refundPrice:1` remains 1 in wrapper tests. | Field ranges, pricing, allocation, rounding and refund conservation require actual Go/DB tests. No float conversion policy added. |
| Date/time | Java `TimestampLocalDateTimeSerializer` emits epoch milliseconds using `ZoneId.systemDefault()`. Go app trade DTO uses `types.JsonDateTime`; its nonzero serialization is milliseconds and zero is null. Admin `TradeOrderBase` still has `time.Time` and pointers. | Do not generalize all Go dates as incompatible: inspect the actual DTO. Fix runtime timezone explicitly and test offset/zero/null per endpoint. |
| Null/missing/zero | Actual app response interceptor preserves `null`, `[]`, missing `data`, 0 and false; contract tests exercise these separately. | Server response shapes still need endpoint fixtures; never globally normalize null to empty array. |
| IDs/enums | Java Long, Go int64 and TypeScript number declarations remain recorded. JS numeric 9007199254740993 rounds to 9007199254740992; a string ID passes through the wrapper unchanged. | Decide supported wire representation with server/client evidence; do not invent a global stringify rule. All status/channel/delivery enums require full field verification. |
| Side effects | Java method source records service calls; Go registration is not service behavior evidence. | Ownership, stock/coupon/point/payment transaction consistency, fulfillment and refunds are not covered by source inventory tests. |

## Reproduced consumer gap and static discrepancies

1. **Payment:** fixed Go `internal/service/pay/client/weixin/client.go` writes `DisplayContent: *resp.PrepayId`; fixed mini-app `wechatMiniProgramPay()` calls `JSON.parse(data.displayContent)`, expects string `timeStamp`, `nonceStr`, `packageValue`, `signType`, `paySign`, and maps `packageValue` to terminal `package`. The test executes that exact client method: synthetic complete object reaches the terminal double; bare prepay ID throws `SyntaxError`; business failure never invokes payment. This demonstrates consumer rejection, not a real WeChat call, signed-parameter correctness or a server fix.
2. **Order detail:** fixed Java `AppTradeOrderController.getOrderDetail` returns `success(null)` for absent/wrong-owner order and supports optional `sync` with a waiting-payment state check. Fixed Go `GetOrderDetail` ignores sync and responds with `ErrNotFound` for nil/wrong-owner order. Tests verify the fixed-source discrepancy; no HTTP reproduction or trusted payment synchronization was executed.
3. **After-sale delivery:** both fixed frontend and Java use PUT `/app-api/trade/after-sale/delivery`; Go registers POST. This is an exact static method mismatch. Mutation route semantics remain untested.
4. **Missing app routes:** the pinned router scan does not register app order delete/item comment, after-sale log list, express list/pickup list/get, or app file upload/presigned-url/create. The admin upload registration exists and must not be substituted for the missing app path without a documented decision.
5. **Reference drift:** three selected admin calls have no mapping in the pinned Java scan: coupon-template export-excel, pay/channel page, pay/channel export-excel. The latter two are also absent from Go. No assumed compatibility based on dates/version strings. Review whether to pick an older admin reference or implement approved scope; current references are candidates, not an accepted deployment pair.

## Tests and deferred cases

Run from the Go repository (Python standard library and Node v25.8.1 used locally; Node must expose `node:module.stripTypeScriptTypes`):

```sh
rtk proxy python3 -B docs/batch-ab/contracts/inventory.py
rtk proxy python3 -B -m unittest discover -s docs/batch-ab/contracts/tests -p 'test_*.py' -v
rtk proxy node --test docs/batch-ab/contracts/tests/frontend-contracts.cjs
```

The generator reads pinned Git objects and overwrites only the CSV, evidence.json and fields.json. It requires references at paths in references.json; relocate those paths explicitly when handing off to another host. No npm install, Go generation, database, browser, account, SMS or payment call is performed.

Python tests check provenance, hashes, required scope, route claims, source anchors and the known baseline differences. Node tests execute actual fixed app functions and selected admin TypeScript wrappers (types stripped using Node standard library) with request/terminal doubles. The app response interceptor is executed with doubles as well. Passing source-gap assertions means the known baseline discrepancy is still documented, **not that the discrepancy is repaired**. Keep these historical checks separate from future tests against the evolving working tree.

| Case area | Automated in this batch | Required follow-up; status 待处理 |
|---|---|---|
| Login/refresh/logout | App request body/query and mini-app fields; admin login/logout wrapper | HTTP auth errors, token format, expiry/type/replay, Redis fault, role/tenant boundaries |
| Menu/permission | Actual admin permission-info and simple-list paths | Real login with restricted role, menu/permission shape and forbidden actions |
| Product/SKU | ID/list/page wrapper, Java/Go field evidence | SKU arrays/nullable properties, missing/zero prices, missing SPU, Java Long boundary |
| Address/cart | JSON vs query, selected=false, IDs array | Ownership, empty list shape, invalid IDs/count and terminal wire array serialization |
| Settlement/create | Indexed two-item query, optional IDs omitted, false kept, empty items serialized, create JSON | Actual Go bind/validation including negatives, absent vs false pointStatus, freight/self-pickup requirements, coupon/point and stock consistency |
| Payment | channelExtras forwarding; actual mini-app display parser positive/negative/business-error | Factory channel, openid identity binding, signature correctness, disabled/cross-tenant channel, actual authorized terminal payment |
| Order detail | sync true/false/omitted frontend; Go/Java static null+sync discrepancy | HTTP missing/wrong owner null semantics, pay sync state/retry/ownership/rate behavior |
| Delivery/pickup | Actual admin delivery JSON, ID/code query; app lookup | HTTP role/tenant, state transitions, pickup code deduplication and delivery validation |
| After-sale | Create body, delivery PUT, cancel DELETE, admin refund PUT/disagree body | Refund amount/type/enums, ownership, audit state, partial/concurrent refunds and stock restoration |
| Files | Actual app multipart config including header and directory; route evidence | Actual upload bytes/error payload, file ownership and tenant isolation; no external object storage touched |
| Shared representation | Actual interceptor distinguishes null/empty/missing/0/false; JS safe integer demonstration | Endpoint date epoch/timezone/zero, list pagination, numeric enum mapping, HTTP status vs business code fixtures |

The deferred column is the explicit coverage debt for T01/T12 completion. Nothing here claims Batch D acceptance, migration/transaction integration or production readiness.
