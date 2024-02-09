--- You should run this query using `mysql -u root < setup_db.sql`

DROP DATABASE IF EXISTS devbox_drupal;
CREATE DATABASE devbox_drupal;

USE devbox_drupal

CREATE USER IF NOT EXISTS 'devbox_user'@'localhost' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON devbox_drupal.* TO 'devbox_user'@'localhost' IDENTIFIED BY 'password';

-- Connect in drupal using:
-- Database: devbox_drupal
-- User: devbox_user
-- Password: password
-- Host: 127.0.0.1
-- Port: 3306
