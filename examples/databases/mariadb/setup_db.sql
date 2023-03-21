--- You should run this query using `mysql -u root < setup_db.sql`

DROP DATABASE IF EXISTS devbox_lamp;
CREATE DATABASE devbox_lamp;

USE devbox_lamp

CREATE USER 'devbox_user'@'localhost' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON devbox_lamp.* TO 'devbox_user'@'localhost' IDENTIFIED BY 'password';

DROP TABLE IF EXISTS colors;
CREATE TABLE colors (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	hex VARCHAR(7) NOT NULL,
	PRIMARY KEY (id));

INSERT INTO colors (name, hex) VALUES ('red', '#FF0000'), ('blue', '#0000FF'), ('green', '#00FF00');


