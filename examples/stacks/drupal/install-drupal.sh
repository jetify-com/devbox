#!/bin/sh

set -e

echo "Make sure to run this script only after 'devbox shell'"

mysql -u root < setup_db.sql
composer install

echo "Your Drupal demo website is ready,"
echo "Open localhost:8081 in your browser."
