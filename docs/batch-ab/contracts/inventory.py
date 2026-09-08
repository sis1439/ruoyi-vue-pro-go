"""Pinned-source inventory. Static discovery only; never an alignment verdict."""
import csv
import json
import re
import subprocess
from functools import cache
from pathlib import Path

HERE = Path(__file__).resolve().parent
PINS = json.loads((HERE / "references.json").read_text())
APP_FILES = ["member/auth", "member/user", "member/address", "product/spu", "product/category", "trade/cart", "trade/order", "trade/afterSale", "trade/delivery", "trade/config", "pay/order", "pay/channel", "infra/file", "infra/tenant", "promotion/coupon", "member/point"]
ADMIN_PREFIXES = ("src/api/login/", "src/api/system/permission/", "src/api/system/menu/", "src/api/mall/product/", "src/api/mall/trade/order/", "src/api/mall/trade/afterSale/", "src/api/mall/trade/delivery/", "src/api/mall/trade/config/", "src/api/member/user/", "src/api/member/address/", "src/api/pay/order/", "src/api/pay/channel/", "src/api/pay/app/", "src/api/infra/file/", "src/api/mall/promotion/coupon/")

def git(repo, *args):
    return subprocess.check_output(["rtk", "proxy", "git", "-C", PINS[repo]["path"], *args], text=True)

@cache
def source(repo, path):
    return git(repo, "show", PINS[repo]["commit"] + ":" + path)

@cache
def files(repo):
    return git(repo, "ls-tree", "-r", "--name-only", PINS[repo]["commit"]).splitlines()

def routes():
    result = {}
    for path in files("go"):
        if not path.startswith("internal/api/router/") or not path.endswith(".go"):
            continue
        groups = {"engine": ""}
        guards = {"engine": []}
        for n, line in enumerate(source("go", path).splitlines(), 1):
            line = line.split("//")[0]
            m = re.search(r'(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"', line)
            if m and m[2] in groups:
                groups[m[1]] = groups[m[2]] + m[3]
                guards[m[1]] = list(guards[m[2]])
                if 'middleware.Auth()' in line:
                    guards[m[1]].append('middleware.Auth()')
            use = re.search(r'(\w+)\.Use\((.*)\)', line)
            if use and use[1] in guards:
                guards[use[1]].append(use[2])
            m = re.search(r'(\w+)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]*)",(.*)', line)
            if m and m[1] in groups:
                result[(m[2], groups[m[1]] + m[3])] = {"source": f"{path}:{n}", "registration": line.strip(), "group_middleware": guards[m[1]].copy()}
    return result

def java_routes():
    result = {}
    paths = files("java")
    by_name = {}
    for p in paths:
        by_name.setdefault(p.rsplit("/", 1)[-1], []).append(p)
    for path in paths:
        if not path.endswith("Controller.java") or "/controller/" not in path:
            continue
        side = "app" if "/controller/app/" in path else "admin" if "/controller/admin/" in path else None
        if not side:
            continue
        text = source("java", path)
        root = re.search(r'@RequestMapping\("([^"]+)"\)', text)
        if not root:
            continue
        mappings = list(re.finditer(r'@(Get|Post|Put|Delete|Patch)Mapping\(([^\n]+)\)', text))
        for i, m in enumerate(mappings):
            block = text[m.start():mappings[i+1].start() if i+1 < len(mappings) else len(text)]
            types = sorted(set(re.findall(r'\b\w+(?:ReqVO|RespVO|RespDTO|ReqDTO)\b', block)))
            vo = [p for t in types for p in by_name.get(t + ".java", [])]
            for suffix in re.findall(r'"([^"]+)"', m[2]):
                key = (m[1].upper(), f"/{side}-api" + root[1] + '/' + suffix.lstrip('/'))
                result[key] = {"source": f"{path}:{text[:m.start()].count(chr(10))+1}", "method_source": block.strip(), "vo_sources": vo}
    return result

