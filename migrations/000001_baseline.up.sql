-- Source: internal/model; generated with pinned GORM PostgreSQL dialect.
-- Forward-only baseline; never edit after application.
BEGIN;

-- internal/model/infra_api_access_log.go (*model.InfraApiAccessLog)
CREATE TABLE "infra_api_access_log" (
  "id" bigserial,
  "trace_id" varchar(64),
  "user_id" bigint DEFAULT 0,
  "user_type" smallint DEFAULT 0,
  "application_name" varchar(50) NOT NULL,
  "request_method" varchar(16) NOT NULL,
  "request_url" varchar(255) NOT NULL,
  "request_params" text,
  "response_body" text,
  "user_ip" varchar(50),
  "user_agent" varchar(512),
  "operate_module" varchar(50),
  "operate_name" varchar(50),
  "operate_type" smallint DEFAULT 0,
  "begin_time" timestamptz,
  "end_time" timestamptz,
  "duration" bigint DEFAULT 0,
  "result_code" bigint DEFAULT 0,
  "result_msg" varchar(512),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/infra_api_error_log.go (*model.InfraApiErrorLog)
CREATE TABLE "infra_api_error_log" (
  "id" bigserial,
  "trace_id" varchar(64),
  "user_id" bigint DEFAULT 0,
  "user_type" smallint DEFAULT 0,
  "application_name" varchar(50) NOT NULL,
  "request_method" varchar(16) NOT NULL,
  "request_url" varchar(255) NOT NULL,
  "request_params" text,
  "user_ip" varchar(50),
  "user_agent" varchar(512),
  "exception_time" timestamptz,
  "exception_name" varchar(128),
  "exception_message" text,
  "exception_root_cause_message" text,
  "exception_stack_trace" text,
  "exception_class_name" varchar(512),
  "exception_file_name" varchar(512),
  "exception_method_name" varchar(512),
  "exception_line_number" bigint,
  "process_status" smallint DEFAULT 0,
  "process_time" timestamptz,
  "process_user_id" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/infra_file.go (*model.InfraFileConfig)
CREATE TABLE "infra_file_config" (
  "id" bigserial,
  "name" varchar(63) NOT NULL,
  "storage" integer NOT NULL,
  "master" smallint DEFAULT (0) CHECK ("master" IN (0,1)),
  "config" json,
  "remark" varchar(255),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/infra_file.go (*model.InfraFile)
CREATE TABLE "infra_file" (
  "id" bigserial,
  "config_id" bigint NOT NULL,
  "name" varchar(255),
  "path" varchar(255),
  "url" varchar(1024),
  "type" varchar(63),
  "size" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/infra_job.go (*model.InfraJob)
CREATE TABLE "infra_job" (
  "id" bigserial,
  "name" varchar(32) NOT NULL,
  "status" smallint NOT NULL DEFAULT 0,
  "handler_name" varchar(64) NOT NULL,
  "handler_param" varchar(255),
  "cron_expression" varchar(32) NOT NULL,
  "retry_count" bigint NOT NULL DEFAULT 0,
  "retry_interval" bigint NOT NULL DEFAULT 0,
  "monitor_timeout" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/infra_job_log.go (*model.InfraJobLog)
CREATE TABLE "infra_job_log" (
  "id" bigserial,
  "job_id" bigint NOT NULL,
  "handler_name" varchar(64) NOT NULL,
  "handler_param" varchar(255),
  "execute_index" smallint NOT NULL DEFAULT 1,
  "begin_time" timestamptz NOT NULL,
  "end_time" timestamptz,
  "duration" bigint,
  "status" smallint NOT NULL,
  "result" varchar(4000),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotProductDO)
CREATE TABLE "iot_product" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "product_key" varchar(64) NOT NULL,
  "category_id" bigint,
  "icon" varchar(255),
  "pic_url" varchar(255),
  "description" varchar(255),
  "status" smallint NOT NULL DEFAULT 0,
  "device_type" smallint NOT NULL DEFAULT 0,
  "net_type" smallint NOT NULL DEFAULT 0,
  "location_type" smallint NOT NULL DEFAULT 0,
  "codec_type" varchar(64),
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotDeviceDO)
CREATE TABLE "iot_device" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "device_name" varchar(64) NOT NULL,
  "nickname" varchar(64),
  "serial_number" varchar(64),
  "pic_url" varchar(255),
  "group_ids" JSONB,
  "product_id" bigint NOT NULL,
  "product_key" varchar(64) NOT NULL,
  "device_type" smallint NOT NULL,
  "gateway_id" bigint DEFAULT 0,
  "state" smallint NOT NULL DEFAULT 0,
  "online_time" timestamptz,
  "offline_time" timestamptz,
  "active_time" timestamptz,
  "ip" varchar(64),
  "firmware_id" bigint,
  "device_secret" varchar(64),
  "auth_type" varchar(64),
  "location_type" smallint,
  "latitude" decimal(10,8),
  "longitude" decimal(11,8),
  "area_id" integer,
  "address" varchar(255),
  "config" JSONB,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotThingModelDO)
CREATE TABLE "iot_thing_model" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "id" bigserial,
  "identifier" varchar(64) NOT NULL,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "product_id" bigint NOT NULL,
  "product_key" varchar(64) NOT NULL,
  "type" smallint NOT NULL,
  "property" JSONB,
  "event" JSONB,
  "service" JSONB,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotDeviceGroupDO)
CREATE TABLE "iot_device_group" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "status" smallint NOT NULL DEFAULT 0,
  "description" varchar(255),
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotOtaFirmwareDO)
CREATE TABLE "iot_ota_firmware" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "version" varchar(64) NOT NULL,
  "product_id" bigint NOT NULL,
  "file_url" varchar(255) NOT NULL,
  "file_size" bigint NOT NULL,
  "file_digest_algorithm" varchar(32) NOT NULL,
  "file_digest_value" varchar(64) NOT NULL,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotOtaTaskDO)
CREATE TABLE "iot_ota_task" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "firmware_id" bigint NOT NULL,
  "status" smallint NOT NULL DEFAULT 0,
  "device_scope" smallint NOT NULL,
  "device_total_count" integer NOT NULL DEFAULT 0,
  "device_success_count" integer NOT NULL DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotOtaTaskRecordDO)
CREATE TABLE "iot_ota_task_record" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "firmware_id" bigint NOT NULL,
  "task_id" bigint NOT NULL,
  "device_id" bigint NOT NULL,
  "from_firmware_id" bigint,
  "status" smallint NOT NULL DEFAULT 0,
  "progress" integer NOT NULL DEFAULT 0,
  "description" varchar(255),
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotAlertConfigDO)
CREATE TABLE "iot_alert_config" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "level" smallint NOT NULL,
  "status" smallint NOT NULL DEFAULT 0,
  "scene_rule_ids" JSONB,
  "receive_user_ids" JSONB,
  "receive_types" JSONB,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotAlertRecordDO)
CREATE TABLE "iot_alert_record" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "config_id" bigint NOT NULL,
  "config_name" varchar(64) NOT NULL,
  "config_level" smallint NOT NULL,
  "scene_rule_id" bigint,
  "product_id" bigint NOT NULL,
  "device_id" bigint NOT NULL,
  "device_message" text,
  "process_status" boolean NOT NULL DEFAULT false,
  "process_remark" varchar(255),
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotDataRuleDO)
CREATE TABLE "iot_data_rule" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "status" smallint NOT NULL DEFAULT 0,
  "source_configs" JSONB,
  "sink_ids" JSONB,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotDataSinkDO)
