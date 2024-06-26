# Create this Postgresql database and run the script be free to tune it to how you want it to be!

```sql
-- Database: web-crawler

-- DROP DATABASE IF EXISTS "web-crawler";

CREATE DATABASE "web-crawler"
    WITH
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'English_United States.1252'
    LC_CTYPE = 'English_United States.1252'
    LOCALE_PROVIDER = 'libc'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1
    IS_TEMPLATE = False;

-- Table: public.crawled_pages

-- DROP TABLE IF EXISTS public.crawled_pages;

CREATE TABLE IF NOT EXISTS public.crawled_pages
(
    url text COLLATE pg_catalog."default" NOT NULL,
    content text COLLATE pg_catalog."default" NOT NULL,
    title text COLLATE pg_catalog."default" NOT NULL,
    parent_url text COLLATE pg_catalog."default",
    "timestamp" timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    content_hash text COLLATE pg_catalog."default",
    host text COLLATE pg_catalog."default",
    icon_link text COLLATE pg_catalog."default",
    site_name text COLLATE pg_catalog."default",
    description text COLLATE pg_catalog."default",
    CONSTRAINT crawled_pages_pkey PRIMARY KEY (url),
    CONSTRAINT crawled_pages_parent_link_fkey FOREIGN KEY (parent_url)
        REFERENCES public.crawled_pages (url) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.crawled_pages
    OWNER to postgres;
-- Index: idx_content_hash

-- DROP INDEX IF EXISTS public.idx_content_hash;

CREATE INDEX IF NOT EXISTS idx_content_hash
    ON public.crawled_pages USING btree
    (content_hash COLLATE pg_catalog."default" text_pattern_ops ASC NULLS LAST)
    WITH (deduplicate_items=False)
    TABLESPACE pg_default;
-- Index: idx_url

-- DROP INDEX IF EXISTS public.idx_url;

CREATE INDEX IF NOT EXISTS idx_url
    ON public.crawled_pages USING btree
    (url COLLATE pg_catalog."C" ASC NULLS LAST)
    TABLESPACE pg_default;

-- Table: public.page_words

-- DROP TABLE IF EXISTS public.page_words;

CREATE TABLE IF NOT EXISTS public.page_words
(
    url text COLLATE pg_catalog."default" NOT NULL,
    word text COLLATE pg_catalog."default" NOT NULL,
    frequency double precision NOT NULL,
    CONSTRAINT word_frequencies_pkey PRIMARY KEY (url, word),
    CONSTRAINT word_frequencies_page_url_fkey FOREIGN KEY (url)
        REFERENCES public.crawled_pages (url) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.page_words
    OWNER to postgres;
-- Index: idx_word

-- DROP INDEX IF EXISTS public.idx_word;

CREATE INDEX IF NOT EXISTS idx_word
    ON public.page_words USING btree
    (word COLLATE pg_catalog."default" ASC NULLS LAST)
    TABLESPACE pg_default;

```
