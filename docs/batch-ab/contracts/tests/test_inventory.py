"""Inventory provenance and fixed-baseline gap tests, NOT live API acceptance."""
import csv
import hashlib
import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import inventory as inv

class InventoryTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        with (inv.HERE.parent / 'api-compatibility.csv').open() as f:
            cls.rows = list(csv.DictReader(f))
        cls.evidence = json.loads((inv.HERE / 'evidence.json').read_text())
        cls.routes = inv.routes()

    def test_fixed_commits_and_reference_checksums(self):
        self.assertEqual(inv.PINS['uniapp']['commit'], '3c4bf3864415054a88fe616a414e098972329412')
        for repo, pin in inv.PINS.items():
            self.assertEqual(inv.git(repo, 'rev-parse', pin['commit'] + '^{commit}').strip(), pin['commit'])
            for path, digest in pin.get('files', {}).items():
                self.assertEqual(hashlib.sha256(inv.source(repo,path).encode()).hexdigest(), digest['sha256'])

    def test_scope_and_no_invented_acceptance(self):
        keys = {(r['method'], r['path']) for r in self.rows}
        required = [('POST','/app-api/member/auth/login'),('POST','/app-api/member/auth/refresh-token'),('POST','/app-api/member/auth/logout'),('GET','/admin-api/system/auth/get-permission-info'),('GET','/admin-api/system/menu/simple-list'),('GET','/app-api/product/spu/get-detail'),('GET','/app-api/member/address/list'),('POST','/app-api/trade/cart/add'),('GET','/app-api/trade/order/settlement'),('POST','/app-api/trade/order/create'),('POST','/app-api/pay/order/submit'),('GET','/app-api/trade/order/get-detail'),('PUT','/admin-api/trade/order/delivery'),('PUT','/admin-api/trade/order/pick-up-by-id'),('POST','/app-api/trade/after-sale/create'),('PUT','/app-api/trade/after-sale/delivery'),('POST','/app-api/infra/file/upload'),('POST','/admin-api/infra/file/upload'),('GET','/app-api/promotion/coupon/page'),('GET','/app-api/member/point/record/page')]
        for key in required:
            self.assertIn(key, keys)
        self.assertEqual(len({r['id'] for r in self.rows}), len(self.rows))
        for row in self.rows:
            self.assertNotEqual(row['status'], '已对齐')
            self.assertNotEqual(row['status'], '不启用')
            self.assertIn('not run', row['acceptance'])
            for field,value in row.items():
                self.assertTrue(value, (row['id'],field))

    def test_every_source_locator_and_route_claim_has_pinned_evidence(self):
        for row in self.rows:
            item=self.evidence[row['id']]
            path,line=row['frontend_source'].rsplit(':',1)
            self.assertIn(row['path'].split('-api',1)[1].lstrip('/'), inv.source(row['client'],path).splitlines()[int(line)-1])
            key=row['method'],row['path']
            self.assertEqual(row['go_router_source']!='NOT_FOUND',key in self.routes)
            if item['java']:
                path,line=item['java']['source'].rsplit(':',1)
                self.assertIn('Mapping(',inv.source('java',path).splitlines()[int(line)-1])
                self.assertIn('public ',item['java']['method_source'])

    def test_baseline_after_sale_method_mismatch_and_missing_app_upload(self):
        self.assertIn(('POST','/app-api/trade/after-sale/delivery'),self.routes)
        self.assertNotIn(('PUT','/app-api/trade/after-sale/delivery'),self.routes)
        self.assertNotIn(('POST','/app-api/infra/file/upload'),self.routes)
        self.assertIn(('POST','/admin-api/infra/file/upload'),self.routes)

    def test_baseline_order_sync_and_null_divergence_is_not_closed(self):
        go=inv.source('go','internal/api/handler/app/mall/trade/order.go')
        detail=go.split('func (h *AppTradeOrderHandler) GetOrderDetail(',1)[1].split('\nfunc ',1)[0]
        self.assertNotIn('c.Query("sync")',detail)
        self.assertIn('order == nil || order.UserID != context.GetUserId(c)',detail)
        self.assertIn('errors.ErrNotFound',detail)
        item=next(e for e in self.evidence.values() if e['frontend']['repo']=='uniapp' and 'getOrderDetail' in e.get('java',{}).get('method_source',''))
        self.assertIn('success(null)',item['java']['method_source'])
        self.assertIn('syncOrderPayStatusQuietly',item['java']['method_source'])

    def test_baseline_bare_prepay_output_has_source_evidence(self):
        go=inv.source('go','internal/service/pay/client/weixin/client.go')
        self.assertIn('DisplayContent: *resp.PrepayId',go)
        frontend=inv.source('uniapp','sheep/platform/pay.js')
        self.assertIn('JSON.parse(data.displayContent)',frontend)
        self.assertIn('package: payConfig.packageValue',frontend)

    def test_dm_admin_is_not_the_yudao_vue_application(self):
        self.assertTrue(inv.source('dm-admin','go.mod').startswith('module dm-admin'))
        self.assertIn('github.com/gin-gonic/gin',inv.source('dm-admin','go.mod'))
        self.assertIn('dm-admin/go-admin/adapter/gin',inv.source('dm-admin','main.go'))
        self.assertNotIn('package.json',inv.files('dm-admin'))
        self.assertEqual(json.loads(inv.source('admin','package.json'))['name'],'yudao-ui-admin-vue3')

if __name__ == '__main__':
    unittest.main(verbosity=2)