CREATE TABLE "iot_data_sink" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "status" smallint NOT NULL DEFAULT 0,
  "type" smallint NOT NULL,
  "config" JSONB,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotSceneRuleDO)
CREATE TABLE "iot_scene_rule" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "status" smallint NOT NULL DEFAULT 0,
  "triggers" JSONB,
  "actions" JSONB,
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotProductCategoryDO)
CREATE TABLE "iot_product_category" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "sort" integer,
  "status" smallint NOT NULL DEFAULT 0,
  "description" varchar(255),
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotDeviceMessageDO)
CREATE TABLE "iot_device_message" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" varchar(64),
  "report_time" bigint NOT NULL,
  "ts" bigint NOT NULL,
  "device_id" bigint NOT NULL,
  "server_id" varchar(64),
  "upstream" boolean NOT NULL,
  "reply" boolean NOT NULL,
  "identifier" varchar(64),
  "request_id" varchar(64),
  "method" varchar(64),
  "params" text,
  "data" text,
  "code" bigint,
  "msg" varchar(255),
  PRIMARY KEY ("id")
);

-- internal/model/iot.go (*model.IotDevicePropertyDO)
CREATE TABLE "iot_device_property" (
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz NOT NULL,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "id" bigserial,
  "device_id" bigint NOT NULL,
  "identifier" varchar(64) NOT NULL,
  "value" text NOT NULL,
  PRIMARY KEY ("id")
);

