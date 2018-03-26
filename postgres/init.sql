CREATE TABLE account (
  username VARCHAR(255) PRIMARY KEY,
  password VARCHAR(255)
);

CREATE TABLE file (
  username VARCHAR(255),
  filename VARCHAR(255),
  content_type VARCHAR(255),
  content_length VARCHAR(255),
  PRIMARY KEY(username, filename)
);