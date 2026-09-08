-- Service-backed identifiers, retained even after soft deletion: external retries must not reuse them.
-- Sources: trade/order_update.go CreateOrder; pay/order.go CreateOrder (AppID, MerchantOrderId).
BEGIN;
CREATE UNIQUE INDEX trade_order_tenant_no ON trade_order(tenant_id,no) WHERE no<>'';
CREATE UNIQUE INDEX pay_order_tenant_app_merchant ON pay_order(tenant_id,app_id,merchant_order_id) WHERE merchant_order_id IS NOT NULL AND merchant_order_id<>'';
-- PayRefund.validatePayRefundExist and notification lookup (AppID, No).
CREATE UNIQUE INDEX pay_refund_tenant_app_merchant ON pay_refund(tenant_id,app_id,merchant_refund_id) WHERE merchant_refund_id IS NOT NULL AND merchant_refund_id<>'';
CREATE UNIQUE INDEX pay_extension_tenant_no ON pay_order_extension(tenant_id,no) WHERE no IS NOT NULL AND no<>'';
COMMIT;