-- internal/model/member/config.go (*member.MemberConfig)
CREATE TABLE "member_config" (
  "id" bigserial,
  "point_trade_deduct_enable" smallint DEFAULT (0) CHECK ("point_trade_deduct_enable" IN (0,1)),
  "point_trade_deduct_unit_price" bigint DEFAULT 0,
  "point_trade_deduct_max_price" bigint DEFAULT 0,
  "point_trade_give_point" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/group.go (*member.MemberGroup)
CREATE TABLE "member_group" (
  "id" bigserial,
  "name" text,
  "remark" text,
  "status" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/level.go (*member.MemberLevel)
CREATE TABLE "member_level" (
  "id" bigserial,
  "name" varchar(64) NOT NULL DEFAULT '',
  "level" bigint NOT NULL DEFAULT 0,
  "experience" bigint NOT NULL DEFAULT 0,
  "discount_percent" bigint NOT NULL DEFAULT 100,
  "icon" varchar(255) DEFAULT '',
  "background_url" varchar(255) DEFAULT '',
  "status" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/member_address.go (*member.MemberAddress)
CREATE TABLE "member_address" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "name" varchar(50) NOT NULL,
  "mobile" varchar(20) NOT NULL,
  "area_id" bigint NOT NULL,
  "detail_address" varchar(255) NOT NULL,
  "default_status" smallint NOT NULL DEFAULT (0) CHECK ("default_status" IN (0,1)),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/member_user.go (*member.MemberUser)
CREATE TABLE "member_user" (
  "id" bigserial,
  "mobile" varchar(11),
  "password" varchar(100) DEFAULT '',
  "status" integer DEFAULT 0,
  "register_ip" varchar(32) DEFAULT '',
  "register_terminal" integer DEFAULT 0,
  "login_ip" varchar(32) DEFAULT '',
  "login_date" timestamptz,
  "nickname" varchar(30) DEFAULT '',
  "avatar" varchar(255) DEFAULT '',
  "name" varchar(30) DEFAULT '',
  "sex" integer DEFAULT 0,
  "birthday" timestamptz,
  "area_id" integer,
  "mark" varchar(255) DEFAULT '',
  "point" integer DEFAULT 0,
  "tag_ids" varchar(255),
  "level_id" bigint,
  "experience" integer DEFAULT 0,
  "group_id" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/point.go (*member.MemberPointRecord)
CREATE TABLE "member_point_record" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "biz_id" varchar(64),
  "biz_type" bigint NOT NULL,
  "title" varchar(64) NOT NULL,
  "description" varchar(255) DEFAULT '',
  "point" bigint NOT NULL,
  "total_point" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/signin.go (*member.MemberSignInConfig)
CREATE TABLE "member_sign_in_config" (
  "id" bigserial,
  "day" bigint,
  "point" bigint,
  "experience" bigint,
  "status" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/signin.go (*member.MemberSignInRecord)
CREATE TABLE "member_sign_in_record" (
  "id" bigserial,
  "user_id" bigint,
  "day" bigint,
  "point" bigint,
  "experience" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/member/tag.go (*member.MemberTag)
CREATE TABLE "member_tag" (
  "id" bigserial,
  "name" varchar(30) NOT NULL DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_app.go (*pay.PayApp)
CREATE TABLE "pay_app" (
  "id" bigserial,
  "app_key" varchar(64) NOT NULL,
  "name" varchar(64) NOT NULL,
  "status" bigint NOT NULL DEFAULT 0,
  "remark" varchar(255) DEFAULT '',
  "order_notify_url" varchar(1024) NOT NULL,
  "refund_notify_url" varchar(1024) NOT NULL,
  "transfer_notify_url" varchar(1024) DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_channel.go (*pay.PayChannel)
CREATE TABLE "pay_channel" (
  "id" bigserial,
  "code" varchar(32) NOT NULL,
  "status" bigint NOT NULL DEFAULT 0,
  "fee_rate" decimal DEFAULT 0,
  "remark" varchar(255) DEFAULT '',
  "app_id" bigint NOT NULL,
  "config" json,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_notify.go (*pay.PayNotifyTask)
CREATE TABLE "pay_notify_task" (
  "id" bigserial,
  "app_id" bigint,
  "type" bigint,
  "data_id" bigint,
  "merchant_order_id" text,
  "merchant_refund_id" text,
  "merchant_transfer_id" text,
  "status" bigint,
  "next_notify_time" timestamptz,
  "last_execute_time" timestamptz,
  "notify_times" bigint,
  "max_notify_times" bigint,
  "notify_url" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_notify.go (*pay.PayNotifyLog)
CREATE TABLE "pay_notify_log" (
  "id" bigserial,
  "task_id" bigint,
  "notify_times" bigint,
  "response" text,
  "status" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_order.go (*pay.PayOrder)
CREATE TABLE "pay_order" (
  "id" bigserial,
  "app_id" bigint,
  "channel_id" bigint,
  "channel_code" text,
  "merchant_order_id" text,
  "subject" text,
  "body" text,
  "notify_url" text,
  "price" bigint,
  "channel_fee_rate" decimal,
  "channel_fee_price" bigint,
  "status" bigint,
  "user_ip" text,
  "expire_time" timestamptz,
  "success_time" timestamptz,
  "extension_id" bigint,
  "no" text,
  "refund_price" bigint,
  "channel_user_id" text,
  "channel_order_no" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_order_extension.go (*pay.PayOrderExtension)
CREATE TABLE "pay_order_extension" (
  "id" bigserial,
  "no" text,
  "order_id" bigint,
  "channel_id" bigint,
  "channel_code" text,
  "user_ip" text,
  "status" bigint,
  "channel_extras" text,
  "channel_error_code" text,
  "channel_error_msg" text,
  "channel_notify_data" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_refund.go (*pay.PayRefund)
CREATE TABLE "pay_refund" (
  "id" bigserial,
  "no" text,
  "app_id" bigint,
  "channel_id" bigint,
  "channel_code" text,
  "order_id" bigint,
  "order_no" text,
  "user_id" bigint,
  "user_type" bigint,
  "merchant_order_id" text,
  "merchant_refund_id" text,
  "notify_url" text,
  "status" bigint,
  "pay_price" bigint,
  "refund_price" bigint,
  "reason" text,
  "user_ip" text,
  "channel_order_no" text,
  "channel_refund_no" text,
  "success_time" timestamptz,
  "channel_error_code" text,
  "channel_error_msg" text,
  "channel_notify_data" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_wallet.go (*pay.PayWallet)
CREATE TABLE "pay_wallet" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "user_type" bigint NOT NULL DEFAULT 0,
  "balance" bigint NOT NULL DEFAULT 0,
  "total_expense" bigint NOT NULL DEFAULT 0,
  "total_recharge" bigint NOT NULL DEFAULT 0,
  "freeze_price" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_wallet_recharge.go (*pay.PayWalletRecharge)
CREATE TABLE "pay_wallet_recharge" (
  "id" bigserial,
  "wallet_id" bigint NOT NULL,
  "total_price" bigint NOT NULL DEFAULT 0,
  "pay_price" bigint NOT NULL DEFAULT 0,
  "bonus_price" bigint NOT NULL DEFAULT 0,
  "package_id" bigint DEFAULT 0,
  "pay_status" boolean NOT NULL DEFAULT false,
  "pay_order_id" bigint,
  "pay_channel_code" varchar(32),
  "pay_time" timestamptz,
  "refund_status" bigint NOT NULL DEFAULT 0,
  "pay_refund_id" bigint DEFAULT 0,
  "refund_total_price" bigint NOT NULL DEFAULT 0,
  "refund_pay_price" bigint NOT NULL DEFAULT 0,
  "refund_bonus_price" bigint NOT NULL DEFAULT 0,
  "refund_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_wallet_recharge_package.go (*pay.PayWalletRechargePackage)
CREATE TABLE "pay_wallet_recharge_package" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "pay_price" bigint NOT NULL DEFAULT 0,
  "bonus_price" bigint NOT NULL DEFAULT 0,
  "status" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/pay_wallet_transaction.go (*pay.PayWalletTransaction)
CREATE TABLE "pay_wallet_transaction" (
  "id" bigserial,
  "wallet_id" bigint NOT NULL,
  "biz_type" bigint NOT NULL,
  "biz_id" varchar(64) NOT NULL,
  "no" varchar(64) NOT NULL,
  "title" varchar(128) NOT NULL,
  "price" bigint NOT NULL DEFAULT 0,
  "balance" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/pay/transfer.go (*pay.PayTransfer)
CREATE TABLE "pay_transfer" (
  "id" bigserial,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  "no" varchar(64) NOT NULL,
  "app_id" bigint NOT NULL,
  "channel_id" bigint NOT NULL,
  "channel_code" varchar(32) NOT NULL,
  "merchant_transfer_id" varchar(64),
  "subject" varchar(512) NOT NULL,
  "price" bigint NOT NULL,
  "type" integer,
  "user_account" varchar(64) NOT NULL,
  "user_name" varchar(64),
  "status" bigint NOT NULL,
  "success_time" timestamptz,
  "notify_url" varchar(128),
  "user_ip" varchar(50),
  "channel_extras" text,
  "channel_transfer_no" varchar(64),
  "channel_error_code" varchar(128),
  "channel_error_msg" varchar(256),
  "channel_notify_data" varchar(4096),
  "channel_package_info" varchar(4096),
  PRIMARY KEY ("id")
);

-- internal/model/product/browse_history.go (*product.ProductBrowseHistory)
CREATE TABLE "product_browse_history" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "user_deleted" smallint CHECK ("user_deleted" IN (0,1)),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_brand.go (*product.ProductBrand)
CREATE TABLE "product_brand" (
  "id" bigserial,
  "name" varchar(255) NOT NULL,
  "pic_url" varchar(255) DEFAULT '',
  "sort" bigint DEFAULT 0,
  "description" varchar(1024) DEFAULT '',
  "status" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_category.go (*product.ProductCategory)
CREATE TABLE "product_category" (
  "id" bigserial,
  "parent_id" bigint NOT NULL DEFAULT 0,
  "name" varchar(255) NOT NULL,
  "pic_url" varchar(255) DEFAULT '',
  "sort" integer DEFAULT 0,
  "status" integer DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_comment.go (*product.ProductComment)
CREATE TABLE "product_comment" (
  "id" bigserial,
  "user_id" bigint,
  "user_nickname" varchar(64),
  "user_avatar" varchar(255),
  "anonymous" smallint CHECK ("anonymous" IN (0,1)),
  "order_id" bigint,
  "order_item_id" bigint,
  "spu_id" bigint,
  "spu_name" varchar(255),
  "sku_id" bigint,
  "sku_pic_url" varchar(255),
  "sku_properties" text,
  "visible" smallint CHECK ("visible" IN (0,1)),
  "scores" bigint,
  "description_scores" bigint,
  "benefit_scores" bigint,
  "content" text,
  "pic_urls" text,
  "reply_status" smallint CHECK ("reply_status" IN (0,1)),
  "reply_user_id" bigint,
  "reply_content" text,
  "reply_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_favorite.go (*product.ProductFavorite)
CREATE TABLE "product_favorite" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_property.go (*product.ProductProperty)
CREATE TABLE "product_property" (
  "id" bigserial,
  "name" varchar(255) NOT NULL,
  "remark" varchar(500) DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_property_value.go (*product.ProductPropertyValue)
CREATE TABLE "product_property_value" (
  "id" bigserial,
  "property_id" bigint NOT NULL,
  "name" varchar(255) NOT NULL,
  "remark" varchar(500) DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_sku.go (*product.ProductSku)
CREATE TABLE "product_sku" (
  "id" bigserial,
  "spu_id" bigint NOT NULL,
  "properties" json,
  "price" bigint DEFAULT 0,
  "market_price" bigint DEFAULT 0,
  "cost_price" bigint DEFAULT 0,
  "bar_code" varchar(64) DEFAULT '',
  "pic_url" varchar(255) DEFAULT '',
  "stock" bigint DEFAULT 0,
  "weight" decimal DEFAULT 0,
  "volume" decimal DEFAULT 0,
  "first_brokerage_price" bigint DEFAULT 0,
  "second_brokerage_price" bigint DEFAULT 0,
  "sales_count" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/product_spu.go (*product.ProductSpu)
CREATE TABLE "product_spu" (
  "id" bigserial,
  "name" varchar(255) NOT NULL,
  "keyword" varchar(255) DEFAULT '',
  "introduction" varchar(1024) DEFAULT '',
  "description" text,
  "category_id" bigint NOT NULL,
  "brand_id" bigint DEFAULT 0,
  "pic_url" varchar(255) NOT NULL,
  "slider_pic_urls" json,
  "sort" bigint DEFAULT 0,
  "status" bigint DEFAULT 0,
  "spec_type" smallint DEFAULT (0) CHECK ("spec_type" IN (0,1)),
  "price" bigint DEFAULT 0,
  "market_price" bigint DEFAULT 0,
  "cost_price" bigint DEFAULT 0,
  "stock" bigint DEFAULT 0,
  "delivery_types" text,
  "delivery_template_id" bigint DEFAULT 0,
  "give_integral" bigint DEFAULT 0,
  "sub_commission_type" smallint DEFAULT (0) CHECK ("sub_commission_type" IN (0,1)),
  "sales_count" bigint DEFAULT 0,
  "virtual_sales_count" bigint DEFAULT 0,
  "browse_count" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/product/statistics.go (*product.ProductStatistics)
CREATE TABLE "product_statistics" (
  "id" bigserial,
  "time" timestamptz NOT NULL,
  "spu_id" bigint NOT NULL,
  "browse_count" bigint DEFAULT 0,
  "browse_user_count" bigint DEFAULT 0,
  "favorite_count" bigint DEFAULT 0,
  "cart_count" bigint DEFAULT 0,
  "order_count" bigint DEFAULT 0,
  "order_pay_count" bigint DEFAULT 0,
  "order_pay_price" bigint DEFAULT 0,
  "after_sale_count" bigint DEFAULT 0,
  "after_sale_refund_price" bigint DEFAULT 0,
  "browse_convert_percent" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/article.go (*promotion.PromotionArticleCategory)
CREATE TABLE "promotion_article_category" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "pic_url" varchar(255),
  "sort" bigint NOT NULL DEFAULT 0,
  "status" smallint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/article.go (*promotion.PromotionArticle)
CREATE TABLE "promotion_article" (
  "id" bigserial,
  "category_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL DEFAULT 0,
  "title" varchar(64) NOT NULL,
  "author" varchar(64),
  "pic_url" varchar(255),
  "introduction" varchar(255),
  "browse_count" bigint NOT NULL DEFAULT 0,
  "sort" bigint NOT NULL DEFAULT 0,
  "status" smallint NOT NULL DEFAULT 0,
  "recommend_hot" smallint NOT NULL DEFAULT (0) CHECK ("recommend_hot" IN (0,1)),
  "recommend_banner" smallint NOT NULL DEFAULT (0) CHECK ("recommend_banner" IN (0,1)),
  "content" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/banner.go (*promotion.PromotionBanner)
CREATE TABLE "promotion_banner" (
  "id" bigserial,
  "title" varchar(64) NOT NULL,
  "pic_url" varchar(255) NOT NULL,
  "url" varchar(255) NOT NULL,
  "status" smallint NOT NULL DEFAULT 0,
  "sort" bigint NOT NULL DEFAULT 0,
  "position" smallint NOT NULL DEFAULT 1,
  "memo" varchar(255),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/bargain.go (*promotion.PromotionBargainActivity)
CREATE TABLE "promotion_bargain_activity" (
  "id" bigserial,
  "name" varchar(255) NOT NULL,
  "start_time" timestamptz NOT NULL,
  "end_time" timestamptz NOT NULL,
  "status" bigint NOT NULL DEFAULT 0,
  "spu_id" bigint NOT NULL,
  "sku_id" bigint NOT NULL,
  "bargain_first_price" bigint,
  "bargain_min_price" bigint,
  "stock" bigint DEFAULT 0,
  "total_stock" bigint DEFAULT 0,
  "help_max_count" bigint,
  "bargain_count" bigint,
  "total_limit_count" bigint,
  "random_min_price" bigint,
  "random_max_price" bigint,
  "remark" varchar(255) DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/bargain.go (*promotion.PromotionBargainRecord)
CREATE TABLE "promotion_bargain_record" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "activity_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "sku_id" bigint NOT NULL,
  "bargain_first_price" bigint,
  "bargain_price" bigint,
  "status" bigint NOT NULL DEFAULT 0,
  "end_time" timestamptz,
  "order_id" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/bargain.go (*promotion.PromotionBargainHelp)
CREATE TABLE "promotion_bargain_help" (
  "id" bigserial,
  "activity_id" bigint NOT NULL,
  "record_id" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "reduce_price" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/combination.go (*promotion.PromotionCombinationActivity)
CREATE TABLE "promotion_combination_activity" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "spu_id" bigint NOT NULL,
  "total_limit_count" bigint NOT NULL,
  "single_limit_count" bigint NOT NULL,
  "start_time" timestamptz NOT NULL,
  "end_time" timestamptz NOT NULL,
  "user_size" bigint NOT NULL,
  "virtual_group" boolean NOT NULL DEFAULT false,
  "status" bigint NOT NULL,
  "limit_duration" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/combination.go (*promotion.PromotionCombinationProduct)
CREATE TABLE "promotion_combination_product" (
  "id" bigserial,
  "activity_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "sku_id" bigint NOT NULL,
  "combination_price" bigint NOT NULL,
  "activity_status" bigint NOT NULL,
  "activity_start_time" timestamptz,
  "activity_end_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/combination.go (*promotion.PromotionCombinationRecord)
CREATE TABLE "promotion_combination_record" (
  "id" bigserial,
  "activity_id" bigint NOT NULL,
  "combination_price" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "spu_name" varchar(255),
  "pic_url" varchar(255),
  "sku_id" bigint NOT NULL,
  "count" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "nickname" varchar(64),
  "avatar" varchar(255),
  "head_id" bigint NOT NULL,
  "status" bigint NOT NULL,
  "order_id" bigint NOT NULL,
  "user_size" bigint NOT NULL,
  "user_count" bigint NOT NULL,
  "virtual_group" smallint CHECK ("virtual_group" IN (0,1)),
  "expire_time" timestamptz,
  "start_time" timestamptz,
  "end_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/coupon.go (*promotion.PromotionCoupon)
CREATE TABLE "promotion_coupon" (
  "id" bigserial,
  "template_id" bigint NOT NULL,
  "name" varchar(64) NOT NULL,
  "status" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "take_type" bigint NOT NULL,
  "use_price" bigint NOT NULL,
  "valid_start_time" timestamptz NOT NULL,
  "valid_end_time" timestamptz NOT NULL,
  "product_scope" bigint NOT NULL,
  "product_scope_values" varchar(255),
  "discount_type" bigint NOT NULL,
  "discount_price" bigint,
  "discount_percent" bigint,
  "discount_limit_price" bigint,
  "use_order_id" bigint,
  "use_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/coupon_template.go (*promotion.PromotionCouponTemplate)
CREATE TABLE "promotion_coupon_template" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "description" varchar(255),
  "status" bigint NOT NULL,
  "total_count" bigint NOT NULL,
  "take_count" bigint NOT NULL DEFAULT 0,
  "take_limit_count" bigint NOT NULL,
  "take_type" bigint NOT NULL,
  "use_price" bigint NOT NULL,
  "product_scope" bigint NOT NULL,
  "product_scope_values" varchar(255),
  "validity_type" bigint NOT NULL,
  "valid_start_time" timestamptz,
  "valid_end_time" timestamptz,
  "fixed_start_term" bigint,
  "fixed_end_term" bigint,
  "discount_type" bigint NOT NULL,
  "discount_price" bigint,
  "discount_percent" bigint,
  "discount_limit_price" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/discount.go (*promotion.PromotionDiscountActivity)
CREATE TABLE "promotion_discount_activity" (
  "id" bigserial,
  "name" text,
  "status" bigint,
  "start_time" timestamptz,
  "end_time" timestamptz,
  "remark" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/discount.go (*promotion.PromotionDiscountProduct)
CREATE TABLE "promotion_discount_product" (
  "id" bigserial,
  "activity_id" bigint,
  "spu_id" bigint,
  "sku_id" bigint,
  "discount_type" bigint,
  "discount_percent" bigint,
  "discount_price" bigint,
  "activity_name" text,
  "activity_status" bigint,
  "activity_start_time" timestamptz,
  "activity_end_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/diy.go (*promotion.PromotionDiyTemplate)
CREATE TABLE "promotion_diy_template" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "preview_pic_urls" varchar(2000),
  "property" JSONB,
  "remark" varchar(255),
  "used" smallint NOT NULL DEFAULT (0) CHECK ("used" IN (0,1)),
  "used_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/diy.go (*promotion.PromotionDiyPage)
CREATE TABLE "promotion_diy_page" (
  "id" bigserial,
  "template_id" bigint NOT NULL,
  "name" varchar(64) NOT NULL,
  "remark" varchar(255),
  "preview_pic_urls" varchar(2000),
  "property" JSONB,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/kefu.go (*promotion.PromotionKefuConversation)
CREATE TABLE "promotion_kefu_conversation" (
  "id" bigserial,
  "user_id" bigint,
  "last_message_time" timestamptz,
  "last_message_content" text,
  "last_message_content_type" bigint,
  "admin_pinned" smallint CHECK ("admin_pinned" IN (0,1)),
  "user_deleted" smallint CHECK ("user_deleted" IN (0,1)),
  "admin_deleted" smallint CHECK ("admin_deleted" IN (0,1)),
  "admin_unread_message_count" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/kefu.go (*promotion.PromotionKefuMessage)
CREATE TABLE "promotion_kefu_message" (
  "id" bigserial,
  "conversation_id" bigint,
  "sender_id" bigint,
  "sender_type" bigint,
  "receiver_id" bigint,
  "receiver_type" bigint,
  "content_type" bigint,
  "content" text,
  "read_status" smallint CHECK ("read_status" IN (0,1)),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/point_activity.go (*promotion.PromotionPointActivity)
CREATE TABLE "promotion_point_activity" (
  "id" bigserial,
  "spu_id" bigint NOT NULL,
  "status" bigint NOT NULL,
  "remark" varchar(255) DEFAULT '',
  "sort" bigint NOT NULL DEFAULT 0,
  "stock" bigint NOT NULL DEFAULT 0,
  "total_stock" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/point_product.go (*promotion.PromotionPointProduct)
CREATE TABLE "promotion_point_product" (
  "id" bigserial,
  "activity_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "sku_id" bigint NOT NULL,
  "count" bigint NOT NULL DEFAULT 0,
  "point" bigint NOT NULL DEFAULT 0,
  "price" bigint NOT NULL DEFAULT 0,
  "stock" bigint NOT NULL DEFAULT 0,
  "activity_status" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/reward_activity.go (*promotion.PromotionRewardActivity)
CREATE TABLE "promotion_reward_activity" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "status" bigint NOT NULL DEFAULT 0,
  "start_time" timestamptz NOT NULL,
  "end_time" timestamptz NOT NULL,
  "product_scope" bigint NOT NULL DEFAULT 1,
  "product_scope_values" json,
  "condition_type" bigint NOT NULL DEFAULT 1,
  "rules" json NOT NULL,
  "remark" varchar(255),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/seckill.go (*promotion.PromotionSeckillActivity)
CREATE TABLE "promotion_seckill_activity" (
  "id" bigserial,
  "spu_id" bigint NOT NULL,
  "name" varchar(255) NOT NULL,
  "status" bigint NOT NULL DEFAULT 0,
  "remark" varchar(255) DEFAULT '',
  "start_time" timestamptz NOT NULL,
  "end_time" timestamptz NOT NULL,
  "sort" bigint NOT NULL DEFAULT 0,
  "config_ids" varchar(255),
  "total_limit_count" bigint DEFAULT 0,
  "single_limit_count" bigint DEFAULT 0,
  "stock" bigint DEFAULT 0,
  "total_stock" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/seckill.go (*promotion.PromotionSeckillProduct)
CREATE TABLE "promotion_seckill_product" (
  "id" bigserial,
  "activity_id" bigint NOT NULL,
  "config_ids" varchar(255),
  "spu_id" bigint NOT NULL,
  "sku_id" bigint NOT NULL,
  "seckill_price" bigint NOT NULL DEFAULT 0,
  "stock" bigint NOT NULL DEFAULT 0,
  "activity_status" bigint NOT NULL DEFAULT 0,
  "activity_start_time" timestamptz NOT NULL,
  "activity_end_time" timestamptz NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/promotion/seckill.go (*promotion.PromotionSeckillConfig)
CREATE TABLE "promotion_seckill_config" (
  "id" bigserial,
  "name" varchar(255) NOT NULL,
  "start_time" varchar(10) NOT NULL,
  "end_time" varchar(10) NOT NULL,
  "slider_pic_urls" JSONB,
  "status" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_config.go (*model.SystemConfig)
CREATE TABLE "infra_config" (
  "id" bigserial,
  "category" varchar(50) NOT NULL,
  "name" varchar(100) NOT NULL,
  "config_key" varchar(100) NOT NULL,
  "value" varchar(500) NOT NULL,
  "type" smallint NOT NULL DEFAULT 1,
  "visible" smallint NOT NULL DEFAULT (1) CHECK ("visible" IN (0,1)),
  "remark" varchar(500),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_dept.go (*model.SystemDept)
CREATE TABLE "system_dept" (
  "id" bigserial,
  "name" text NOT NULL DEFAULT '',
  "parent_id" bigint NOT NULL DEFAULT 0,
  "sort" integer NOT NULL DEFAULT 0,
  "leader_user_id" bigint DEFAULT 0,
  "phone" text DEFAULT '',
  "email" text DEFAULT '',
  "status" integer NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_dept.go (*model.SystemPost)
CREATE TABLE "system_post" (
  "id" bigserial,
  "name" text NOT NULL DEFAULT '',
  "code" text NOT NULL DEFAULT '',
  "sort" integer NOT NULL DEFAULT 0,
  "status" integer NOT NULL DEFAULT 0,
  "remark" text DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_dict.go (*model.SystemDictType)
CREATE TABLE "system_dict_type" (
  "id" bigserial,
  "name" text NOT NULL DEFAULT '',
  "type" text NOT NULL DEFAULT '',
  "status" integer NOT NULL DEFAULT 0,
  "remark" text DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  PRIMARY KEY ("id")
);

-- internal/model/system_dict.go (*model.SystemDictData)
CREATE TABLE "system_dict_data" (
  "id" bigserial,
  "sort" integer NOT NULL DEFAULT 0,
  "label" text NOT NULL DEFAULT '',
  "value" text NOT NULL DEFAULT '',
  "dict_type" text NOT NULL DEFAULT '',
  "status" integer NOT NULL DEFAULT 0,
  "color_type" text DEFAULT '',
  "css_class" text DEFAULT '',
  "remark" text DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  PRIMARY KEY ("id")
);

-- internal/model/system_login_log.go (*model.SystemLoginLog)
CREATE TABLE "system_login_log" (
  "id" bigserial,
  "log_type" bigint NOT NULL,
  "trace_id" varchar(64),
  "user_id" bigint NOT NULL DEFAULT 0,
  "user_type" smallint NOT NULL,
  "username" varchar(50) NOT NULL DEFAULT '',
  "result" smallint NOT NULL,
  "user_ip" varchar(50) NOT NULL DEFAULT '',
  "user_agent" varchar(512) NOT NULL DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_mail_account.go (*model.SystemMailAccount)
CREATE TABLE "system_mail_account" (
  "id" bigserial,
  "mail" text NOT NULL,
  "username" text NOT NULL,
  "password" text NOT NULL,
  "host" text NOT NULL,
  "port" bigint NOT NULL,
  "ssl_enable" smallint NOT NULL DEFAULT (0) CHECK ("ssl_enable" IN (0,1)),
  "starttls_enable" smallint NOT NULL DEFAULT (0) CHECK ("starttls_enable" IN (0,1)),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_mail_log.go (*model.SystemMailLog)
CREATE TABLE "system_mail_log" (
  "id" bigserial,
  "user_id" bigint,
  "user_type" bigint,
  "to_mails" text NOT NULL,
  "cc_mails" text,
  "bcc_mails" text,
  "account_id" bigint NOT NULL,
  "from_mail" text NOT NULL,
  "template_id" bigint NOT NULL,
  "template_code" text NOT NULL,
  "template_nickname" text,
  "template_title" text NOT NULL,
  "template_content" text NOT NULL,
  "template_params" text NOT NULL,
  "send_status" bigint NOT NULL,
  "send_time" timestamptz,
  "send_message_id" text,
  "send_exception" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_mail_template.go (*model.SystemMailTemplate)
CREATE TABLE "system_mail_template" (
  "id" bigserial,
  "name" text NOT NULL,
  "code" text NOT NULL,
  "account_id" bigint NOT NULL,
  "nickname" text,
  "title" text NOT NULL,
  "content" text NOT NULL,
  "params" text,
  "status" bigint NOT NULL DEFAULT 0,
  "remark" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_menu.go (*model.SystemMenu)
CREATE TABLE "system_menu" (
  "id" bigserial,
  "name" text NOT NULL,
  "permission" text DEFAULT '',
  "type" integer NOT NULL,
  "sort" integer NOT NULL DEFAULT 0,
  "parent_id" bigint NOT NULL DEFAULT 0,
  "path" text DEFAULT '',
  "icon" text DEFAULT '',
  "component" text DEFAULT '',
  "component_name" text DEFAULT '',
  "status" integer NOT NULL DEFAULT 0,
  "visible" smallint NOT NULL DEFAULT (1) CHECK ("visible" IN (0,1)),
  "keep_alive" smallint NOT NULL DEFAULT (1) CHECK ("keep_alive" IN (0,1)),
  "always_show" smallint NOT NULL DEFAULT (1) CHECK ("always_show" IN (0,1)),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  PRIMARY KEY ("id")
);

-- internal/model/system_notice.go (*model.SystemNotice)
CREATE TABLE "system_notice" (
  "id" bigserial,
  "title" text NOT NULL,
  "type" integer NOT NULL,
  "content" text NOT NULL,
  "status" integer NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_notify_message.go (*model.SystemNotifyMessage)
CREATE TABLE "system_notify_message" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "user_type" bigint NOT NULL,
  "template_id" bigint NOT NULL,
  "template_code" text NOT NULL,
  "template_nickname" text,
  "template_content" text NOT NULL,
  "template_type" bigint NOT NULL,
  "template_params" text NOT NULL,
  "read_status" smallint NOT NULL DEFAULT (0) CHECK ("read_status" IN (0,1)),
  "read_time" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_notify_template.go (*model.SystemNotifyTemplate)
CREATE TABLE "system_notify_template" (
  "id" bigserial,
  "name" text NOT NULL,
  "code" text NOT NULL,
  "nickname" text,
  "content" text NOT NULL,
  "type" bigint NOT NULL,
  "params" text,
  "status" bigint NOT NULL DEFAULT 0,
  "remark" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_oauth2_client.go (*model.SystemOAuth2Client)
CREATE TABLE "system_oauth2_client" (
  "id" bigserial,
  "client_id" text NOT NULL,
  "client_secret" text NOT NULL,
  "name" text NOT NULL,
  "logo" text,
  "description" text,
  "status" bigint NOT NULL DEFAULT 0,
  "access_token_validity_seconds" bigint NOT NULL,
  "refresh_token_validity_seconds" bigint NOT NULL,
  "redirect_uris" text,
  "authorized_grant_types" text NOT NULL,
  "scopes" text NOT NULL,
  "auto_approve_scopes" text,
  "authorities" text,
  "resource_ids" text,
  "additional_information" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_operate_log.go (*model.SystemOperateLog)
CREATE TABLE "system_operate_log" (
  "id" bigserial,
  "trace_id" varchar(64),
  "user_id" bigint NOT NULL DEFAULT 0,
  "user_type" smallint NOT NULL,
  "type" varchar(50) NOT NULL DEFAULT '',
  "sub_type" varchar(50) NOT NULL DEFAULT '',
  "biz_id" bigint NOT NULL DEFAULT 0,
  "action" varchar(2000) NOT NULL DEFAULT '',
  "extra" varchar(2000) NOT NULL DEFAULT '',
  "request_method" varchar(16) NOT NULL DEFAULT '',
  "request_url" varchar(255) NOT NULL DEFAULT '',
  "user_ip" varchar(50) NOT NULL DEFAULT '',
  "user_agent" varchar(512) NOT NULL DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_role.go (*model.SystemRole)
CREATE TABLE "system_role" (
  "id" bigserial,
  "name" text NOT NULL,
  "code" text NOT NULL,
  "sort" integer,
  "data_scope" integer NOT NULL DEFAULT 1,
  "data_scope_dept_ids" text,
  "status" integer NOT NULL,
  "type" integer NOT NULL DEFAULT 1,
  "remark" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_role_menu.go (*model.SystemRoleMenu)
CREATE TABLE "system_role_menu" (
  "id" bigserial,
  "role_id" bigint NOT NULL,
  "menu_id" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_sms.go (*model.SystemSmsChannel)
CREATE TABLE "system_sms_channel" (
  "id" bigserial,
  "signature" varchar(10) NOT NULL,
  "code" varchar(63) NOT NULL,
  "status" integer NOT NULL,
  "remark" varchar(255),
  "api_key" varchar(63) NOT NULL,
  "api_secret" varchar(63),
  "callback_url" varchar(255),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_sms.go (*model.SystemSmsTemplate)
CREATE TABLE "system_sms_template" (
  "id" bigserial,
  "type" integer NOT NULL,
  "status" integer NOT NULL,
  "code" varchar(63) NOT NULL,
  "name" varchar(63) NOT NULL,
  "content" varchar(255) NOT NULL,
  "params" JSONB,
  "remark" varchar(255),
  "api_template_id" varchar(63) NOT NULL,
  "channel_id" bigint NOT NULL,
  "channel_code" varchar(63) NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_sms.go (*model.SystemSmsLog)
CREATE TABLE "system_sms_log" (
  "id" bigserial,
  "channel_id" bigint NOT NULL,
  "channel_code" varchar(63) NOT NULL,
  "template_id" bigint NOT NULL,
  "template_code" varchar(63) NOT NULL,
  "template_type" integer NOT NULL,
  "template_content" varchar(255) NOT NULL,
  "template_params" text,
  "api_template_id" varchar(63) NOT NULL,
  "mobile" varchar(11) NOT NULL,
  "user_id" bigint,
  "user_type" integer,
  "send_status" integer NOT NULL DEFAULT 0,
  "send_time" timestamptz,
  "api_send_code" varchar(63),
  "api_send_msg" varchar(255),
  "api_request_id" varchar(63),
  "api_serial_no" varchar(63),
  "receive_status" integer NOT NULL DEFAULT 0,
  "receive_time" timestamptz,
  "api_receive_code" varchar(63),
  "api_receive_msg" varchar(255),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_sms.go (*model.SystemSmsCode)
CREATE TABLE "system_sms_code" (
  "id" bigserial,
  "mobile" varchar(11) NOT NULL,
  "code" varchar(6) NOT NULL,
  "scene" integer NOT NULL,
  "used" boolean NOT NULL DEFAULT false,
  "used_time" timestamptz,
  "today_index" integer NOT NULL DEFAULT 1,
  "create_ip" varchar(30),
  "used_ip" varchar(30),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_social.go (*model.SocialUser)
CREATE TABLE "system_social_user" (
  "id" bigserial,
  "type" bigint NOT NULL,
  "openid" text NOT NULL,
  "token" text,
  "raw_token_info" text,
  "nickname" text NOT NULL,
  "avatar" text,
  "raw_user_info" text,
  "code" text,
  "state" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_social.go (*model.SocialUserBind)
CREATE TABLE "system_social_user_bind" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "user_type" bigint NOT NULL,
  "social_type" bigint NOT NULL,
  "social_user_id" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_social.go (*model.SocialClient)
CREATE TABLE "system_social_client" (
  "id" bigserial,
  "name" text NOT NULL,
  "social_type" bigint NOT NULL,
  "user_type" bigint NOT NULL,
  "client_id" text NOT NULL,
  "client_secret" text NOT NULL,
  "agent_id" text,
  "status" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_tenant.go (*model.SystemTenant)
CREATE TABLE "system_tenant" (
  "id" bigserial,
  "name" text NOT NULL,
  "contact_user_id" bigint,
  "contact_name" text,
  "contact_mobile" text,
  "status" integer NOT NULL DEFAULT 0,
  "website" text,
  "package_id" bigint,
  "expire_time" timestamptz,
  "account_count" integer,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  PRIMARY KEY ("id")
);

-- internal/model/system_tenant_package.go (*model.SystemTenantPackage)
CREATE TABLE "system_tenant_package" (
  "id" bigserial,
  "name" text NOT NULL,
  "status" integer NOT NULL DEFAULT 0,
  "menu_ids" text,
  "remark" text DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  PRIMARY KEY ("id")
);

-- internal/model/system_user.go (*model.SystemUser)
CREATE TABLE "system_users" (
  "id" bigserial,
  "username" text NOT NULL,
  "password" text NOT NULL,
  "nickname" text NOT NULL,
  "remark" text,
  "dept_id" bigint,
  "post_ids" text,
  "email" text,
  "mobile" text,
  "sex" integer,
  "avatar" text,
  "status" integer NOT NULL,
  "login_ip" text,
  "login_date" timestamptz,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_user_post.go (*model.SystemUserPost)
CREATE TABLE "system_user_post" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "post_id" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/system_user_role.go (*model.SystemUserRole)
CREATE TABLE "system_user_role" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "role_id" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/after_sale.go (*trade.AfterSale)
CREATE TABLE "trade_after_sale" (
  "id" bigserial,
  "no" text,
  "status" bigint,
  "way" bigint,
  "type" bigint,
  "user_id" bigint,
  "apply_reason" text,
  "apply_description" text,
  "apply_pic_urls" text,
  "order_id" bigint,
  "order_no" text,
  "order_item_id" bigint,
  "spu_id" bigint,
  "spu_name" text,
  "sku_id" bigint,
  "properties" text,
  "pic_url" text,
  "count" bigint,
  "audit_time" timestamptz,
  "audit_user_id" bigint,
  "audit_reason" text,
  "refund_price" bigint,
  "pay_refund_id" bigint,
  "refund_time" timestamptz,
  "logistics_id" bigint,
  "logistics_no" text,
  "delivery_time" timestamptz,
  "receive_time" timestamptz,
  "receive_reason" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/after_sale_log.go (*trade.AfterSaleLog)
CREATE TABLE "trade_after_sale_log" (
  "id" bigserial,
  "user_id" bigint,
  "user_type" bigint,
  "after_sale_id" bigint,
  "before_status" bigint,
  "after_status" bigint,
  "operate_type" bigint,
  "content" text,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/brokerage/brokerage_record.go (*brokerage.BrokerageRecord)
CREATE TABLE "trade_brokerage_record" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "biz_id" varchar(64) NOT NULL,
  "biz_type" bigint NOT NULL,
  "title" varchar(64) NOT NULL,
  "description" varchar(255) NOT NULL,
  "price" bigint NOT NULL,
  "total_price" bigint NOT NULL,
  "status" bigint NOT NULL,
  "frozen_days" bigint DEFAULT 0,
  "unfreeze_time" timestamptz,
  "source_user_level" bigint DEFAULT 0,
  "source_user_id" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/brokerage/brokerage_user.go (*brokerage.BrokerageUser)
CREATE TABLE "trade_brokerage_user" (
  "id" bigserial,
  "bind_user_id" bigint DEFAULT 0,
  "bind_user_time" timestamptz,
  "brokerage_enabled" smallint DEFAULT (0) CHECK ("brokerage_enabled" IN (0,1)),
  "brokerage_time" timestamptz,
  "brokerage_price" bigint DEFAULT 0,
  "frozen_price" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/brokerage/brokerage_withdraw.go (*brokerage.BrokerageWithdraw)
CREATE TABLE "trade_brokerage_withdraw" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "price" bigint NOT NULL,
  "fee_price" bigint DEFAULT 0,
  "total_price" bigint NOT NULL,
  "type" bigint NOT NULL,
  "user_name" varchar(64) DEFAULT '',
  "user_account" varchar(64) DEFAULT '',
  "qr_code_url" varchar(255) DEFAULT '',
  "bank_name" varchar(100) DEFAULT '',
  "bank_address" varchar(200) DEFAULT '',
  "status" bigint NOT NULL,
  "audit_reason" varchar(255) DEFAULT '',
  "audit_time" timestamptz,
  "remark" varchar(255) DEFAULT '',
  "pay_transfer_id" bigint DEFAULT 0,
  "transfer_channel_code" varchar(16) DEFAULT '',
  "transfer_time" timestamptz,
  "transfer_error_msg" varchar(255) DEFAULT '',
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/cart.go (*trade.Cart)
CREATE TABLE "trade_cart" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "sku_id" bigint NOT NULL,
  "count" bigint NOT NULL DEFAULT 1,
  "selected" smallint NOT NULL CHECK ("selected" IN (0,1)),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/config.go (*trade.TradeConfig)
CREATE TABLE "trade_config" (
  "id" bigserial,
  "after_sale_refund_reasons" JSONB,
  "after_sale_return_reasons" JSONB,
  "delivery_express_free_enabled" smallint DEFAULT (0) CHECK ("delivery_express_free_enabled" IN (0,1)),
  "delivery_express_free_price" bigint DEFAULT 0,
  "delivery_pick_up_enabled" smallint DEFAULT (0) CHECK ("delivery_pick_up_enabled" IN (0,1)),
  "brokerage_withdraw_min_price" bigint DEFAULT 0,
  "brokerage_withdraw_fee_percent" bigint DEFAULT 0,
  "brokerage_enabled" smallint DEFAULT (0) CHECK ("brokerage_enabled" IN (0,1)),
  "brokerage_frozen_days" bigint DEFAULT 0,
  "brokerage_first_percent" bigint DEFAULT 0,
  "brokerage_second_percent" bigint DEFAULT 0,
  "brokerage_enabled_condition" bigint DEFAULT 1,
  "brokerage_bind_mode" bigint DEFAULT 1,
  "brokerage_poster_urls" JSONB,
  "brokerage_withdraw_types" varchar(255),
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/delivery.go (*trade.TradeDeliveryExpress)
CREATE TABLE "trade_delivery_express" (
  "id" bigserial,
  "code" varchar(64) NOT NULL,
  "name" varchar(64) NOT NULL,
  "logo" varchar(256) DEFAULT '',
  "sort" bigint NOT NULL DEFAULT 0,
  "status" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/delivery.go (*trade.TradeDeliveryPickUpStore)
CREATE TABLE "trade_delivery_pick_up_store" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "introduction" varchar(256) DEFAULT '',
  "phone" varchar(11) NOT NULL,
  "area_id" bigint NOT NULL,
  "detail_address" varchar(256) NOT NULL,
  "logo" varchar(256) NOT NULL,
  "opening_time" text,
  "closing_time" text,
  "latitude" decimal(10,6),
  "longitude" decimal(10,6),
  "verify_user_ids" varchar(500),
  "status" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/delivery.go (*trade.TradeDeliveryExpressTemplate)
CREATE TABLE "trade_delivery_express_template" (
  "id" bigserial,
  "name" varchar(64) NOT NULL,
  "charge_mode" bigint NOT NULL DEFAULT 0,
  "sort" bigint NOT NULL DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/delivery.go (*trade.TradeDeliveryExpressTemplateCharge)
CREATE TABLE "trade_delivery_express_template_charge" (
  "id" bigserial,
  "template_id" bigint NOT NULL,
  "area_ids" text NOT NULL,
  "charge_mode" bigint NOT NULL,
  "start_count" decimal NOT NULL,
  "start_price" bigint NOT NULL,
  "extra_count" decimal NOT NULL,
  "extra_price" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/delivery.go (*trade.TradeDeliveryExpressTemplateFree)
CREATE TABLE "trade_delivery_express_template_free" (
  "id" bigserial,
  "template_id" bigint NOT NULL,
  "area_ids" text NOT NULL,
  "free_price" bigint NOT NULL,
  "free_count" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/trade_order.go (*trade.TradeOrder)
CREATE TABLE "trade_order" (
  "id" bigserial,
  "no" varchar(32) NOT NULL,
  "type" bigint NOT NULL,
  "terminal" bigint NOT NULL,
  "user_id" bigint NOT NULL,
  "user_ip" varchar(50) NOT NULL,
  "user_remark" varchar(255),
  "status" bigint NOT NULL,
  "product_count" bigint NOT NULL,
  "finish_time" timestamptz,
  "cancel_time" timestamptz,
  "cancel_type" bigint,
  "remark" varchar(255),
  "comment_status" smallint NOT NULL DEFAULT (0) CHECK ("comment_status" IN (0,1)),
  "brokerage_user_id" bigint,
  "pay_order_id" bigint,
  "pay_status" smallint NOT NULL DEFAULT (0) CHECK ("pay_status" IN (0,1)),
  "pay_time" timestamptz,
  "pay_channel_code" varchar(16),
  "total_price" bigint NOT NULL,
  "discount_price" bigint NOT NULL,
  "delivery_price" bigint NOT NULL,
  "adjust_price" bigint NOT NULL,
  "pay_price" bigint NOT NULL,
  "delivery_type" bigint NOT NULL,
  "logistics_id" bigint,
  "logistics_no" varchar(64),
  "delivery_time" timestamptz,
  "receive_time" timestamptz,
  "receiver_name" varchar(30) NOT NULL,
  "receiver_mobile" varchar(20) NOT NULL,
  "receiver_area_id" bigint NOT NULL,
  "receiver_detail_address" varchar(255) NOT NULL,
  "pick_up_store_id" bigint,
  "pick_up_verify_code" varchar(64),
  "refund_status" bigint NOT NULL,
  "refund_price" bigint NOT NULL,
  "coupon_id" bigint NOT NULL DEFAULT 0,
  "coupon_price" bigint NOT NULL,
  "use_point" bigint NOT NULL DEFAULT 0,
  "point_price" bigint NOT NULL DEFAULT 0,
  "give_point" bigint NOT NULL DEFAULT 0,
  "refund_point" bigint NOT NULL DEFAULT 0,
  "vip_price" bigint NOT NULL DEFAULT 0,
  "give_coupon_template_counts" json,
  "give_coupon_ids" varchar(255),
  "seckill_activity_id" bigint,
  "bargain_activity_id" bigint,
  "bargain_record_id" bigint,
  "combination_activity_id" bigint,
  "combination_head_id" bigint,
  "combination_record_id" bigint,
  "point_activity_id" bigint,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/trade_order.go (*trade.TradeOrderItem)
CREATE TABLE "trade_order_item" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "order_id" bigint NOT NULL,
  "cart_id" bigint NOT NULL,
  "spu_id" bigint NOT NULL,
  "spu_name" varchar(255) NOT NULL,
  "sku_id" bigint NOT NULL,
  "properties" JSONB,
  "pic_url" varchar(255),
  "count" bigint NOT NULL,
  "comment_status" smallint NOT NULL DEFAULT (0) CHECK ("comment_status" IN (0,1)),
  "price" bigint NOT NULL,
  "discount_price" bigint NOT NULL,
  "delivery_price" bigint NOT NULL,
  "adjust_price" bigint NOT NULL,
  "pay_price" bigint NOT NULL,
  "coupon_price" bigint NOT NULL,
  "point_price" bigint NOT NULL,
  "use_point" bigint NOT NULL DEFAULT 0,
  "give_point" bigint NOT NULL DEFAULT 0,
  "vip_price" bigint NOT NULL DEFAULT 0,
  "after_sale_id" bigint,
  "after_sale_status" bigint NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/trade_order.go (*trade.TradeOrderLog)
CREATE TABLE "trade_order_log" (
  "id" bigserial,
  "user_id" bigint NOT NULL,
  "user_type" smallint NOT NULL,
  "order_id" bigint NOT NULL,
  "before_status" bigint,
  "after_status" bigint,
  "operate_type" bigint NOT NULL,
  "content" varchar(2000) NOT NULL,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);

-- internal/model/trade/trade_statistics.go (*trade.TradeStatistics)
CREATE TABLE "trade_statistics" (
  "id" bigserial,
  "time" timestamptz NOT NULL,
  "order_create_count" bigint DEFAULT 0,
  "order_pay_count" bigint DEFAULT 0,
  "order_pay_price" bigint DEFAULT 0,
  "after_sale_count" bigint DEFAULT 0,
  "after_sale_refund_price" bigint DEFAULT 0,
  "brokerage_settlement_price" bigint DEFAULT 0,
  "wallet_pay_price" bigint DEFAULT 0,
  "recharge_pay_count" bigint DEFAULT 0,
  "recharge_pay_price" bigint DEFAULT 0,
  "recharge_refund_count" bigint DEFAULT 0,
  "recharge_refund_price" bigint DEFAULT 0,
  "creator" varchar(64) DEFAULT '',
  "updater" varchar(64) DEFAULT '',
  "create_time" timestamptz,
  "update_time" timestamptz,
  "deleted" smallint DEFAULT (0) CHECK ("deleted" IN (0,1)),
  "tenant_id" bigint DEFAULT 0,
  PRIMARY KEY ("id")
);
COMMIT;
