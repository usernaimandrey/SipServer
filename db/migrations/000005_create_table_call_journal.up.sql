CREATE TABLE IF NOT EXISTS call_journals (
  id              BIGSERIAL PRIMARY KEY,

  -- Корреляция SIP
  call_id         TEXT NOT NULL,
  init_branch     TEXT,               -- branch входящего INVITE (Via branch от UAC)
  from_tag        TEXT,
  to_tag          TEXT,

  -- Кто кому
  caller_user     TEXT NOT NULL,      -- "1001"
  callee_user     TEXT NOT NULL,      -- "1002"
  caller_uri      TEXT,
  callee_uri      TEXT,

  -- Тайминги
  invite_at       TIMESTAMPTZ NOT NULL DEFAULT now(),  -- начало дозвона
  first_18x_at    TIMESTAMPTZ,                         -- первый 180/183 (если нужно)
  answer_at       TIMESTAMPTZ,                         -- 2xx
  end_at          TIMESTAMPTZ,                         -- итоговое завершение попытки

  -- Итог попытки
  result          call_result,         -- заполняется когда попытка завершена
  final_code      INTEGER,             -- 200/486/603/487/408/5xx...
  final_reason    TEXT,

  -- Длительности (в мс)
  ring_ms         INTEGER,             -- answer_at - invite_at
  talk_ms         INTEGER,             -- end_at - answer_at (если ответили и потом BYE)

  ended_by        ended_by,            -- кто завершил

  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS call_journals_call_id_idx
  ON call_journals(call_id);

CREATE INDEX IF NOT EXISTS call_journals_invite_at_idx
  ON call_journals(invite_at DESC);

CREATE INDEX IF NOT EXISTS call_journals_parties_idx
  ON call_journals(caller_user, callee_user, invite_at DESC);

CREATE INDEX IF NOT EXISTS call_journals_callee_idx
  ON call_journals(callee_user, invite_at DESC);

CREATE INDEX IF NOT EXISTS call_journals_result_idx
  ON call_journals(result, invite_at DESC);

-- Частичный индекс "пропущенные" (быстро для UI)
CREATE INDEX IF NOT EXISTS call_journals_missed_idx
  ON call_journals(invite_at DESC)
  WHERE result IN ('rejected','cancelled','no_answer','failed');
