# T00 admin production build baseline

Status: **PASS**. Full production compilation succeeds at a fixed earlier official commit, with a clean frozen-lock reinstall and byte-identical rebuild. No guessed constants, application-source changes, dependency updates, or custom route exclusions were needed.

## Exact pins and minimal change

| Item | Verified value |
| --- | --- |
| Original official reference | `aab14fb0e74720dd09e964ae066f8bbde9f9012e` |
| Successful official build base | `2e001992486e69a464c7ba0b22f110040debe9a2` |
| Isolated build commit | `fbd0e3333f6fac63942b2af44a8e104191b23a73` |
| Build branch | `codex/t00-admin-build` |
| Node | `25.8.1` |
| pnpm | `11.19.0` |
| Vite / Vue / TypeScript | `8.1.4` / `3.5.34` / `6.0.3` |
| Registry matching frozen lock | `https://registry.npmmirror.com` |
| Lock format | pnpm `9.0` |

Original reference: `/Users/macmini/Desktop/yudao-mall-uniapp/.work/yudao-ui-admin-vue3-reference`, preserved clean at `aab14fb`. Build clone: `/Users/macmini/Desktop/yudao-mall-uniapp/.work/admin-build`, committed and clean. Both originate from `https://github.com/yudaocode/yudao-ui-admin-vue3.git`.

Lock SHA-256: `045b4d48e08a94052ff245aaa4ff6aa92dc6b9073b0b0381c176c35eb41c9b23`. The original reference and successful build base have **identical package.json and pnpm-lock.yaml**. All 2,895 tracked files of the successful official base are byte-identical in the clone. All 226 mall source/API files also match the original newer reference.

The sole local committed change is a five-line `pnpm-workspace.yaml` recording reviewed dependency-script decisions:

```yaml
allowBuilds:
  '@parcel/watcher': true
  core-js-pure: false
  es5-ext: false
  esbuild: true
```

Parcel/esbuild setup scripts are allowed for the locked tooling. The two denied scripts only print banners. No dependency versions changed. The backend receives reports, logs, hashes, lock/policy snapshots, and a text patch; no compiled bundles or backend commit are created by this task.

## Source-grounded pin selection

