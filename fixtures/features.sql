CREATE TABLE IF NOT EXISTS features (
  full_name  VARCHAR (255) NOT NULL PRIMARY KEY,
  service  VARCHAR (255) NOT NULL,
  name  VARCHAR (255) NOT NULL,
  enabled BOOLEAN,
  strategies TEXT,
  variants TEXT,
  created_at TEXT
);

INSERT OR REPLACE
  INTO features (full_name, service, name, enabled, strategies, variants, created_at)
  VALUES (
    'dummy.feature1',
    'dummy',
    'feature',
    1,
    '[{"name":"userWithId","parameters":{"userIds":"1,2,3,4"}},{"name":"default","parameters":{}}]',
    null,
    '2020-02-10 10:10:10'
  );

INSERT OR REPLACE
  INTO features (full_name, service, name, enabled, strategies, variants, created_at)
  VALUES (
    'dummy.feature2',
    'dummy',
    'feature',
    0,
    '[{"name":"userWithId","parameters":{"userIds":"1,2,3,4"}},{"name":"default","parameters":{}}]',
    null,
    '2020-02-10 10:10:10'
  );
