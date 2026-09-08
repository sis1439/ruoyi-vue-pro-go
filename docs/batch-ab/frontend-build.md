# T00 — reproducible frontend CLI build baseline

Status: **PASS** for dependency installation and production compilation of H5 and mp-weixin. Verified on macOS arm64; this is a build baseline, not browser, backend integration, or WeChat release acceptance.

## Source and isolation

- Original: `/Users/macmini/Desktop/yudao-mall-uniapp`, commit `3c4bf3864415054a88fe616a414e098972329412`.
- Isolated clone: `/Users/macmini/Desktop/yudao-mall-uniapp/.work/uniapp-build`.
- Branch: `codex/t00-frontend-build`.
- Verified commit: `51dbba93527830846618c9b319f6163de4d11ba5`.
- Created with `git clone --no-hardlinks --no-checkout` and checked out the exact base. Only tracked Git content was cloned; `.work`, local dependencies, and untracked handoff files were not copied.
- The original checkout has no tracked diff. The isolated clone is committed and clean. No backend commit was created.

## Exact toolchain and lock

| Component | Fixed version |
| --- | --- |
| Node.js | 25.8.1 |
| npm | 11.12.1 |
| DCloud compiler/platform packages | 3.0.0-5020420260813003 |
| Vite | 5.2.8 |
| Vue | 3.5.11 |
| Sass | 1.77.8 |
| DCloud types | 3.4.31 |
| Lock format | npm package-lock v3 |

The matched DCloud packages are `@dcloudio/uni-app`, `uni-components`, `uni-h5`, `uni-mp-weixin`, `uni-cli-shared`, and `vite-plugin-uni`. Existing direct application and development dependencies were pinned to the lower-bound versions already declared in the source package. No alternative package manager lock was introduced. The complete transitive tree is locked with registry URLs and integrity hashes.

