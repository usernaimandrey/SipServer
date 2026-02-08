--
-- PostgreSQL database dump
--

-- Dumped from database version 13.21
-- Dumped by pg_dump version 13.21

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

ALTER TABLE IF EXISTS ONLY public.user_configs DROP CONSTRAINT IF EXISTS user_configs_user_fk;
ALTER TABLE IF EXISTS ONLY public.call_sessions DROP CONSTRAINT IF EXISTS call_sessions_journal_id_fkey;
DROP TRIGGER IF EXISTS trg_user_configs_updated_at ON public.user_configs;
DROP INDEX IF EXISTS public.user_configs_user_id_idx;
DROP INDEX IF EXISTS public.call_sessions_state_idx;
DROP INDEX IF EXISTS public.call_sessions_dialog_uniq;
DROP INDEX IF EXISTS public.call_sessions_call_id_idx;
DROP INDEX IF EXISTS public.call_journals_result_idx;
DROP INDEX IF EXISTS public.call_journals_parties_idx;
DROP INDEX IF EXISTS public.call_journals_missed_idx;
DROP INDEX IF EXISTS public.call_journals_invite_at_idx;
DROP INDEX IF EXISTS public.call_journals_callee_idx;
DROP INDEX IF EXISTS public.call_journals_call_id_idx;
ALTER TABLE IF EXISTS ONLY public.users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE IF EXISTS ONLY public.users DROP CONSTRAINT IF EXISTS users_login_uniq;
ALTER TABLE IF EXISTS ONLY public.user_configs DROP CONSTRAINT IF EXISTS user_configs_user_id_uniq;
ALTER TABLE IF EXISTS ONLY public.user_configs DROP CONSTRAINT IF EXISTS user_configs_pkey;
ALTER TABLE IF EXISTS ONLY public.schema_migrations DROP CONSTRAINT IF EXISTS schema_migrations_pkey;
ALTER TABLE IF EXISTS ONLY public.call_sessions DROP CONSTRAINT IF EXISTS call_sessions_pkey;
ALTER TABLE IF EXISTS ONLY public.call_journals DROP CONSTRAINT IF EXISTS call_journals_pkey;
ALTER TABLE IF EXISTS public.users ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.user_configs ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.call_sessions ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.call_journals ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE IF EXISTS public.users_id_seq;
DROP TABLE IF EXISTS public.users;
DROP SEQUENCE IF EXISTS public.user_configs_id_seq;
DROP TABLE IF EXISTS public.user_configs;
DROP TABLE IF EXISTS public.schema_migrations;
DROP SEQUENCE IF EXISTS public.call_sessions_id_seq;
DROP TABLE IF EXISTS public.call_sessions;
DROP SEQUENCE IF EXISTS public.call_journals_id_seq;
DROP TABLE IF EXISTS public.call_journals;
DROP FUNCTION IF EXISTS public.set_updated_at();
DROP TYPE IF EXISTS public.session_state;
DROP TYPE IF EXISTS public.ended_by;
DROP TYPE IF EXISTS public.call_schema;
DROP TYPE IF EXISTS public.call_result;
--
-- Name: call_result; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.call_result AS ENUM (
    'answered',
    'rejected',
    'cancelled',
    'no_answer',
    'failed'
);


--
-- Name: call_schema; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.call_schema AS ENUM (
    'redirect',
    'proxy'
);


--
-- Name: ended_by; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.ended_by AS ENUM (
    'caller',
    'callee',
    'system'
);


--
-- Name: session_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.session_state AS ENUM (
    'early',
    'active',
    'terminated'
);


