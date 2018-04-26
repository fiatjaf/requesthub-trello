CREATE TABLE input (
  address text PRIMARY KEY,
  owner text NOT NULL
);

CREATE TABLE output (
  id serial PRIMARY KEY,
  kind text NOT NULL,
  target text NOT NULL,
  filter text NOT NULL,
  owner text NOT NULL,
  data jsonb,

  UNIQUE (target, owner, filter)
);
CREATE INDEX ON output (target);

CREATE TABLE pipe (
  i text REFERENCES input(address),
  o int REFERENCES output(id)
);

