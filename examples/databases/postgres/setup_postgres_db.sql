--- You should run this query using psql < setup_db.sql`

DROP DATABASE IF EXISTS devbox_lamp;
CREATE DATABASE devbox_lamp;

CREATE USER devbox_user WITH PASSWORD 'password';

DROP TABLE IF EXISTS colors;
CREATE TABLE colors (
	id SERIAL NOT NULL PRIMARY KEY,
	name VARCHAR(100) NOT NULL,
	hex VARCHAR(7) NOT NULL);

INSERT INTO colors (name, hex) VALUES ('red', '#FF0000'), ('blue', '#0000FF'), ('green', '#00FF00');

GRANT ALL PRIVILEGES ON colors TO devbox_user;
