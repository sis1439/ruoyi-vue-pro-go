-- Reviewed query/business keys; do not infer foreign keys from *_id names.
BEGIN;
-- Username/mobile lookups are tenant-scoped and ignore deleted rows. Empty mobile is allowed.
CREATE UNIQUE INDEX system_users_tenant_username_live ON system_users(tenant_id,username) WHERE deleted=0;
CREATE UNIQUE INDEX member_user_tenant_mobile_live ON member_user(tenant_id,mobile) WHERE deleted=0 AND mobile IS NOT NULL AND mobile<>'';
CREATE UNIQUE INDEX system_role_tenant_code_live ON system_role(tenant_id,code) WHERE deleted=0;
CREATE UNIQUE INDEX system_user_role_live ON system_user_role(tenant_id,user_id,role_id) WHERE deleted=0;
CREATE UNIQUE INDEX system_role_menu_live ON system_role_menu(tenant_id,role_id,menu_id) WHERE deleted=0;
CREATE UNIQUE INDEX pay_app_tenant_key_live ON pay_app(tenant_id,app_key) WHERE deleted=0;
CREATE UNIQUE INDEX trade_config_tenant_live ON trade_config(tenant_id) WHERE deleted=0;
CREATE UNIQUE INDEX member_config_tenant_live ON member_config(tenant_id) WHERE deleted=0;
-- Daily statistics are recalculated, not appended on retry.
CREATE UNIQUE INDEX product_statistics_day_live ON product_statistics(tenant_id,spu_id,time) WHERE deleted=0;
-- Order page/detail, order-item loading, SKU lookup, coupon selection and tenant file listing.
CREATE INDEX trade_order_tenant_user_time ON trade_order(tenant_id,user_id,create_time DESC,id DESC) WHERE deleted=0;
CREATE INDEX trade_order_item_tenant_order ON trade_order_item(tenant_id,order_id) WHERE deleted=0;
CREATE INDEX product_sku_tenant_spu ON product_sku(tenant_id,spu_id) WHERE deleted=0;
CREATE INDEX trade_cart_tenant_user ON trade_cart(tenant_id,user_id) WHERE deleted=0;
CREATE INDEX promotion_coupon_tenant_user_status ON promotion_coupon(tenant_id,user_id,status) WHERE deleted=0;
CREATE INDEX infra_file_tenant_time ON infra_file(tenant_id,create_time DESC) WHERE deleted=0;
CREATE INDEX product_browse_tenant_spu_time ON product_browse_history(tenant_id,spu_id,create_time) WHERE deleted=0;
CREATE INDEX product_favorite_tenant_spu_time ON product_favorite(tenant_id,spu_id,create_time) WHERE deleted=0;
COMMIT;
