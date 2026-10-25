-- Create "users" table
CREATE TABLE "public"."users" (
  "id" bigserial NOT NULL,
  "external_id" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id")
);
-- Create index "idx_users_external_id" to table: "users"
CREATE UNIQUE INDEX "idx_users_external_id" ON "public"."users" ("external_id");
-- Create "device_tokens" table
CREATE TABLE "public"."device_tokens" (
  "id" bigserial NOT NULL,
  "user_id" bigint NOT NULL,
  "token" text NOT NULL,
  "platform" character varying(16) NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_users_tokens" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE CASCADE ON DELETE CASCADE
);
-- Create index "idx_device_tokens_active_cursor" to table: "device_tokens"
CREATE INDEX "idx_device_tokens_active_cursor" ON "public"."device_tokens" ("id") WHERE (active = true);
-- Create index "idx_device_tokens_token" to table: "device_tokens"
CREATE UNIQUE INDEX "idx_device_tokens_token" ON "public"."device_tokens" ("token");
-- Create index "idx_device_tokens_user_id" to table: "device_tokens"
CREATE INDEX "idx_device_tokens_user_id" ON "public"."device_tokens" ("user_id");
-- Create "admins" table
CREATE TABLE "public"."admins" (
  "id" bigserial NOT NULL,
  "email" text NOT NULL,
  "password_hash" text NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id")
);
-- Create index "idx_admins_email_lower" to table: "admins"
CREATE UNIQUE INDEX "idx_admins_email_lower" ON "public"."admins" ((lower(email)));
-- Create "notifications" table
CREATE TABLE "public"."notifications" (
  "id" bigserial NOT NULL,
  "title" text NOT NULL,
  "body" text NOT NULL,
  "created_by" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_notifications_admin" FOREIGN KEY ("created_by") REFERENCES "public"."admins" ("id") ON UPDATE CASCADE ON DELETE RESTRICT,
  CONSTRAINT "ck_notifications_body_length" CHECK ((char_length(body) >= 1) AND (char_length(body) <= 2000)),
  CONSTRAINT "ck_notifications_title_length" CHECK ((char_length(title) >= 1) AND (char_length(title) <= 120))
);
-- Create index "idx_notifications_created_at" to table: "notifications"
CREATE INDEX "idx_notifications_created_at" ON "public"."notifications" ("created_at");
-- Create index "idx_notifications_created_by" to table: "notifications"
CREATE INDEX "idx_notifications_created_by" ON "public"."notifications" ("created_by");
-- Create "notification_batches" table
CREATE TABLE "public"."notification_batches" (
  "notification_id" bigint NOT NULL,
  "batch_no" bigint NOT NULL,
  "token_count" bigint NOT NULL DEFAULT 0,
  "success" bigint NOT NULL DEFAULT 0,
  "failure" bigint NOT NULL DEFAULT 0,
  "invalid" bigint NOT NULL DEFAULT 0,
  "status" character varying(16) NOT NULL DEFAULT 'sending',
  "error" text NULL,
  "started_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "completed_at" timestamptz NULL,
  PRIMARY KEY ("notification_id", "batch_no"),
  CONSTRAINT "fk_notification_batches_notification" FOREIGN KEY ("notification_id") REFERENCES "public"."notifications" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "ck_notification_batches_status" CHECK ((status)::text = ANY ((ARRAY['sending'::character varying, 'done'::character varying, 'failed'::character varying])::text[]))
);
-- Create index "idx_notification_batches_status" to table: "notification_batches"
CREATE INDEX "idx_notification_batches_status" ON "public"."notification_batches" ("status");
-- Create "notification_jobs" table
CREATE TABLE "public"."notification_jobs" (
  "notification_id" bigint NOT NULL,
  "status" character varying(16) NOT NULL DEFAULT 'pending',
  "high_watermark" bigint NOT NULL DEFAULT 0,
  "cursor" bigint NOT NULL DEFAULT 0,
  "total" bigint NOT NULL DEFAULT 0,
  "sent" bigint NOT NULL DEFAULT 0,
  "failed" bigint NOT NULL DEFAULT 0,
  "invalid" bigint NOT NULL DEFAULT 0,
  "attempts" bigint NOT NULL DEFAULT 0,
  "next_attempt_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "lease_owner" text NULL,
  "lease_until" timestamptz NULL,
  "last_error" text NULL,
  "started_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  PRIMARY KEY ("notification_id"),
  CONSTRAINT "fk_notification_jobs_notification" FOREIGN KEY ("notification_id") REFERENCES "public"."notifications" ("id") ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT "ck_notification_jobs_status" CHECK ((status)::text = ANY ((ARRAY['pending'::character varying, 'processing'::character varying, 'done'::character varying, 'failed'::character varying])::text[]))
);
-- Create index "idx_notification_jobs_claim" to table: "notification_jobs"
CREATE INDEX "idx_notification_jobs_claim" ON "public"."notification_jobs" ("status", "next_attempt_at", "lease_until");
