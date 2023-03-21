<?php

/**
 * @file
 * Router script for the built-in PHP web server.
 *
 * The built-in web server should only be used for development and testing as it
 * has a number of limitations that makes running Drupal on it highly insecure
 * and somewhat limited.
 *
 * Note that:
 * - The server is single-threaded, any requests made during the execution of
 *   the main request will hang until the main request has been completed.
 * - The web server does not enforce any of the settings in .htaccess in
 *   particular a remote user will be able to download files that normally would
 *   be protected from direct access such as .module files.
 *
 * The router script is needed to work around a bug in PHP, see
 * https://bugs.php.net/bug.php?id=61286.
 *
 * Usage:
 * php -S localhost:8888 .ht.router.php
 *
 * @see http://php.net/manual/en/features.commandline.webserver.php
 */

$url = parse_url($_SERVER['REQUEST_URI']);
if (file_exists(__DIR__ . $url['path'])) {
  // Serve the requested resource as-is.
  return FALSE;
}

// Work around the PHP bug.
$path = $url['path'];
$script = 'index.php';
if (strpos($path, '.php') !== FALSE) {
  // Work backwards through the path to check if a script exists. Otherwise
  // fallback to index.php.
  do {
    $path = dirname($path);
    if (preg_match('/\.php$/', $path) && is_file(__DIR__ . $path)) {
      // Discovered that the path contains an existing PHP file. Use that as the
      // script to include.
      $script = ltrim($path, '/');
      break;
    }
  } while ($path !== '/' && $path !== '.');
}

// Update $_SERVER variables to point to the correct index-file.
$index_file_absolute = $_SERVER['DOCUMENT_ROOT'] . DIRECTORY_SEPARATOR . $script;
$index_file_relative = DIRECTORY_SEPARATOR . $script;

// SCRIPT_FILENAME will point to the router script itself, it should point to
// the full path of index.php.
$_SERVER['SCRIPT_FILENAME'] = $index_file_absolute;

// SCRIPT_NAME and PHP_SELF will either point to index.php or contain the full
// virtual path being requested depending on the URL being requested. They
// should always point to index.php relative to document root.
$_SERVER['SCRIPT_NAME'] = $index_file_relative;
$_SERVER['PHP_SELF'] = $index_file_relative;

// Require the script and let core take over.
require $_SERVER['SCRIPT_FILENAME'];
