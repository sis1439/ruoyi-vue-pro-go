#!/usr/bin/env python3
"""Real HTTP smoke in a disposable PostgreSQL schema; never sends external messages/payments.

Build server, migrate and bootstrap binaries in tmp/batch-ab first. Requires
TEST_POSTGRES_DSN (libpq keyword DSN), T09_REDIS_ADDR, psql, and local free port 58091.
"""
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "docs/batch-ab/evidence"
DSN = os.environ["TEST_POSTGRES_DSN"]
assert "://" not in DSN, "This smoke runner requires a libpq keyword DSN"
SCHEMA = "smoke_" + secrets.token_hex(8)
env = dict(os.environ, RUOYI_DATABASE_DSN=DSN + " search_path=" + SCHEMA,
           RUOYI_JWT_SECRET=secrets.token_urlsafe(48),
           RUOYI_BOOTSTRAP_USERNAME="smoke_admin",
           RUOYI_BOOTSTRAP_PASSWORD=secrets.token_urlsafe(12),
           RUOYI_BOOTSTRAP_WEBSITE="localhost", GIN_MODE="release",
           MALL_CORS_ORIGINS="http://localhost:58092")
results = []

def sql(statement, scoped=True):
    if scoped:
        statement = "SET search_path TO " + SCHEMA + "; " + statement
    subprocess.run(["psql", "-X", "-v", "ON_ERROR_STOP=1", "-d",
                    DSN, "-c", statement],
                   check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)

def request(method, path, data=None, token=None, tenant="1", expected=0):
    headers = {"Content-Type": "application/json", "tenant-id": tenant}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request("http://localhost:58091" + path,
                                 data=json.dumps(data).encode() if data is not None else None,
                                 headers=headers, method=method)
    try:
        response = urllib.request.urlopen(req, timeout=10)
    except urllib.error.HTTPError as exc:
        response = exc
    body = json.load(response)
    code = body.get("code")
    results.append({"method": method, "path": path.split("?")[0], "http": response.status, "code": code})
    assert code == expected, (method, path.split("?")[0], response.status, code, body.get("msg"))
    return body.get("data")

