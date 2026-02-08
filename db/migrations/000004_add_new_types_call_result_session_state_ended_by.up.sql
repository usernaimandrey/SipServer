DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'call_result') THEN
    CREATE TYPE call_result AS ENUM (
      'answered',     -- 2xx на INVITE
      'rejected',     -- 486/603/...
      'cancelled',    -- CANCEL до 2xx (обычно caller)
      'no_answer',    -- таймаут/408/внутренний таймер
      'failed'        -- прочие 4xx/5xx/6xx
    );
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_state') THEN
    CREATE TYPE session_state AS ENUM (
      'early',        -- INVITE идёт, но 2xx ещё нет (опционально)
      'active',       -- диалог установлен (после 2xx)
      'terminated'    -- завершено (BYE/ошибка)
    );
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ended_by') THEN
    CREATE TYPE ended_by AS ENUM ('caller','callee','system');
  END IF;
END $$;
