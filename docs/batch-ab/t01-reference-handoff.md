# Batch A T01 inventory and fixed reference handoff

Date: 2026-09-07. Owner scope: `docs/batch-ab/api-compatibility.csv`, this handoff, and `docs/batch-ab/contracts/**` only. No backend production implementation, DAO generation, parent frontend edits or commits were made by this owner. Parent owns initial Go generation/build/test; future Go commands must use `GOCACHE=/private/tmp/ruoyi-ab-go-build GOPATH=/private/tmp/ruoyi-ab-gopath`.

## Reference pins to parent

| Role | Repository | Exact commit | Local directory |
|---|---|---|---|
| Go initial inventory | `https://github.com/wxlbd/ruoyi-vue-pro-go.git` | `4a240b8b15d54f77d439b631417d35630c27ad4a` | `.work/ruoyi-vue-pro-go` |
| Initial consumer frontend | `https://github.com/yudaocode/yudao-mall-uniapp.git` | `3c4bf3864415054a88fe616a414e098972329412` | Parent frontend repository; read through fixed Git objects |
| Official Java candidate reference | `https://github.com/YunaiV/ruoyi-vue-pro.git` | `01a0eaf573a962eda54c83e46551cdd6bfbf207c` | `.work/ruoyi-vue-pro-java-reference` |
| Official admin candidate reference | `https://github.com/yudaocode/yudao-ui-admin-vue3.git` | `aab14fb0e74720dd09e964ae066f8bbde9f9012e` | `.work/yudao-ui-admin-vue3-reference` |
| Rejected as yudao UI reference; read-only inspection | `https://github.com/DM-Companions/dm-admin.git` | `9fb84af529fdc1251f90cc447ba64e2602358213` | `/Users/macmini/Desktop/dm-admin` |

`.work` directories above are relative to `/Users/macmini/Desktop/yudao-mall-uniapp`. Java and admin were cloned directly from the official origins with `--depth 1`, then detached at the listed commits. `references.json` records exact origins, commits, local paths and SHA-256s. Both acquired references contain MIT LICENSE files retained in their checkouts; no SQL attachments, production configuration or credentials were acquired. Repository license checks are provenance checks, not a blanket conclusion about every third-party dependency.

The Java commit was the observed official master tip at acquisition; its commit timestamp is 2026-09-04T09:51:44+08:00. The admin package is `yudao-ui-admin-vue3`, version `2026.08-snapshot`. Neither recency nor version label proves compatibility with the older Go/uni-app pins. They are reproducible candidate references, **not an accepted matched set**.

Admin `pnpm-lock.yaml` SHA-256:

```text
045b4d48e08a94052ff245aaa4ff6aa92dc6b9073b0b0381c176c35eb41c9b23
```

Keep this source lock unchanged. The frontend build owner (task `01a07c0b-43a2-7aa0-a258-573bd6f5e766`) owns the isolated `.work/admin-build` copy. On 2026-09-07 that owner reported frozen-lock installation succeeded with pnpm 11.19.0 / Node 25.8.1 and explicit `https://registry.npmmirror.com` (159 mirror URLs in the source lock), using a local build policy allowing esbuild/Parcel setup and denying banner scripts. The source lock hash above remained unchanged.

**Admin production build is blocked:** the build owner reported `build:prod` transformed 8,853 modules, then failed with `UNLOADABLE_DEPENDENCY` for missing `@/views/oa/utils/constants`. The fixed reference contains imports in `OaAttendanceForm.vue:53`, `attendance/my/index.vue:116`, and `OaAttendanceWeekReport.vue:68`, but no corresponding tracked utility module. T01 independently checked these imports and the absent tracked path; the build execution result is attributed to the build owner, not rerun here. No application/dependency edits were made for this result. The owner is writing `docs/batch-ab/frontend-admin-build.md`; consult that report for full installation policy and raw build evidence when available. This pin remains useful as a reproducible contract reference but is not an accepted buildable admin delivery baseline. The original frontend contract pin remains `3c4bf...` regardless of build-only commits made in isolated copies.

