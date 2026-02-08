ALTER TABLE users ADD COLUMN login VARCHAR(255);


ALTER TABLE users
  ADD CONSTRAINT users_login_uniq UNIQUE (login);
