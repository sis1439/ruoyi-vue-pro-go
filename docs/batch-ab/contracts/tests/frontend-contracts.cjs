// Executes fixed frontend functions with request/uni doubles. No network or browser.
const assert = require('node:assert/strict');
const { test } = require('node:test');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { stripTypeScriptTypes } = require('node:module');
const pins = JSON.parse(fs.readFileSync(path.join(__dirname, '../references.json')));
const read = file => execFileSync('rtk', ['proxy', 'git', '-C', pins.uniapp.path, 'show', `${pins.uniapp.commit}:${file}`], {encoding: 'utf8'});
function api(name, extras = {}) {
  const context = vm.createContext({ request: x => x, isEmpty: x => x == null || x === '', ...extras });
  const code = read(`sheep/api/${name}.js`).replace(/^import .*;\s*$/gm, '').replace(/export default (\w+);?/, 'globalThis.api = $1;');
  vm.runInContext(code, context);
  return context.api;
}
const plain = x => JSON.parse(JSON.stringify(x));

test('login JSON, logout POST, refresh token query, social mini-app fields', () => {
  const auth = api('member/auth');
  const input = {mobile:'13800000000', password:'synthetic-only'};
  assert.equal(auth.login(input).data, input);
  assert.equal(auth.login(input).method, 'POST');
  assert.equal(auth.logout().url, '/member/auth/logout');
  assert.equal(auth.logout().method, 'POST');
  assert.deepEqual(plain(auth.refreshToken('synthetic').params), {refreshToken:'synthetic'});
  assert.equal(auth.refreshToken('synthetic').data, undefined);
  assert.deepEqual(plain(auth.weixinMiniAppLogin('p','l','s').data), {phoneCode:'p',loginCode:'l',state:'s'});
});
test('address create/update body and delete query remain distinct', () => {
  const a=api('member/address'), data={name:'fixture',areaId:110101,defaultStatus:false};
  assert.equal(a.createAddress(data).data,data);
  assert.equal(a.updateAddress(data).method,'PUT');
  assert.deepEqual(plain(a.deleteAddress(7).params),{id:7});
  assert.equal(a.deleteAddress(7).method,'DELETE');
});
test('product IDs arrays and pagination forwarded, zero kept', () => {
  const a=api('product/spu');
  assert.deepEqual(plain(a.getSpuListByIds([]).params), {ids:[]});
  assert.deepEqual(plain(a.getSpuPage({pageNo:1,pageSize:10,minPrice:0}).params),{pageNo:1,pageSize:10,minPrice:0});
  assert.equal(a.getSpuDetail(7).params.id,7);
});
test('cart JSON false and query array survive wrapper', () => {
  const a=api('trade/cart');
  assert.deepEqual(plain(a.updateCartSelected({ids:[1,2],selected:false}).data),{ids:[1,2],selected:false});
  assert.deepEqual(plain(a.deleteCart([1,2]).params),{ids:[1,2]});
  assert.equal(a.getCartList().custom.auth,true);
});
test('settlement emits Spring indexed dotted keys, preserves false, omits absent optional IDs', () => {
  const a=api('trade/order');
  const data={items:[{skuId:7,count:2,cartId:9},{skuId:8,count:1}],pointStatus:false,deliveryType:1,couponId:0,addressId:0};
  const r=a.settlementOrder(data), q=new URL(r.url,'https://fixture.invalid').searchParams;
  assert.equal(r.method,'GET');
  assert.equal(q.get('items[0].skuId'),'7');
  assert.equal(q.get('items[0].count'),'2');
  assert.equal(q.get('items[0].cartId'),'9');
  assert.equal(q.get('items[1].skuId'),'8');
  assert.equal(q.has('items[1].cartId'),false);
  assert.equal(q.get('pointStatus'),'false');
  assert.equal(q.has('couponId'),false);
  assert.equal(q.has('addressId'),false);
  assert.equal(data.items.length,2);
});
test('settlement empty input is serialized, not validated by wrapper', () => {
  const r=api('trade/order').settlementOrder({items:[],pointStatus:false,deliveryType:2});
  assert.equal(r.url.includes('items'),false);
  // Server must reject this; this test makes no assertion that it currently does.
});
test('order create is JSON; detail preserves sync true/false/omitted and cancellation DELETE', () => {
  const a=api('trade/order'), data={items:[{skuId:7,count:1}],pointStatus:false};
  assert.equal(a.createOrder(data).data,data);
  assert.deepEqual(plain(a.getOrderDetail(123,true).params),{id:123,sync:true});
  assert.deepEqual(plain(a.getOrderDetail(123,false).params),{id:123,sync:false});
  assert.deepEqual(plain(a.getOrderDetail(123).params),{id:123});
  assert.equal(a.cancelOrder(123).method,'DELETE');
  assert.equal(a.receiveOrder(123).method,'PUT');
});
test('pay submission retains channelExtras; get supports id/no and sync', () => {
  const a=api('pay/order'), data={id:4,channelCode:'wx_lite',channelExtras:{openid:'synthetic'}};
  assert.equal(a.submitOrder(data).data,data);
  assert.deepEqual(plain(a.getOrder(undefined,false,'P1').params),{no:'P1',sync:false});
});
test('delivery lookup and after-sale PUT /delivery are fixed client contracts', () => {
  assert.equal(api('trade/delivery').getDeliveryPickUpStore(7).params.id,7);
  const a=api('trade/afterSale');
  assert.equal(a.deliveryAfterSale({id:7,logisticsNo:'fixture'}).method,'PUT');
  assert.equal(a.createAfterSale({refundPrice:1}).data.refundPrice,1);
  assert.equal(a.cancelAfterSale(7).method,'DELETE');
  assert.equal(a.getAfterSaleLogList(7).url,'/trade/after-sale-log/list');
});
test('coupon and trade-config requests remain in first-phase inventory', () => {
  assert.equal(api('promotion/coupon').takeCoupon(7).data.templateId,7);
  assert.equal(api('trade/config').getTradeConfig().url,'/trade/config/get');
});
test('upload uses multipart file plus directory and tenant header', async () => {
  let captured;
  const a=api('infra/file',{baseUrl:'https://fixture.invalid',apiPath:'/app-api',tenantId:'7',getAccessToken:()=> 'synthetic',uni:{showLoading(){},hideLoading(){},uploadFile(o){captured=o;o.success({data:JSON.stringify({code:0,data:'fixture-url'})});o.complete();}}});
  const result=await a.uploadFile('/synthetic.png','reviews');
  assert.equal(captured.url,'https://fixture.invalid/app-api/infra/file/upload');
  assert.equal(captured.name,'file');
  assert.equal(captured.formData.directory,'reviews');
  assert.equal(captured.header['tenant-id'],'7');
  assert.equal(captured.header.Authorization,'Bearer synthetic');
  assert.equal(result.code,0);
});
function payment() {
  const src=read('sheep/platform/pay.js');
  const method=src.slice(src.indexOf('  async wechatMiniProgramPay()'),src.indexOf('  // 余额支付'));
  const calls=[];
  const context=vm.createContext({uni:{requestPayment:x=>calls.push(x)},sheep:{$helper:{toast(){}}}});
  vm.runInContext(`globalThis.pay = {${method}}`,context);
  return {pay:context.pay,calls};
}
test('actual mini-app payment method parses signed displayContent and maps packageValue', async () => {
  const {pay,calls}=payment();
  const fixture={timeStamp:'1700000000',nonceStr:'fixture',packageValue:'prepay_id=fixture',signType:'RSA',paySign:'synthetic-not-a-signature'};
  pay.prepay=async channel=>{assert.equal(channel,'wx_lite');return {code:0,data:{displayContent:JSON.stringify(fixture)}};};
  await pay.wechatMiniProgramPay();
  assert.equal(calls.length,1);
  for(const k of ['timeStamp','nonceStr','signType','paySign']) assert.equal(calls[0][k],fixture[k]);
  assert.equal(calls[0].package,fixture.packageValue);
});
test('bare prepay ID cannot be consumed as displayContent (known baseline gap)', async () => {
  const {pay,calls}=payment();
  pay.prepay=async()=>({code:0,data:{displayContent:'wx-prepay-fixture'}});
  await assert.rejects(pay.wechatMiniProgramPay(),{name:'SyntaxError'});
  assert.equal(calls.length,0);
});
test('payment business failure does not invoke terminal payment', async () => {
  const {pay,calls}=payment();pay.prepay=async()=>({code:400,msg:'fixture failure'});
  await pay.wechatMiniProgramPay();assert.equal(calls.length,0);
});
test('numeric IDs above JavaScript safe range lose precision; no invented coercion policy', () => {
  const a=api('trade/order');
  assert.equal(a.getOrderDetail(Number.MAX_SAFE_INTEGER).params.id,9007199254740991);
  assert.equal(JSON.parse('{"id":9007199254740993}').id,9007199254740992);
  assert.equal(a.getOrderDetail('9007199254740993').params.id,'9007199254740993');
});