Official provenance: [DCloud CLI guide](https://uniapp.dcloud.net.cn/quickstart-cli), [DCloud CLI capabilities](https://uniapp.dcloud.net.cn/worktile/CLI.html), and [official stable Vue/Vite preset at immutable commit 6085e203](https://github.com/dcloudio/uni-preset-vue/blob/6085e2034de05a4aff527687cbfe517bd0855b63/package.json). The preset supplies the exact DCloud release and Vite pairing; its package file is archived in the evidence. Vue 3.5.11 is preserved from this project's original declared range. HBuilderX is not used or required by these commands.

Lock SHA-256: `36a1f5372912db488c37d59fb26ce54860248ed425d7f9f966deaf19c41c95fb`.

`.node-version`, exact package engines, `packageManager`, and `.npmrc` record/enforce the runtime and registry. The build scripts use POSIX shell syntax, tested on this macOS host; Windows CMD/PowerShell portability is not claimed. Node 25.8.1 is the tested host runtime, not a recommendation to deploy a production Node service on that release.

## Build-only changes

1. `package.json`: add two production build scripts, declare missing official tooling/Vite/Sass/types, pin all direct versions, and record runtime versions.
2. `package-lock.json`: commit the resolved dependency graph; `.gitignore` now allows it and excludes generated `dist/`.
3. `.node-version` and `.npmrc`: runtime, exact saving, strict engines, official npm registry.
4. `vite.config.js`: correct Vite's config callback to `({ mode })` so `loadEnv` receives the production mode.
5. `sheep/libs/mplive-manifest-plugin.js`: this file is a Vite build helper, not page business logic. Skip rewriting the manifest when the requested live-plugin configuration is already present. The existing enable/disable behavior remains intact. Five isolated helper checks cover no-op, enabling, and disabling.

All 584 tracked baseline files were compared byte-for-byte. Only the four pre-existing build/package files listed above changed; three build/package files were added. Pages, router source, components, application logic, `.env`, `pages.json`, and `manifest.json` are unchanged.

## Reproduce

Use the recorded Node and npm versions. From the isolated clone:

```sh
cd /Users/macmini/Desktop/yudao-mall-uniapp/.work/uniapp-build
rtk node --version
rtk npm --version
rtk npm --cache /private/tmp/t00-npm-cache ci --no-audit --no-fund
rtk npm run build:h5
rtk npm run build:mp-weixin
```

The cache location avoids this session's unwritable default npm cache; it is not required for the lock semantics. No global uni CLI or HBuilderX is needed. Both scripts set `UNI_INPUT_DIR="$PWD"`, allowing the existing root-level source layout. An initial relative `UNI_INPUT_DIR=.` attempt failed component resolution because this compiler keeps the input directory as supplied; those diagnostic logs are retained. The final scripts use the absolute working directory and resolve the original imports without edits.

To apply to a separate clean checkout at the base commit, use the supplied Git format-patch:

```sh
rtk git am /Users/macmini/Desktop/yudao-mall-uniapp/.work/ruoyi-vue-pro-go/docs/batch-ab/t00-frontend-build.patch
```

The patch is not applied to the original checkout by this task.

## Verification and outputs

| Check | Result |
| --- | --- |
| Initial pinned npm install | PASS |
| Clean `npm ci` | PASS; lock unchanged |
| Full `npm ls --all` | Exit 0; optional absent dependencies remain optional |
| Initial H5 and mp-weixin production builds | Both exit 0 |
| Rebuilds after `npm ci` and deleting `dist/build` | Both exit 0 |
| Per-file SHA-256 comparison | All 1,056 files identical across both builds |
| H5 entry assets | All 3 local entry references exist |
| WeChat generated pages | JS/JSON/WXML exist for all 58 registered pages |
| WeChat generated JSON | All 199 files parse |
| Source preservation | Only allowed build/package files differ |
| Patch | Verified against a temporary index at the exact base commit |

| Target | Artifact directory in clone | Files | Exact file bytes |
| --- | --- | ---: | ---: |
| H5 | `dist/build/h5` | 179 | 2,291,432 |
| WeChat | `dist/build/mp-weixin` | 877 | 1,985,079 |

Portable `h5-build.tar.gz` and `mp-weixin-build.tar.gz` archives are local-only in `/Users/macmini/Desktop/yudao-mall-uniapp/.work/uniapp-build/dist/artifacts/`, outside the backend repository. Only metadata, text evidence, the lock snapshot, and the patch are supplied to the backend. H5 uses the existing history router and needs an index fallback when hosted. WeChat output can be imported into WeChat DevTools from `dist/build/mp-weixin`; no upload was performed. The existing app ID and environment endpoints were preserved.

Reproducibility means identical output bytes on this checkout, OS, architecture, and exact runtime across a clean dependency reinstall and output rebuild. Cross-host/cross-OS byte reproducibility has not been tested.

## Retained warnings and limits

- mp-weixin emits 50 circular-chunk warnings from existing application dependency cycles and one unsupported `img` selector warning. They are not suppressed; full logs are retained. Compilation passes, but this does not establish runtime behavior of those cycles.
- npm reports deprecated transitive `phin` packages. No dependency audit or security certification is claimed.
- The compiler reports uni statistics 2.0 enabled. No analytics configuration was changed.
- No browser interaction, API authentication, payment flow, device test, or WeChat upload/release verification was performed.

## Evidence index

All paths below are relative to this document:

- [Machine-readable baseline](evidence/t00-frontend/baseline.json), [runtime](evidence/t00-frontend/runtime.json), [verification](evidence/t00-frontend/verification.json), [source preservation](evidence/t00-frontend/source-preservation.json).
- [Install log](evidence/t00-frontend/install.log), [clean install](evidence/t00-frontend/npm-ci.log), [direct dependencies](evidence/t00-frontend/dependencies.log), [full dependency tree](evidence/t00-frontend/dependency-tree.log).
- [H5 build](evidence/t00-frontend/build-h5.log), [WeChat build](evidence/t00-frontend/build-mp-weixin.log), [H5 clean rebuild](evidence/t00-frontend/rebuild-h5.log), [WeChat clean rebuild](evidence/t00-frontend/rebuild-mp-weixin.log).
- [H5 hashes](evidence/t00-frontend/artifacts-h5.json), [WeChat hashes](evidence/t00-frontend/artifacts-mp-weixin.json); first-build hash snapshots retained alongside them.
- [Archived lock](evidence/t00-frontend/package-lock.json), [official preset snapshot](evidence/t00-frontend/official-preset-package.json), [build helper checks](evidence/t00-frontend/build-plugin-check.log).
- [Commit](evidence/t00-frontend/commit.log), [patch](t00-frontend-build.patch), [evidence checksums](evidence/t00-frontend/SHA256SUMS).

## Supplemental admin baseline

**PASS**: full admin production build at fixed official base `2e001992486e69a464c7ba0b22f110040debe9a2`, isolated commit `fbd0e3333f6fac63942b2af44a8e104191b23a73`. Node **25.8.1**, pnpm **11.19.0**. The original reference remains clean at `aab14fb0e74720dd09e964ae066f8bbde9f9012e`; its package file and lock are identical to the successful build base. Lock SHA-256: `045b4d48e08a94052ff245aaa4ff6aa92dc6b9073b0b0381c176c35eb41c9b23`.

Clean frozen installation and both production builds pass, with **2,468 byte-identical output files**. All 226 mall source/API files match the original reference and all 124 mall Vue entries occur in the compiled router. Only the five-line dependency-build policy was added locally; no application edits or custom route exclusions. See the [admin runbook, pin rationale, artifacts, and source-preservation evidence](frontend-admin-build.md).
