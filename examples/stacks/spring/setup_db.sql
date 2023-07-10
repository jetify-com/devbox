-- You should run this query using `mysql -u root < setup_db.sql`

DROP DATABASE IF EXISTS db_example;
CREATE DATABASE db_example;

USE db_example;

CREATE USER 'springuser'@'%' IDENTIFIED BY 'password';
GRANT ALL ON db_example.* TO 'springuser'@'%';