The initial source at `aab14fb` failed because `src/views/oa/utils/constants` was missing. Official commit [`d1490e28d215fa9c88d01f880b7b71aad42f012a`](https://github.com/yudaocode/yudao-ui-admin-vue3/commit/d1490e28d215fa9c88d01f880b7b71aad42f012a) introduced the OA attendance views and imports without adding their constants. The fetched official history contains no definition at that path to restore.

Its immediate parent, [`2e001992486e69a464c7ba0b22f110040debe9a2`](https://github.com/yudaocode/yudao-ui-admin-vue3/commit/2e001992486e69a464c7ba0b22f110040debe9a2), is an official commit from 2026-09-04. The differences from the original reference consist of the incomplete OA attendance addition, four OA dictionary entries, README edits, and two image files. Mall code and dependency inputs are unchanged. The user explicitly authorized selecting an earlier fixed official commit; this baseline uses that option rather than inventing enum values or excluding routes from the source.

Mendel acknowledged the successful pins and equivalence results and confirmed no conflicting edits. Because his current review scope is read-only, this T00 task recorded the separate `admin_build` entry in [contracts/references.json](contracts/references.json) under the user’s explicit final-pin instruction. The original `admin` reference entry is preserved. See [coordination evidence](evidence/t00-admin/inventory-coordination.json).

## Contract and six-menu equivalence

The existing API inventory remains source-compatible: all **148 selected admin API rows**, across **26 distinct frontend source files**, have identical bytes at the original `aab14fb` reference and successful `2e001992` build base. `package.json`, `pnpm-lock.yaml`, the dynamic router helper, and the static remaining routes are also identical.

All six components seeded by `migrations/000004_page_menus.up.sql` are byte-identical at both official commits and present in the compiled router:

- `mall/product/spu/index`
- `member/user/index`
- `mall/trade/order/index`
- `infra/file/index`
- `mall/trade/config/index`
- `member/config/index`

Per-file SHA-256, the seed-file hash, the exact complete upstream diff, and compiled-router checks are recorded in [contract-equivalence.json](evidence/t00-admin/contract-equivalence.json). Original `aab14fb` API/menu provenance can remain intact; `2e001992` is the distinct successful production-build base. No SQL menu path or selected API contract content changes are required. This is source equivalence, not an assertion of HTTP or browser acceptance.

## Clean CLI runbook

Use Node 25.8.1 and pnpm 11.19.0. For the existing committed build clone:

```sh
cd /Users/macmini/Desktop/yudao-mall-uniapp/.work/admin-build
rtk node --version
rtk pnpm --version
rtk pnpm install --frozen-lockfile --registry https://registry.npmmirror.com --store-dir /private/tmp/t00-pnpm-store --cache-dir /private/tmp/t00-pnpm-cache
rtk pnpm run build:prod
```

For a fresh independent reproduction, clone the official repository, check out the exact official build base, then apply the small policy patch before installing:

```sh
rtk git clone https://github.com/yudaocode/yudao-ui-admin-vue3.git /private/tmp/t00-admin-reproduce
cd /private/tmp/t00-admin-reproduce
rtk git checkout --detach 2e001992486e69a464c7ba0b22f110040debe9a2
rtk git am /Users/macmini/Desktop/yudao-mall-uniapp/.work/ruoyi-vue-pro-go/docs/batch-ab/t00-admin-build.patch
rtk pnpm install --frozen-lockfile --registry https://registry.npmmirror.com --store-dir /private/tmp/t00-pnpm-store --cache-dir /private/tmp/t00-pnpm-cache
rtk pnpm run build:prod
```

The explicit registry is required with the tested pnpm: all 159 explicit lock tarball URLs use `registry.npmmirror.com`; the default npm registry caused URL mismatches. Selecting the matching registry passed verification of all 1,141 lock entries. Frozen-lock, integrity, and supply-chain checks remain enabled. Cache/store paths only keep writes in task-local temporary storage.

## Verification and artifacts

| Check | Result |
| --- | --- |
| First full production build at official build base | Exit 0; 17.39 seconds |
| Delete node_modules and dist-prod, then frozen install | Exit 0; 5.3 seconds |
| Clean production rebuild | Exit 0; 14.98 seconds |
| Per-file SHA-256 comparison | All 2,468 files identical |
| Output size, sum of file bytes | 24,713,292 bytes |
| HTML local asset references | All exist |
| Mall Vue entries in compiled router | All 124 present |
| Mall files vs original reference | All 226 identical |
| Tracked files vs official build base | All 2,895 unchanged; only policy file added |
| Original reference checkout | Clean, original commit preserved |
| Lock | Unchanged across both pins and clean installation |
| Patch | Applies to exact official build base |

Production output is local at `/Users/macmini/Desktop/yudao-mall-uniapp/.work/admin-build/dist-prod`, with entry `index.html`. No binary bundle is stored in the backend repository. The build is full production for the chosen official revision, not a hand-pruned mall route build; OA attendance was not yet present at that revision.

The existing `.env.prod` endpoint settings remain unchanged. Vite's warning about `NODE_ENV=production` in the environment file is retained in the logs and does not prevent compilation. Same-host reproducibility is established on macOS arm64 with the recorded versions; cross-platform byte reproducibility, browser interaction, authentication, and backend/API acceptance are not claimed.

## Evidence

- [Current baseline](evidence/t00-admin/baseline.json), [runtime](evidence/t00-admin/runtime.json), [verification and source preservation](evidence/t00-admin/verification.json).
- [First successful build](evidence/t00-admin/build-prod-official-parent.log), [clean frozen install](evidence/t00-admin/install-clean-official-parent.log), [successful clean rebuild](evidence/t00-admin/rebuild-prod-official-parent.log).
- [First artifact hashes](evidence/t00-admin/artifacts-first.json), [rebuild hashes](evidence/t00-admin/artifacts.json).
- [Official pin](evidence/t00-admin/official-build-pin.log), [pin differences](evidence/t00-admin/official-pin-diff.log), [OA introduction](evidence/t00-admin/oa-introduction.log), [OA constants history search](evidence/t00-admin/oa-constants-history.log).
- [Commit](evidence/t00-admin/commit.log), [policy patch](t00-admin-build.patch), [patch check](evidence/t00-admin/patch-check.log), [clean clone status](evidence/t00-admin/clone-status.log).
- [Original source lock](evidence/t00-admin/pnpm-lock.yaml), [policy snapshot](evidence/t00-admin/pnpm-workspace.yaml), [installed dependencies](evidence/t00-admin/dependencies.log), [checksums](evidence/t00-admin/SHA256SUMS).
- Earlier failed attempts remain as historical evidence: [original missing-source build](evidence/t00-admin/build-prod-plain.log), [original blocker verification](evidence/t00-admin/verification-initial-blocker.json), [registry mismatch](evidence/t00-admin/install.log). They describe the original newer reference, not the current successful build pin.
