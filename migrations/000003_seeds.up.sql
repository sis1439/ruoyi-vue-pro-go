-- Non-sensitive development baseline. Explicit provisioning creates the administrator.
BEGIN;
INSERT INTO system_tenant(id,name,status,package_id,expire_time,account_count,create_time,update_time) VALUES (1,'Isolated mall',0,0,'2099-12-31T00:00:00Z',100,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO system_role(id,name,code,data_scope,status,type,tenant_id,create_time,update_time) VALUES(1,'Mall administrator','tenant_admin',1,0,1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO trade_config(tenant_id,after_sale_refund_reasons,after_sale_return_reasons,brokerage_poster_urls,brokerage_withdraw_types,create_time,update_time) VALUES(1,'["Refund requested"]','["Return requested"]','[]','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO member_config(tenant_id,create_time,update_time) VALUES(1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
-- App is disabled until real callback URLs and a validated channel are provisioned.
INSERT INTO pay_app(app_key,name,status,order_notify_url,refund_notify_url,tenant_id,remark,create_time,update_time) VALUES('mall','Mall',1,'','','1','External blocker: provision merchant channel and callback URLs',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO system_dict_type(name,type,status) VALUES('Common status','common_status',0);
INSERT INTO system_dict_data(label,value,dict_type,sort,status) VALUES('Enabled','0','common_status',0,0),('Disabled','1','common_status',1,0);
INSERT INTO system_menu(id,name,type,sort,parent_id,path,status) VALUES(1,'Mall',1,0,0,'/mall',0);
-- internal/api/router/system.go: RequirePermission(infra:api-access-log:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(100,'infra:api-access-log:export','infra:api-access-log:export',3,100,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:api-access-log:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(101,'infra:api-access-log:query','infra:api-access-log:query',3,101,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:api-error-log:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(102,'infra:api-error-log:export','infra:api-error-log:export',3,102,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:api-error-log:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(103,'infra:api-error-log:query','infra:api-error-log:query',3,103,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:api-error-log:update-process)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(104,'infra:api-error-log:update-process','infra:api-error-log:update-process',3,104,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:api-error-log:update-status)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(105,'infra:api-error-log:update-status','infra:api-error-log:update-status',3,105,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:config:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(106,'infra:config:create','infra:config:create',3,106,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:config:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(107,'infra:config:delete','infra:config:delete',3,107,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:config:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(108,'infra:config:export','infra:config:export',3,108,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:config:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(109,'infra:config:query','infra:config:query',3,109,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:config:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(110,'infra:config:update','infra:config:update',3,110,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:file-config:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(111,'infra:file-config:create','infra:file-config:create',3,111,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:file-config:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(112,'infra:file-config:delete','infra:file-config:delete',3,112,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:file-config:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(113,'infra:file-config:query','infra:file-config:query',3,113,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:file-config:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(114,'infra:file-config:update','infra:file-config:update',3,114,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:file:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(115,'infra:file:create','infra:file:create',3,115,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:file:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(116,'infra:file:delete','infra:file:delete',3,116,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:file:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(117,'infra:file:query','infra:file:query',3,117,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:job:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(118,'infra:job:create','infra:job:create',3,118,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:job:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(119,'infra:job:delete','infra:job:delete',3,119,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:job:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(120,'infra:job:export','infra:job:export',3,120,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:job:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(121,'infra:job:query','infra:job:query',3,121,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:job:trigger)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(122,'infra:job:trigger','infra:job:trigger',3,122,1,0,0);
-- internal/api/router/system.go: RequirePermission(infra:job:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(123,'infra:job:update','infra:job:update',3,123,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:config:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(124,'member:config:query','member:config:query',3,124,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:config:save)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(125,'member:config:save','member:config:save',3,125,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:group:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(126,'member:group:create','member:group:create',3,126,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:group:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(127,'member:group:delete','member:group:delete',3,127,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:group:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(128,'member:group:query','member:group:query',3,128,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:group:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(129,'member:group:update','member:group:update',3,129,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:level:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(130,'member:level:create','member:level:create',3,130,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:level:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(131,'member:level:delete','member:level:delete',3,131,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:level:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(132,'member:level:query','member:level:query',3,132,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:level:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(133,'member:level:update','member:level:update',3,133,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:tag:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(134,'member:tag:create','member:tag:create',3,134,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:tag:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(135,'member:tag:delete','member:tag:delete',3,135,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:tag:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(136,'member:tag:query','member:tag:query',3,136,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:tag:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(137,'member:tag:update','member:tag:update',3,137,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:user:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(138,'member:user:query','member:user:query',3,138,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:user:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(139,'member:user:update','member:user:update',3,139,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:user:update-level)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(140,'member:user:update-level','member:user:update-level',3,140,1,0,0);
-- internal/api/router/member.go: RequirePermission(member:user:update-point)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(141,'member:user:update-point','member:user:update-point',3,141,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:app:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(142,'pay:app:create','pay:app:create',3,142,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:app:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(143,'pay:app:delete','pay:app:delete',3,143,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:app:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(144,'pay:app:query','pay:app:query',3,144,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:app:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(145,'pay:app:update','pay:app:update',3,145,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:channel:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(146,'pay:channel:create','pay:channel:create',3,146,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:channel:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(147,'pay:channel:delete','pay:channel:delete',3,147,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:channel:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(148,'pay:channel:query','pay:channel:query',3,148,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:channel:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(149,'pay:channel:update','pay:channel:update',3,149,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:notify:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(150,'pay:notify:query','pay:notify:query',3,150,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:order:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(151,'pay:order:export','pay:order:export',3,151,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:order:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(152,'pay:order:query','pay:order:query',3,152,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:refund:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(153,'pay:refund:export','pay:refund:export',3,153,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:refund:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(154,'pay:refund:query','pay:refund:query',3,154,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:transfer:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(155,'pay:transfer:query','pay:transfer:query',3,155,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet-recharge-package:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(156,'pay:wallet-recharge-package:create','pay:wallet-recharge-package:create',3,156,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet-recharge-package:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(157,'pay:wallet-recharge-package:delete','pay:wallet-recharge-package:delete',3,157,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet-recharge-package:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(158,'pay:wallet-recharge-package:query','pay:wallet-recharge-package:query',3,158,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet-recharge-package:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(159,'pay:wallet-recharge-package:update','pay:wallet-recharge-package:update',3,159,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet-recharge:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(160,'pay:wallet-recharge:query','pay:wallet-recharge:query',3,160,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet-recharge:refund)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(161,'pay:wallet-recharge:refund','pay:wallet-recharge:refund',3,161,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet-recharge:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(162,'pay:wallet-recharge:update','pay:wallet-recharge:update',3,162,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(163,'pay:wallet:query','pay:wallet:query',3,163,1,0,0);
-- internal/api/router/pay.go: RequirePermission(pay:wallet:update-balance)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(164,'pay:wallet:update-balance','pay:wallet:update-balance',3,164,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:brand:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(165,'product:brand:create','product:brand:create',3,165,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:brand:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(166,'product:brand:delete','product:brand:delete',3,166,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:brand:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(167,'product:brand:query','product:brand:query',3,167,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:brand:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(168,'product:brand:update','product:brand:update',3,168,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:browse-history:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(169,'product:browse-history:query','product:browse-history:query',3,169,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:category:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(170,'product:category:create','product:category:create',3,170,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:category:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(171,'product:category:delete','product:category:delete',3,171,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:category:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(172,'product:category:query','product:category:query',3,172,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:category:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(173,'product:category:update','product:category:update',3,173,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:comment:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(174,'product:comment:create','product:comment:create',3,174,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:comment:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(175,'product:comment:query','product:comment:query',3,175,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:comment:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(176,'product:comment:update','product:comment:update',3,176,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:favorite:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(177,'product:favorite:query','product:favorite:query',3,177,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:property:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(178,'product:property:create','product:property:create',3,178,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:property:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(179,'product:property:delete','product:property:delete',3,179,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:property:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(180,'product:property:query','product:property:query',3,180,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:property:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(181,'product:property:update','product:property:update',3,181,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:spu:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(182,'product:spu:create','product:spu:create',3,182,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:spu:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(183,'product:spu:delete','product:spu:delete',3,183,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:spu:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(184,'product:spu:export','product:spu:export',3,184,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:spu:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(185,'product:spu:query','product:spu:query',3,185,1,0,0);
-- internal/api/router/product.go: RequirePermission(product:spu:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(186,'product:spu:update','product:spu:update',3,186,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:config:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(187,'system:config:create','system:config:create',3,187,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:config:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(188,'system:config:delete','system:config:delete',3,188,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:config:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(189,'system:config:export','system:config:export',3,189,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:config:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(190,'system:config:query','system:config:query',3,190,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:config:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(191,'system:config:update','system:config:update',3,191,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dept:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(192,'system:dept:create','system:dept:create',3,192,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dept:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(193,'system:dept:delete','system:dept:delete',3,193,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dept:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(194,'system:dept:query','system:dept:query',3,194,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dept:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(195,'system:dept:update','system:dept:update',3,195,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dict:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(196,'system:dict:create','system:dict:create',3,196,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dict:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(197,'system:dict:delete','system:dict:delete',3,197,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dict:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(198,'system:dict:export','system:dict:export',3,198,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dict:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(199,'system:dict:query','system:dict:query',3,199,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:dict:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(200,'system:dict:update','system:dict:update',3,200,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:login-log:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(201,'system:login-log:query','system:login-log:query',3,201,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-account:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(202,'system:mail-account:create','system:mail-account:create',3,202,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-account:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(203,'system:mail-account:delete','system:mail-account:delete',3,203,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-account:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(204,'system:mail-account:query','system:mail-account:query',3,204,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-account:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(205,'system:mail-account:update','system:mail-account:update',3,205,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-log:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(206,'system:mail-log:query','system:mail-log:query',3,206,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-template:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(207,'system:mail-template:create','system:mail-template:create',3,207,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-template:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(208,'system:mail-template:delete','system:mail-template:delete',3,208,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-template:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(209,'system:mail-template:query','system:mail-template:query',3,209,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-template:send-mail)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(210,'system:mail-template:send-mail','system:mail-template:send-mail',3,210,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:mail-template:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(211,'system:mail-template:update','system:mail-template:update',3,211,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:menu:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(212,'system:menu:create','system:menu:create',3,212,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:menu:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(213,'system:menu:delete','system:menu:delete',3,213,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:menu:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(214,'system:menu:query','system:menu:query',3,214,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:menu:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(215,'system:menu:update','system:menu:update',3,215,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notice:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(216,'system:notice:create','system:notice:create',3,216,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notice:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(217,'system:notice:delete','system:notice:delete',3,217,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notice:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(218,'system:notice:query','system:notice:query',3,218,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notice:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(219,'system:notice:update','system:notice:update',3,219,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notify-message:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(220,'system:notify-message:query','system:notify-message:query',3,220,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notify-template:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(221,'system:notify-template:create','system:notify-template:create',3,221,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notify-template:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(222,'system:notify-template:delete','system:notify-template:delete',3,222,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notify-template:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(223,'system:notify-template:query','system:notify-template:query',3,223,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notify-template:send-notify)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(224,'system:notify-template:send-notify','system:notify-template:send-notify',3,224,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:notify-template:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(225,'system:notify-template:update','system:notify-template:update',3,225,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:oauth2-client:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(226,'system:oauth2-client:create','system:oauth2-client:create',3,226,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:oauth2-client:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(227,'system:oauth2-client:delete','system:oauth2-client:delete',3,227,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:oauth2-client:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(228,'system:oauth2-client:query','system:oauth2-client:query',3,228,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:oauth2-client:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(229,'system:oauth2-client:update','system:oauth2-client:update',3,229,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:operate-log:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(230,'system:operate-log:query','system:operate-log:query',3,230,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:permission:assign-role-data-scope)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(231,'system:permission:assign-role-data-scope','system:permission:assign-role-data-scope',3,231,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:permission:assign-role-menu)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(232,'system:permission:assign-role-menu','system:permission:assign-role-menu',3,232,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:permission:assign-user-role)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(233,'system:permission:assign-user-role','system:permission:assign-user-role',3,233,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:post:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(234,'system:post:create','system:post:create',3,234,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:post:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(235,'system:post:delete','system:post:delete',3,235,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:post:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(236,'system:post:query','system:post:query',3,236,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:post:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(237,'system:post:update','system:post:update',3,237,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:role:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(238,'system:role:create','system:role:create',3,238,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:role:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(239,'system:role:delete','system:role:delete',3,239,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:role:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(240,'system:role:query','system:role:query',3,240,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:role:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(241,'system:role:update','system:role:update',3,241,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-channel:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(242,'system:sms-channel:create','system:sms-channel:create',3,242,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-channel:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(243,'system:sms-channel:delete','system:sms-channel:delete',3,243,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-channel:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(244,'system:sms-channel:query','system:sms-channel:query',3,244,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-channel:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(245,'system:sms-channel:update','system:sms-channel:update',3,245,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-log:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(246,'system:sms-log:export','system:sms-log:export',3,246,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-log:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(247,'system:sms-log:query','system:sms-log:query',3,247,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-template:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(248,'system:sms-template:create','system:sms-template:create',3,248,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-template:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(249,'system:sms-template:delete','system:sms-template:delete',3,249,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-template:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(250,'system:sms-template:export','system:sms-template:export',3,250,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-template:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(251,'system:sms-template:query','system:sms-template:query',3,251,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-template:send-sms)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(252,'system:sms-template:send-sms','system:sms-template:send-sms',3,252,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:sms-template:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(253,'system:sms-template:update','system:sms-template:update',3,253,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:social-client:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(254,'system:social-client:create','system:social-client:create',3,254,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:social-client:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(255,'system:social-client:delete','system:social-client:delete',3,255,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:social-client:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(256,'system:social-client:query','system:social-client:query',3,256,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:social-client:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(257,'system:social-client:update','system:social-client:update',3,257,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:social-user:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(258,'system:social-user:query','system:social-user:query',3,258,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant-package:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(259,'system:tenant-package:create','system:tenant-package:create',3,259,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant-package:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(260,'system:tenant-package:delete','system:tenant-package:delete',3,260,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant-package:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(261,'system:tenant-package:query','system:tenant-package:query',3,261,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant-package:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(262,'system:tenant-package:update','system:tenant-package:update',3,262,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(263,'system:tenant:create','system:tenant:create',3,263,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(264,'system:tenant:delete','system:tenant:delete',3,264,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(265,'system:tenant:export','system:tenant:export',3,265,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(266,'system:tenant:query','system:tenant:query',3,266,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:tenant:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(267,'system:tenant:update','system:tenant:update',3,267,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:user:create)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(268,'system:user:create','system:user:create',3,268,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:user:delete)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(269,'system:user:delete','system:user:delete',3,269,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:user:export)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(270,'system:user:export','system:user:export',3,270,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:user:import)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(271,'system:user:import','system:user:import',3,271,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:user:query)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(272,'system:user:query','system:user:query',3,272,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:user:update)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(273,'system:user:update','system:user:update',3,273,1,0,0);
-- internal/api/router/system.go: RequirePermission(system:user:update-password)
INSERT INTO system_menu(id,name,permission,type,sort,parent_id,status,visible) VALUES(274,'system:user:update-password','system:user:update-password',3,274,1,0,0);
INSERT INTO system_role_menu(role_id,menu_id,tenant_id) SELECT 1,id,1 FROM system_menu;
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('tradeOrderAutoCancelJob',1,'tradeOrderAutoCancelJob','0 */1 * * * *',1);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('tradeOrderAutoReceiveJob',1,'tradeOrderAutoReceiveJob','0 */1 * * * *',1);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('tradeOrderAutoCommentJob',1,'tradeOrderAutoCommentJob','0 */1 * * * *',1);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('couponExpireJob',1,'couponExpireJob','0 */1 * * * *',1);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('payNotifyJob',1,'payNotifyJob','0 */1 * * * *',1);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('payOrderExpireJob',1,'payOrderExpireJob','0 */1 * * * *',1);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('payOrderSyncJob',1,'payOrderSyncJob','0 */1 * * * *',1);
INSERT INTO infra_job(name,status,handler_name,cron_expression,tenant_id) VALUES('payRefundSyncJob',1,'payRefundSyncJob','0 */1 * * * *',1);
SELECT setval(pg_get_serial_sequence('system_tenant','id'),(SELECT max(id) FROM system_tenant),true);
SELECT setval(pg_get_serial_sequence('system_role','id'),(SELECT max(id) FROM system_role),true);
SELECT setval(pg_get_serial_sequence('system_menu','id'),(SELECT max(id) FROM system_menu),true);
COMMIT;