def frontend_calls(repo, path):
    text = source(repo, path)
    calls = []
    for m in re.finditer(r'url:\s*([\x27"`])([^\x27"`]+)\1', text):
        # Literal URL calls only. Raw source retains dynamic query values.
        url = '/' + m[2].split("?")[0].lstrip('/')
        if "${" in url:
            continue
        start = text.rfind("request", 0, m.start())
        if repo == "admin":
            method_match = re.match(r'request\.(\w+)', text[start:])
            if not method_match:
                continue
            method = {"upload": "POST", "download": "GET", "postOriginal": "POST"}.get(method_match[1], method_match[1].upper())
        else:
            method_match = re.search(r'method:\s*[\x27"](\w+)', text[m.end():m.end()+180])
            method = method_match[1].upper() if method_match else "UNKNOWN"
        if method not in {"GET", "POST", "PUT", "DELETE", "PATCH"}:
            continue
        n = text[:m.start()].count("\n") + 1
        calls.append((method, ("/app-api" if repo == "uniapp" else "/admin-api") + url, f"{path}:{n}"))
    if repo == "uniapp" and path.endswith("infra/file.js"):
        calls.append(("POST", "/app-api/infra/file/upload", f"{path}:12"))
    return calls

def build():
    go, java = routes(), java_routes()
    app_paths = [f"sheep/api/{f}.js" for f in APP_FILES]
    admin_paths = [p for p in files("admin") if p.endswith(".ts") and p.startswith(ADMIN_PREFIXES)]
    rows, evidence, fields = [], {}, {}
    for repo, paths in (("uniapp", app_paths), ("admin", admin_paths)):
        consumers = [(p, source(repo, p)) for p in files(repo) if (p.startswith("pages/") or p.startswith("src/views/") or p.startswith("sheep/components/") or p.startswith("sheep/platform/") or p.startswith("sheep/store/")) and p.endswith((".vue", ".js", ".ts"))]
        for path in paths:
            api_import = "@/" + (path[:-3] if repo == "uniapp" else path[:-3].removesuffix("/index"))
            uses = [p for p,t in consumers if api_import in t or api_import.removeprefix("@/") in t]
            for method, url, loc in frontend_calls(repo, path):
                key = method, url
                j, g = java.get(key, {}), go.get(key, {})
                other = sorted(m for m,u in go if u == url and m != method)
                cid = f"T01-{len(rows)+1:03d}"
                finding = "Route registered only; DTO/binding/auth/behavior NOT accepted"
                if not g:
                    finding = "Method mismatch; Go registers " + ",".join(other) if other else "No matching registration in pinned internal/api/router; not proof service code absent"
                row = {"id": cid, "client": repo, "method": method, "path": url, "frontend_source": loc, "consumer_import_evidence": ";".join(uses) or "UNRESOLVED (API wrapper exists; enabled page not proven)", "enablement": "candidate first-phase; runtime menus/config unverified", "go_router_source": g.get("source", "NOT_FOUND"), "java_controller_source": j.get("source", "UNRESOLVED"), "status": "部分对齐" if g else "未实现", "status_basis": finding, "auth_tenant": "See wire-contract.md; endpoint auth/tenant/role enforcement unverified", "request_encoding": "See evidence method_source and fixed wrapper; body/query/form requirements pending per-field validation", "required_types": "Java VO declarations in fields.json; Go tags in same file; inheritance/validation NOT fully resolved", "arrays": "Endpoint-specific; settlement indexed dotted keys; other serializers require wire capture", "dates_timezone": "Java global LocalDateTime epoch ms vs Go time.Time JSON string candidate gap; endpoint mapping pending", "pagination": "For page: pageNo/pageSize -> data.list/data.total; defaults/empty list pending runtime", "money": "Integer fen where annotated 分 in VO; never infer every numeric field is money", "null_missing_zero": "Distinct; per-endpoint verification pending; see edge cases", "enums_ids": "Preserve Java/Go declarations; safe-integer boundary pending wire decision", "http_business_code": "CommonResult code/msg/data; Go helpers HTTP 200 incl business error; middleware/Java failures pending", "side_effects": "Controller method_source records service calls; transactions/auth/payment side effects NOT tested", "tests": "tests/test_inventory.py; tests/frontend-contracts.cjs (selected app calls only)", "acceptance": "待处理: Batch D HTTP/DB/page E2E not run"}
                signature = re.search(r'public\s+(?:CommonResult|ResponseModel).*?\{', j.get('method_source',''), re.S)
                if signature:
                    row['required_types'] = ' '.join(signature[0][:-1].split()) + '; VO annotations: fields.json (inheritance unresolved)'
                row['auth_tenant'] = 'Go group middleware: ' + ','.join(g.get('group_middleware',[])) + '; inline middleware: see registration; global tenant middleware NOT acceptance; wire-contract.md'
                row['request_encoding'] = ('multipart file + directory' if url.endswith('/file/upload') else 'Java @RequestBody JSON' if '@RequestBody' in j.get('method_source','') else 'Java query/model binding; see signature' if j else 'UNRESOLVED; use fixed frontend source')
                row['dates_timezone'] = 'Java LocalDateTime epoch ms uses JVM system zone; Go AppTradeOrder uses JsonDateTime ms while admin TradeOrderBase uses time.Time; field-by-field review pending'
                if url == '/app-api/trade/order/get-detail':
                    row['status_basis'] = 'Static gap: Go ignores sync; nil/wrong-owner order returns ErrNotFound; fixed Java returns success(null)'
                    row['null_missing_zero'] = 'Java absent/wrong-owner order code=0,data=null; Go ErrNotFound; runtime not executed'
                if url == '/app-api/pay/order/submit':
                    row['status_basis'] = 'Static gap: wx JSAPI DisplayContent bare PrepayId; fixed client JSON.parse requires signed object; frontend negative test reproduces consumer rejection'
                if url == '/app-api/trade/order/settlement':
                    row['arrays'] = 'GET items%5B0%5D.skuId=7&items%5B0%5D.count=2; cartId optional; frontend-contracts.cjs executes fixed serializer'
                    row['required_types'] += '; Go PointStatus *bool required, DeliveryType int required, Items nonempty handler check'
                if '/system/captcha/' in url:
                    row['http_business_code'] = 'EXCEPTION: fixed Java returns Anji ResponseModel, admin uses postOriginal; do not impose CommonResult; Go payload acceptance pending'
                row['tests'] = 'tests/test_inventory.py provenance for all rows; tests/frontend-contracts.cjs SELECTED cases only; see wire-contract.md'
                rows.append(row)
                evidence[cid] = {"go": g, "java": j, "frontend": {"repo": repo,"source":loc}}
                for p in j.get("vo_sources", []):
                    if p not in fields:
                        s = source("java", p)
                        fields[p] = {"repo":"java", "declarations_and_annotations": [f"{n}:{l.strip()}" for n,l in enumerate(s.splitlines(),1) if re.search(r'^\s*(private |public class |@|extends )',l)]}
    # Refresh is implemented outside src/api by the fixed admin Axios interceptor.
    path = "src/config/axios/service.ts"
    line = next(n for n,l in enumerate(source("admin",path).splitlines(),1) if "axios.post(base_url + '/system/auth/refresh-token" in l)
    template = dict(next(r for r in rows if r["path"] == "/admin-api/system/auth/logout"))
    key = "POST", "/admin-api/system/auth/refresh-token"
    template.update(id=f"T01-{len(rows)+1:03d}",method=key[0],path=key[1],frontend_source=f"{path}:{line}",consumer_import_evidence="Axios response refresh interceptor",go_router_source=go.get(key,{}).get("source","NOT_FOUND"),java_controller_source=java.get(key,{}).get("source","UNRESOLVED"))
    template['request_encoding'] = 'POST query refreshToken from fixed Axios interceptor; no JSON body'
    template['required_types'] = 'refreshToken string query; Java SystemAuthController.refreshToken signature in evidence.json'
    template['side_effects'] = 'Token renewal; rotation/revocation and Redis failure behavior untested'
    rows.append(template)
    evidence[template["id"]] = {"go":go.get(key,{}),"java":java.get(key,{}),"frontend":{"repo":"admin","source":template["frontend_source"]}}
    for p in files("go"):
        if p.startswith("internal/api/contract/") and p.endswith(".go") and not p.endswith("_test.go") and any(k in p for k in ("trade", "product", "member", "auth", "file", "pay")):
            fields["go:"+p] = {"repo":"go", "declarations_and_annotations":[f"{n}:{l.strip()}" for n,l in enumerate(source("go",p).splitlines(),1) if "`json:" in l or l.startswith("type ")]}
    return rows,evidence,fields

if __name__ == "__main__":
    rows,evidence,fields = build()
    with (HERE.parent / "api-compatibility.csv").open("w",newline="") as f:
        w=csv.DictWriter(f,fieldnames=list(rows[0])); w.writeheader(); w.writerows(rows)
    (HERE/"evidence.json").write_text(json.dumps(evidence,ensure_ascii=False,indent=2)+"\n")
    (HERE/"fields.json").write_text(json.dumps(fields,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps({"rows":len(rows),"route_present":sum(r["status"]=="部分对齐" for r in rows),"route_or_method_missing":sum(r["status"]=="未实现" for r in rows),"java_unresolved":sum(r["java_controller_source"]=="UNRESOLVED" for r in rows),"field_sources":len(fields)},ensure_ascii=False))