--
-- Name: set_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: call_journals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.call_journals (
    id bigint NOT NULL,
    call_id text NOT NULL,
    init_branch text,
    from_tag text,
    to_tag text,
    caller_user text NOT NULL,
    callee_user text NOT NULL,
    caller_uri text,
    callee_uri text,
    invite_at timestamp with time zone DEFAULT now() NOT NULL,
    first_18x_at timestamp with time zone,
    answer_at timestamp with time zone,
    end_at timestamp with time zone,
    result public.call_result,
    final_code integer,
    final_reason text,
    ring_ms integer,
    talk_ms integer,
    ended_by public.ended_by,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: call_journals_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.call_journals_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: call_journals_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.call_journals_id_seq OWNED BY public.call_journals.id;


--
-- Name: call_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.call_sessions (
    id bigint NOT NULL,
    journal_id bigint NOT NULL,
    call_id text NOT NULL,
    from_tag text NOT NULL,
    to_tag text NOT NULL,
    state public.session_state DEFAULT 'active'::public.session_state NOT NULL,
    remote_target text,
    route_set jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    established_at timestamp with time zone,
    terminated_at timestamp with time zone,
    ended_by public.ended_by,
    term_code integer,
    term_reason text,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: call_sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.call_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: call_sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.call_sessions_id_seq OWNED BY public.call_sessions.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: user_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_configs (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    call_schema public.call_schema DEFAULT 'proxy'::public.call_schema NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: user_configs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_configs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_configs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_configs_id_seq OWNED BY public.user_configs.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    password_hash character varying(50) NOT NULL,
    role character varying(50) DEFAULT 'user'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    login character varying(255)
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: call_journals id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_journals ALTER COLUMN id SET DEFAULT nextval('public.call_journals_id_seq'::regclass);


--
-- Name: call_sessions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_sessions ALTER COLUMN id SET DEFAULT nextval('public.call_sessions_id_seq'::regclass);


--
-- Name: user_configs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_configs ALTER COLUMN id SET DEFAULT nextval('public.user_configs_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: call_journals call_journals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_journals
    ADD CONSTRAINT call_journals_pkey PRIMARY KEY (id);


--
-- Name: call_sessions call_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_sessions
    ADD CONSTRAINT call_sessions_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: user_configs user_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_configs
    ADD CONSTRAINT user_configs_pkey PRIMARY KEY (id);


--
-- Name: user_configs user_configs_user_id_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_configs
    ADD CONSTRAINT user_configs_user_id_uniq UNIQUE (user_id);


--
-- Name: users users_login_uniq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_login_uniq UNIQUE (login);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: call_journals_call_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_journals_call_id_idx ON public.call_journals USING btree (call_id);


--
-- Name: call_journals_callee_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_journals_callee_idx ON public.call_journals USING btree (callee_user, invite_at DESC);


--
-- Name: call_journals_invite_at_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_journals_invite_at_idx ON public.call_journals USING btree (invite_at DESC);


--
-- Name: call_journals_missed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_journals_missed_idx ON public.call_journals USING btree (invite_at DESC) WHERE (result = ANY (ARRAY['rejected'::public.call_result, 'cancelled'::public.call_result, 'no_answer'::public.call_result, 'failed'::public.call_result]));


--
-- Name: call_journals_parties_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_journals_parties_idx ON public.call_journals USING btree (caller_user, callee_user, invite_at DESC);


--
-- Name: call_journals_result_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_journals_result_idx ON public.call_journals USING btree (result, invite_at DESC);


--
-- Name: call_sessions_call_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_sessions_call_id_idx ON public.call_sessions USING btree (call_id, created_at DESC);


--
-- Name: call_sessions_dialog_uniq; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX call_sessions_dialog_uniq ON public.call_sessions USING btree (call_id, from_tag, to_tag);


--
-- Name: call_sessions_state_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX call_sessions_state_idx ON public.call_sessions USING btree (state, created_at DESC);


--
-- Name: user_configs_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX user_configs_user_id_idx ON public.user_configs USING btree (user_id);


--
-- Name: user_configs trg_user_configs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_user_configs_updated_at BEFORE UPDATE ON public.user_configs FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();


--
-- Name: call_sessions call_sessions_journal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.call_sessions
    ADD CONSTRAINT call_sessions_journal_id_fkey FOREIGN KEY (journal_id) REFERENCES public.call_journals(id) ON DELETE CASCADE;


--
-- Name: user_configs user_configs_user_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_configs
    ADD CONSTRAINT user_configs_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

