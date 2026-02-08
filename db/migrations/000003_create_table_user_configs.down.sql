DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'call_schema') THEN
    CREATE TYPE call_schema AS ENUM (
      'redirect',
      'proxy'
    );
  END IF;
END $$;


CREATE TABLE IF NOT EXISTS user_configs (
  id          BIGSERIAL PRIMARY KEY,

  user_id     BIGINT NOT NULL,
  call_schema call_schema NOT NULL DEFAULT 'proxy',

  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT user_configs_user_id_uniq UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS user_configs_user_id_idx
  ON user_configs(user_id);


ALTER TABLE user_configs
  ADD CONSTRAINT user_configs_user_fk
  FOREIGN KEY (user_id)
  REFERENCES users(id)
  ON DELETE CASCADE;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_user_configs_updated_at
BEFORE UPDATE ON user_configs
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