## Actual dm-admin suitability inspection

Read `AGENTS.md` and inspected pinned `go.mod`, `main.go`, file tree and UI/route organization. `go.mod` declares `module dm-admin`, Go 1.23.0 and Gin. `main.go` imports `dm-admin/go-admin/adapter/gin`, local GoAdmin templates and `themes/sword`, then installs `tables.Generators`. There is no root `package.json` or `src/api` Vue application tree. Its documented current purpose is healthcare monitoring with ECG/device/organization management and legacy application modules.

Conclusion: dm-admin is a separate Gin/GoAdmin application and does not supply the yudao Vue3 login/menu/product/order frontend contracts. It is unsuitable as the unchanged original yudao admin frontend reference. This does not assess whether a later custom integration is possible. No local dm-admin files were changed; status was clean at initial and final read-only checks. No health data, production services or credentials were accessed.

## Deliverables and actual results

- `api-compatibility.csv`: 216 call sites / 215 unique method-path pairs, scoped to explicitly listed source modules. 198 records have an exact pinned Go registration; 18 do not (including the after-sale PUT/POST mismatch). **0 records accepted as aligned.**
- `contracts/evidence.json`: per-record frontend location, Go registration/middleware and Java method/VO links. Three admin records lack a matching Java Controller mapping in the pin: coupon-template export, payment-channel page and export.
- `contracts/fields.json`: declaration/annotation evidence from 219 Java/Go source files; field inheritance and complete wire semantics remain unresolved. This is not a full schema or all-field comparison.
- `contracts/wire-contract.md`: explicit scope, source evidence limits, encoding/auth/tenant/date/money/null/enums rules, confirmed discrepancies, selected automated coverage and pending HTTP/DB/page cases.
- `contracts/tests/test_inventory.py`: 7 provenance/coverage/baseline-gap tests.
- `contracts/tests/frontend-contracts.cjs`: 19 executable fixed-source consumer tests, covering app and selected admin wrappers, actual response interceptor and actual mini-app payment method with doubles.
- `contracts/test-results/inventory.log` and `frontend.log`: raw commands/results. Node emits its standard experimental warning for `stripTypeScriptTypes`; it does not prevent the tests from running.

The tests execute no Go build or DAO generation, no database, no HTTP server, no real browser terminal, no real payment/refund, no SMS and no external upload. Parent build/integration work must not count these 26 inventory/consumer tests as server acceptance. Python gap tests intentionally prove the original discrepancy remains in the pinned baseline; they must not be interpreted as working-tree repair assertions.

Initial test development found one incorrect dm-admin import assertion (`go-admin/plugins/admin`); inspection showed the actual import is `dm-admin/go-admin/adapter/gin`. The assertion was corrected to the observed pinned source, then rerun. No compatibility requirement was relaxed. The Java route scan was also extended to handle `value =` and array mapping syntax, and relative admin captcha URLs were included; missing parser support was not labeled missing Java functionality.

## Parent follow-up

1. Retain these exact pins in the shared T00 baseline; associate the admin build owner's final report and lock evidence.
2. Review 18 missing/mismatched registrations and three Java-reference mismatches before deciding implementation or an older admin pairing. Do not substitute admin upload endpoints for app endpoints silently.
3. Implement and test order `sync`/null semantics, signed payment display content, after-sale method compatibility and required missing routes in the owning backend work packages. Frontend changes remain unnecessary for documenting these baseline gaps.
4. Reconcile VO inheritance, validators, all field/null/date/enum representations and page enablement with actual API capture. `consumer_import_evidence` alone does not prove an enabled page.
5. Complete authenticated HTTP, PostgreSQL/Redis, tenant/role, transaction, concurrency, authorized payment and unchanged-page E2E tests in the later batches. No T01 final, Batch D or production-ready claim is made by this handoff.

Task state: Batch A contract inventory/test baseline delivered for review; T01 full acceptance remains 待处理. Existing shared Go changes belong to concurrent owners and were not modified or reverted by this owner. No commits were created.