function adminApi(file) {
  let code=execFileSync('rtk',['proxy','git','-C',pins.admin.path,'show',`${pins.admin.commit}:${file}`],{encoding:'utf8'});
  code=stripTypeScriptTypes(code).replace(/^import .*$/gm,'');
  const names=[...code.matchAll(/export\s+(?:async\s+)?(?:const|function)\s+(\w+)/g)].map(m=>m[1]);
  code=code.replace(/export\s+/g,'');
  const request=Object.fromEntries(['get','post','put','delete','upload','download','postOriginal'].map(method=>[method, x=>({method,...x})]));
  const ctx=vm.createContext({request});
  vm.runInContext(code+`;globalThis.api={${names.join(',')}}`,ctx);
  return ctx.api;
}
test('fixed admin login, logout, permission info and menu retrieval', () => {
  const a=adminApi('src/api/login/index.ts');
  assert.equal(a.login({username:'fixture'}).data.username,'fixture');
  assert.equal(a.loginOut().method,'post');
  assert.equal(a.getInfo().url,'/system/auth/get-permission-info');
  assert.equal(adminApi('src/api/system/menu/index.ts').getSimpleMenusList().url,'/system/menu/simple-list');
});
test('fixed admin delivery body, pickup ID query and verification code query', async () => {
  const a=adminApi('src/api/mall/trade/order/index.ts');
  const r=await a.deliveryOrder({id:7,logisticsId:1,logisticsNo:'fixture'});
  assert.equal(r.method,'put');assert.equal(r.data.logisticsNo,'fixture');
  assert.equal((await a.pickUpOrder(7)).url,'/trade/order/pick-up-by-id?id=7');
  assert.equal((await a.pickUpOrderByVerifyCode('012345')).params.pickUpVerifyCode,'012345');
});
test('fixed admin after-sale refund is PUT query; disagree body', async () => {
  const a=adminApi('src/api/mall/trade/afterSale/index.ts');
  const refund=await a.refund(7);assert.equal(refund.method,'put');assert.equal(refund.url,'/trade/after-sale/refund?id=7');
  assert.equal((await a.disagree({id:7,auditReason:'fixture'})).data.auditReason,'fixture');
});
test('actual app response interceptor preserves null/empty/missing/zero and business errors', async () => {
  let response;
  class Request {constructor(){this.interceptors={request:{use(){}},response:{use(fn){response=fn}}}}}
  const ctx=vm.createContext({Request,baseUrl:'https://fixture.invalid',apiPath:'/app-api',tenantId:7,$platform:{name:'fixture'},$store:()=>({}),uni:{showToast(){}},getTerminal:()=>20});
  let code=read('sheep/request/index.js').replace(/^import .*;\s*$/gm,'').replace(/export default request;?/,'').replace(/export const /g,'const ');
  vm.runInContext(code,ctx);
  for(const body of [{code:0,data:null},{code:0,data:[]},{code:0},{code:0,data:0},{code:0,data:false},{code:403,msg:'fixture denied',data:null}]) {
    const result=await response({config:{url:'/fixture',custom:{}},data:body});
    assert.equal(result,body);
    assert.equal(Object.hasOwn(result,'data'),Object.hasOwn(body,'data'));
  }
});
