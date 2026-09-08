-- Deploy with writers stopped; these mappings describe the pre-alignment Go enums.
BEGIN;
ALTER TABLE member_user ADD COLUMN IF NOT EXISTS email varchar(255) NOT NULL DEFAULT '';
UPDATE pay_order SET status = CASE status WHEN 20 THEN 30 WHEN 30 THEN 20 END WHERE status IN (20,30);
UPDATE trade_order SET cancel_type = CASE cancel_type WHEN 10 THEN 30 WHEN 20 THEN 10 WHEN 40 THEN 20 WHEN 50 THEN 20 WHEN 70 THEN 40 ELSE cancel_type END WHERE cancel_type IN (10,20,40,50,70);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id)
SELECT '拼团过期处理',1,'combinationRecordExpireJob','0 */1 * * * *',tenant_id
FROM (SELECT DISTINCT tenant_id FROM infra_job WHERE deleted=0) tenants
WHERE NOT EXISTS (SELECT 1 FROM infra_job j WHERE j.tenant_id=tenants.tenant_id AND j.handler_name='combinationRecordExpireJob' AND j.deleted=0);
COMMIT;
