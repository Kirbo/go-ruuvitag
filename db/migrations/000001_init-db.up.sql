BEGIN;

CREATE TABLE public.devices (
	id bigserial NOT NULL,
	"tagId" varchar(255) NOT NULL,
	"name" varchar(255) NULL,
	"order" int4 NULL DEFAULT 1000,
	CONSTRAINT devices_pkey PRIMARY KEY (id),
	CONSTRAINT "devices_tagId_key" UNIQUE ("tagId")
);
CREATE INDEX devices_name ON devices USING btree (name);
CREATE INDEX devices_order ON devices USING btree ("order");
CREATE UNIQUE INDEX devices_tag_id ON devices USING btree ("tagId");

CREATE TABLE public.ruuvitag (
	id bigserial NOT NULL,
	"time" timestamptz NOT NULL,
	metric varchar(255) NOT NULL,
	"tagId" varchar(255) NOT NULL,
	value numeric NOT NULL,
	CONSTRAINT ruuvitag_pkey PRIMARY KEY (id)
);
CREATE INDEX ruuvitag_metric ON ruuvitag USING btree (metric);
CREATE INDEX ruuvitag_tag_id ON ruuvitag USING btree ("tagId");
CREATE INDEX ruuvitag_time ON ruuvitag USING btree ("time");
CREATE INDEX ruuvitag_time_idx ON ruuvitag USING btree ("time", metric, "tagId");

CREATE TABLE public.ruuvitag_1 (
	id bigserial NOT NULL,
	"time" timestamptz NOT NULL,
	metric varchar(255) NOT NULL,
	"tagId" varchar(255) NOT NULL,
	value numeric NOT NULL
);

COMMIT;
