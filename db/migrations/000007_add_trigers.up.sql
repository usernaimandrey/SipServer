CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_call_journal_touch ON call_journal;
CREATE TRIGGER trg_call_journal_touch
BEFORE UPDATE ON call_journals
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS trg_call_sessions_touch ON call_sessions;
CREATE TRIGGER trg_call_sessions_touch
BEFORE UPDATE ON call_sessions
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