server = None
sql("CREATE SCHEMA " + SCHEMA, False)
try:
    with tempfile.TemporaryDirectory(prefix="ruoyi-http-") as work, (EVIDENCE / "http-server.log").open("w") as log:
        work = Path(work)
        (work / "config").mkdir()
        config = {
            "app": {"name": "isolated-smoke", "env": "local"},
            "http": {"port": ":58091", "mode": "release"},
            "log": {"level": "warn", "filename": str(work / "server.log"), "max_size": 10},
            "database": {"driver": "postgres", "max_idle": 2, "max_open": 8, "max_lifetime": 60},
            "redis": {"addr": os.environ["T09_REDIS_ADDR"], "db": 2},
        }
        # JSON is valid YAML; no credential is written to the fixture.
        (work / "config/config.local.yaml").write_text(json.dumps(config))
        for binary in ["migrate", "bootstrap"]:
            subprocess.run([str(ROOT / "tmp/batch-ab" / binary)], env=env, cwd=work,
                           check=True, stdout=log, stderr=log)
        sql("UPDATE system_tenant SET website='localhost' WHERE id=1")
        sql("INSERT INTO member_user(mobile,password,nickname,tenant_id) SELECT '13900000001',password,'smoke member',1 FROM system_users LIMIT 1")
        server = subprocess.Popen([str(ROOT / "tmp/batch-ab/server")], cwd=work, env=env, stdout=log, stderr=log)
        for _ in range(100):
            if server.poll() is not None:
                raise RuntimeError("server startup failed; inspect http-server.log")
            try:
                with urllib.request.urlopen("http://localhost:58091/ping", timeout=.5) as response:
                    assert json.load(response)["message"] == "pong"
                    break
            except (OSError, urllib.error.URLError):
                time.sleep(.1)
        else:
            raise RuntimeError("server startup timed out")
        preflight_headers={"Origin":"http://localhost:58092","Access-Control-Request-Method":"POST",
                           "Access-Control-Request-Headers":"authorization,content-type,tenant-id,terminal,platform"}
        preflight=urllib.request.Request("http://localhost:58091/app-api/member/auth/login",headers=preflight_headers,method="OPTIONS")
        with urllib.request.urlopen(preflight,timeout=5) as response:
            allowed={h.strip().lower() for h in response.headers["Access-Control-Allow-Headers"].split(",")}
            assert response.status==204 and {"authorization","content-type","tenant-id","terminal","platform"} <= allowed
            assert response.headers["Access-Control-Allow-Origin"]=="http://localhost:58092" and "*" not in allowed
        preflight_headers["Origin"]="https://untrusted.example.invalid"
        try:
            urllib.request.urlopen(urllib.request.Request("http://localhost:58091/app-api/member/auth/login",headers=preflight_headers,method="OPTIONS"),timeout=5)
            raise AssertionError("untrusted CORS origin allowed")
        except urllib.error.HTTPError as exc:
            assert exc.code==403
        results.append({"method":"OPTIONS","path":"/app-api/member/auth/login","approved_origin_platform_header":True,"untrusted_origin_denied":True})
        login = request("POST", "/admin-api/system/auth/login", {
            "username": env["RUOYI_BOOTSTRAP_USERNAME"], "password": env["RUOYI_BOOTSTRAP_PASSWORD"]})
        token = login["accessToken"]
        permissions = request("GET", "/admin-api/system/auth/get-permission-info", token=token)
        assert "product:spu:query" in permissions["permissions"]
        assert permissions["menus"], "bootstrap must provide menus"
        page = request("GET", "/admin-api/product/spu/page?pageNo=1&pageSize=10", token=token)
        assert page["total"] == 0
        root_id = request("POST", "/admin-api/product/category/create", {"parentId":0,"name":"smoke root","sort":0,"status":0}, token)
        category_id = request("POST", "/admin-api/product/category/create", {"parentId":root_id,"name":"smoke category","sort":0,"status":0}, token)
        brand_id = request("POST", "/admin-api/product/brand/create", {"name":"smoke brand","picUrl":"https://example.invalid/brand.png","sort":0,"status":0}, token)
        product = {"name":"smoke product","categoryId":category_id,"brandId":brand_id,
                   "picUrl":"https://example.invalid/product.png","sliderPicUrls":[],"sort":2,
                   "specType":False,"deliveryTypes":[2],"deliveryTemplateId":1,
                   "giveIntegral":0,"subCommissionType":False,"virtualSalesCount":0,
                   "skus":[{"price":100,"stock":2,"marketPrice":100,"costPrice":20,"properties":[]}]}
        product_id = request("POST", "/admin-api/product/spu/create", product, token)
        detail = request("GET", "/admin-api/product/spu/get-detail?id="+str(product_id), token=token)
        member_login = request("POST", "/app-api/member/auth/login", {"mobile":"13900000001","password":env["RUOYI_BOOTSTRAP_PASSWORD"]})
        member_token = member_login["accessToken"]
        request("GET", "/admin-api/product/spu/page?pageNo=1&pageSize=10", token=member_token, expected=403)
        address = {"name":"Test recipient","mobile":"13900000001","areaId":110101,"detailAddress":"Isolated test address","defaultStatus":True}
        address_id = request("POST", "/app-api/member/address/create", address, member_token)
        address.update(id=address_id, defaultStatus=False)
        request("PUT", "/app-api/member/address/update", address, member_token)
        changed_address = request("GET", "/app-api/member/address/get?id="+str(address_id), token=member_token)
        assert changed_address["defaultStatus"] is False, "false address default lost"
        request("GET", "/app-api/member/address/list", token=member_token, tenant="2", expected=403)
        cart_id = request("POST", "/app-api/trade/cart/add", {"skuId":detail["skus"][0]["id"],"count":1}, member_token)
        assert request("GET", "/app-api/trade/cart/get-count", token=member_token) == 1
        request("DELETE", "/app-api/trade/cart/delete?ids="+str(cart_id), token=member_token)
        assert request("GET", "/app-api/trade/cart/get-count", token=member_token) == 0
        product["id"] = product_id
        product["sort"] = 0
        product["skus"][0].update(id=detail["skus"][0]["id"], stock=0)
        request("PUT", "/admin-api/product/spu/update", product, token)
        detail = request("GET", "/admin-api/product/spu/get-detail?id="+str(product_id), token=token)
        assert detail["sort"] == 0 and detail["stock"] == 0 and detail["skus"][0]["stock"] == 0, "zero-value product updates lost"
        request("GET", "/admin-api/product/spu/page?pageNo=1&pageSize=10", token=token, tenant="2", expected=403)
        request("GET", "/admin-api/product/spu/page?pageNo=1&pageSize=10", expected=401)
        refreshed = request("POST", "/admin-api/system/auth/refresh-token?" + urllib.parse.urlencode({"refreshToken": login["refreshToken"]}), {})
        request("GET", "/admin-api/system/auth/get-permission-info", token=token, expected=401)
        request("GET", "/admin-api/system/auth/get-permission-info", token=refreshed["accessToken"])
        request("POST", "/admin-api/system/auth/logout", {}, token=refreshed["accessToken"])
        request("GET", "/admin-api/system/auth/get-permission-info", token=refreshed["accessToken"], expected=401)
        # This evidence must never contain credentials, signed tokens or refresh query strings.
        log.flush()
        text = (EVIDENCE / "http-server.log").read_text()
        for value in [env["RUOYI_BOOTSTRAP_PASSWORD"], token, login["refreshToken"], refreshed["accessToken"]]:
            assert value not in text, "secret leaked in server log"
        (EVIDENCE / "http-smoke.json").write_text(json.dumps({"passed": True, "checks": results}, indent=2) + "\n")
        print("PASS: isolated bootstrap, HTTP login, permissions, product page, tenant rejection, refresh rotation, logout and log redaction")
finally:
    if server is not None:
        server.terminate()
        try:
            server.wait(timeout=5)
        except subprocess.TimeoutExpired:
            server.kill()
            server.wait()
    sql("DROP SCHEMA " + SCHEMA + " CASCADE", False)
