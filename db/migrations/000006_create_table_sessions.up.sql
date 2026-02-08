CREATE TABLE IF NOT EXISTS call_sessions (
  id                BIGSERIAL PRIMARY KEY,

  journal_id         BIGINT NOT NULL REFERENCES call_journals(id) ON DELETE CASCADE,

  -- Корреляция
  call_id            TEXT NOT NULL,
  from_tag           TEXT NOT NULL,
  to_tag             TEXT NOT NULL,

  -- Состояние сессии
  state              session_state NOT NULL DEFAULT 'active',

  -- Диалоговые данные 
  remote_target      TEXT,             
  route_set          JSONB,            

  -- Тайминги
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),  
  established_at     TIMESTAMPTZ,                         
  terminated_at      TIMESTAMPTZ,

  
  ended_by           ended_by,
  term_code          INTEGER,          
  term_reason        TEXT,

  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE INDEX IF NOT EXISTS call_sessions_state_idx
  ON call_sessions(state, created_at DESC);


CREATE UNIQUE INDEX IF NOT EXISTS call_sessions_dialog_uniq
  ON call_sessions(call_id, from_tag, to_tag);


CREATE INDEX IF NOT EXISTS call_sessions_call_id_idx
  ON call_sessions(call_id, created_at DESC);
