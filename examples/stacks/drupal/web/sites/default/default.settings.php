<?php

// phpcs:ignoreFile

/**
 * @file
 * Drupal site-specific configuration file.
 *
 * IMPORTANT NOTE:
 * This file may have been set to read-only by the Drupal installation program.
 * If you make changes to this file, be sure to protect it again after making
 * your modifications. Failure to remove write permissions to this file is a
 * security risk.
 *
 * In order to use the selection rules below the multisite aliasing file named
 * sites/sites.php must be present. Its optional settings will be loaded, and
 * the aliases in the array $sites will override the default directory rules
 * below. See sites/example.sites.php for more information about aliases.
 *
 * The configuration directory will be discovered by stripping the website's
 * hostname from left to right and pathname from right to left. The first
 * configuration file found will be used and any others will be ignored. If no
 * other configuration file is found then the default configuration file at
 * 'sites/default' will be used.
 *
 * For example, for a fictitious site installed at
 * https://www.drupal.org:8080/my-site/test/, the 'settings.php' file is searched
 * for in the following directories:
 *
 * - sites/8080.www.drupal.org.my-site.test
 * - sites/www.drupal.org.my-site.test
 * - sites/drupal.org.my-site.test
 * - sites/org.my-site.test
 *
 * - sites/8080.www.drupal.org.my-site
 * - sites/www.drupal.org.my-site
 * - sites/drupal.org.my-site
 * - sites/org.my-site
 *
 * - sites/8080.www.drupal.org
 * - sites/www.drupal.org
 * - sites/drupal.org
 * - sites/org
 *
 * - sites/default
 *
 * Note that if you are installing on a non-standard port number, prefix the
 * hostname with that number. For example,
 * https://www.drupal.org:8080/my-site/test/ could be loaded from
 * sites/8080.www.drupal.org.my-site.test/.
 *
 * @see example.sites.php
 * @see \Drupal\Core\DrupalKernel::getSitePath()
 *
 * In addition to customizing application settings through variables in
 * settings.php, you can create a services.yml file in the same directory to
 * register custom, site-specific service definitions and/or swap out default
 * implementations with custom ones.
 */

/**
 * Database settings:
 *
 * The $databases array specifies the database connection or
 * connections that Drupal may use.  Drupal is able to connect
 * to multiple databases, including multiple types of databases,
 * during the same request.
 *
 * One example of the simplest connection array is shown below. To use the
 * sample settings, copy and uncomment the code below between the @code and
 * @endcode lines and paste it after the $databases declaration. You will need
 * to replace the database username and password and possibly the host and port
 * with the appropriate credentials for your database system.
 *
 * The next section describes how to customize the $databases array for more
 * specific needs.
 *
 * @code
 * $databases['default']['default'] = [
 *   'database' => 'databasename',
 *   'username' => 'sql_username',
 *   'password' => 'sql_password',
 *   'host' => 'localhost',
 *   'port' => '3306',
 *   'driver' => 'mysql',
 *   'prefix' => '',
 *   'collation' => 'utf8mb4_general_ci',
 * ];
 * @endcode
 */
$databases = [];

/**
 * Customizing database settings.
 *
 * Many of the values of the $databases array can be customized for your
 * particular database system. Refer to the sample in the section above as a
 * starting point.
 *
 * The "driver" property indicates what Drupal database driver the
 * connection should use.  This is usually the same as the name of the
 * database type, such as mysql or sqlite, but not always.  The other
 * properties will vary depending on the driver.  For SQLite, you must
 * specify a database file name in a directory that is writable by the
 * webserver.  For most other drivers, you must specify a
 * username, password, host, and database name.
 *
 * Drupal core implements drivers for mysql, pgsql, and sqlite. Other drivers
 * can be provided by contributed or custom modules. To use a contributed or
 * custom driver, the "namespace" property must be set to the namespace of the
 * driver. The code in this namespace must be autoloadable prior to connecting
 * to the database, and therefore, prior to when module root namespaces are
 * added to the autoloader. To add the driver's namespace to the autoloader,
 * set the "autoload" property to the PSR-4 base directory of the driver's
 * namespace. This is optional for projects managed with Composer if the
 * driver's namespace is in Composer's autoloader.
 *
 * For each database, you may optionally specify multiple "target" databases.
 * A target database allows Drupal to try to send certain queries to a
 * different database if it can but fall back to the default connection if not.
 * That is useful for primary/replica replication, as Drupal may try to connect
 * to a replica server when appropriate and if one is not available will simply
 * fall back to the single primary server (The terms primary/replica are
 * traditionally referred to as master/slave in database server documentation).
 *
 * The general format for the $databases array is as follows:
 * @code
 * $databases['default']['default'] = $info_array;
 * $databases['default']['replica'][] = $info_array;
 * $databases['default']['replica'][] = $info_array;
 * $databases['extra']['default'] = $info_array;
 * @endcode
 *
 * In the above example, $info_array is an array of settings described above.
 * The first line sets a "default" database that has one primary database
 * (the second level default).  The second and third lines create an array
 * of potential replica databases.  Drupal will select one at random for a given
 * request as needed.  The fourth line creates a new database with a name of
 * "extra".
 *
 * For MySQL, MariaDB or equivalent databases the 'isolation_level' option can
 * be set. The recommended transaction isolation level for Drupal sites is
 * 'READ COMMITTED'. The 'REPEATABLE READ' option is supported but can result
 * in deadlocks, the other two options are 'READ UNCOMMITTED' and 'SERIALIZABLE'.
 * They are available but not supported; use them at your own risk. For more
 * info:
 * https://dev.mysql.com/doc/refman/5.7/en/innodb-transaction-isolation-levels.html
 *
 * On your settings.php, change the isolation level:
 * @code
 * $databases['default']['default']['init_commands'] = [
 *   'isolation_level' => 'SET SESSION TRANSACTION ISOLATION LEVEL READ COMMITTED',
 * ];
 * @endcode
 *
 * You can optionally set a prefix for all database table names by using the
 * 'prefix' setting. If a prefix is specified, the table name will be prepended
 * with its value. Be sure to use valid database characters only, usually
 * alphanumeric and underscore. If no prefix is desired, do not set the 'prefix'
 * key or set its value to an empty string ''.
 *
 * For example, to have all database table prefixed with 'main_', set:
 * @code
 *   'prefix' => 'main_',
 * @endcode
 *
 * Advanced users can add or override initial commands to execute when
 * connecting to the database server, as well as PDO connection settings. For
 * example, to enable MySQL SELECT queries to exceed the max_join_size system
 * variable, and to reduce the database connection timeout to 5 seconds:
 * @code
 * $databases['default']['default'] = [
 *   'init_commands' => [
 *     'big_selects' => 'SET SQL_BIG_SELECTS=1',
 *   ],
 *   'pdo' => [
 *     PDO::ATTR_TIMEOUT => 5,
 *   ],
 * ];
 * @endcode
 *
 * WARNING: The above defaults are designed for database portability. Changing
 * them may cause unexpected behavior, including potential data loss. See
 * https://www.drupal.org/developing/api/database/configuration for more
 * information on these defaults and the potential issues.
 *
 * More details can be found in the constructor methods for each driver:
 * - \Drupal\mysql\Driver\Database\mysql\Connection::__construct()
 * - \Drupal\pgsql\Driver\Database\pgsql\Connection::__construct()
 * - \Drupal\sqlite\Driver\Database\sqlite\Connection::__construct()
 *
 * Sample Database configuration format for PostgreSQL (pgsql):
 * @code
 *   $databases['default']['default'] = [
 *     'driver' => 'pgsql',
 *     'database' => 'databasename',
 *     'username' => 'sql_username',
 *     'password' => 'sql_password',
 *     'host' => 'localhost',
 *     'prefix' => '',
 *   ];
 * @endcode
 *
 * Sample Database configuration format for SQLite (sqlite):
 * @code
 *   $databases['default']['default'] = [
 *     'driver' => 'sqlite',
 *     'database' => '/path/to/database_filename',
 *   ];
 * @endcode
 *
 * Sample Database configuration format for a driver in a contributed module:
 * @code
 *   $databases['default']['default'] = [
 *     'driver' => 'my_driver',
 *     'namespace' => 'Drupal\my_module\Driver\Database\my_driver',
 *     'autoload' => 'modules/my_module/src/Driver/Database/my_driver/',
 *     'database' => 'databasename',
 *     'username' => 'sql_username',
 *     'password' => 'sql_password',
 *     'host' => 'localhost',
 *     'prefix' => '',
 *   ];
 * @endcode
 *
 * Sample Database configuration format for a driver that is extending another
 * database driver.
 * @code
 *   $databases['default']['default'] = [
 *     'driver' => 'my_driver',
 *     'namespace' => 'Drupal\my_module\Driver\Database\my_driver',
 *     'autoload' => 'modules/my_module/src/Driver/Database/my_driver/',
 *     'database' => 'databasename',
 *     'username' => 'sql_username',
 *     'password' => 'sql_password',
 *     'host' => 'localhost',
 *     'prefix' => '',
 *     'dependencies' => [
 *       'parent_module' => [
 *         'namespace' => 'Drupal\parent_module',
 *         'autoload' => 'core/modules/parent_module/src/',
 *       ],
 *     ],
 *   ];
 * @endcode
 */

/**
 * Location of the site configuration files.
 *
 * The $settings['config_sync_directory'] specifies the location of file system
 * directory used for syncing configuration data. On install, the directory is
 * created. This is used for configuration imports.
 *
 * The default location for this directory is inside a randomly-named
 * directory in the public files path. The setting below allows you to set
 * its location.
 */
# $settings['config_sync_directory'] = '/directory/outside/webroot';

/**
 * Settings:
 *
 * $settings contains environment-specific configuration, such as the files
 * directory and reverse proxy address, and temporary configuration, such as
 * security overrides.
 *
 * @see \Drupal\Core\Site\Settings::get()
 */

/**
 * Salt for one-time login links, cancel links, form tokens, etc.
 *
 * This variable will be set to a random value by the installer. All one-time
 * login links will be invalidated if the value is changed. Note that if your
 * site is deployed on a cluster of web servers, you must ensure that this
 * variable has the same value on each server.
 *
 * For enhanced security, you may set this variable to the contents of a file
 * outside your document root, and vary the value across environments (like
 * production and development); you should also ensure that this file is not
 * stored with backups of your database.
 *
 * Example:
 * @code
 *   $settings['hash_salt'] = file_get_contents('/home/example/salt.txt');
 * @endcode
 */
$settings['hash_salt'] = '';

/**
 * Deployment identifier.
 *
 * Drupal's dependency injection container will be automatically invalidated and
 * rebuilt when the Drupal core version changes. When updating contributed or
 * custom code that changes the container, changing this identifier will also
 * allow the container to be invalidated as soon as code is deployed.
 */
# $settings['deployment_identifier'] = \Drupal::VERSION;

/**
 * Access control for update.php script.
 *
 * If you are updating your Drupal installation using the update.php script but
 * are not logged in using either an account with the "Administer software
 * updates" permission or the site maintenance account (the account that was
 * created during installation), you will need to modify the access check
 * statement below. Change the FALSE to a TRUE to disable the access check.
 * After finishing the upgrade, be sure to open this file again and change the
 * TRUE back to a FALSE!
 */
$settings['update_free_access'] = FALSE;

/**
 * Fallback to HTTP for Update Manager and for fetching security advisories.
 *
 * If your site fails to connect to updates.drupal.org over HTTPS (either when
 * fetching data on available updates, or when fetching the feed of critical
 * security announcements), you may uncomment this setting and set it to TRUE to
 * allow an insecure fallback to HTTP. Note that doing so will open your site up
 * to a potential man-in-the-middle attack. You should instead attempt to
 * resolve the issues before enabling this option.
 * @see https://www.drupal.org/docs/system-requirements/php-requirements#openssl
 * @see https://en.wikipedia.org/wiki/Man-in-the-middle_attack
 * @see \Drupal\update\UpdateFetcher
 * @see \Drupal\system\SecurityAdvisories\SecurityAdvisoriesFetcher
 */
# $settings['update_fetch_with_http_fallback'] = TRUE;

/**
 * External access proxy settings:
 *
 * If your site must access the Internet via a web proxy then you can enter the
 * proxy settings here. Set the full URL of the proxy, including the port, in
 * variables:
 * - $settings['http_client_config']['proxy']['http']: The proxy URL for HTTP
 *   requests.
 * - $settings['http_client_config']['proxy']['https']: The proxy URL for HTTPS
 *   requests.
 * You can pass in the user name and password for basic authentication in the
 * URLs in these settings.
 *
 * You can also define an array of host names that can be accessed directly,
 * bypassing the proxy, in $settings['http_client_config']['proxy']['no'].
 */
# $settings['http_client_config']['proxy']['http'] = 'http://proxy_user:proxy_pass@example.com:8080';
# $settings['http_client_config']['proxy']['https'] = 'http://proxy_user:proxy_pass@example.com:8080';
# $settings['http_client_config']['proxy']['no'] = ['127.0.0.1', 'localhost'];

/**
 * Reverse Proxy Configuration:
 *
 * Reverse proxy servers are often used to enhance the performance
 * of heavily visited sites and may also provide other site caching,
 * security, or encryption benefits. In an environment where Drupal
 * is behind a reverse proxy, the real IP address of the client should
 * be determined such that the correct client IP address is available
 * to Drupal's logging, statistics, and access management systems. In
 * the most simple scenario, the proxy server will add an
 * X-Forwarded-For header to the request that contains the client IP
 * address. However, HTTP headers are vulnerable to spoofing, where a
 * malicious client could bypass restrictions by setting the
 * X-Forwarded-For header directly. Therefore, Drupal's proxy
 * configuration requires the IP addresses of all remote proxies to be
 * specified in $settings['reverse_proxy_addresses'] to work correctly.
 *
 * Enable this setting to get Drupal to determine the client IP from the
 * X-Forwarded-For header. If you are unsure about this setting, do not have a
 * reverse proxy, or Drupal operates in a shared hosting environment, this
 * setting should remain commented out.
 *
 * In order for this setting to be used you must specify every possible
 * reverse proxy IP address in $settings['reverse_proxy_addresses'].
 * If a complete list of reverse proxies is not available in your
 * environment (for example, if you use a CDN) you may set the
 * $_SERVER['REMOTE_ADDR'] variable directly in settings.php.
 * Be aware, however, that it is likely that this would allow IP
 * address spoofing unless more advanced precautions are taken.
 */
# $settings['reverse_proxy'] = TRUE;

/**
 * Reverse proxy addresses.
 *
 * Specify every reverse proxy IP address in your environment, as an array of
 * IPv4/IPv6 addresses or subnets in CIDR notation. This setting is required if
 * $settings['reverse_proxy'] is TRUE.
 */
# $settings['reverse_proxy_addresses'] = ['a.b.c.d', 'e.f.g.h/24', ...];

/**
 * Reverse proxy trusted headers.
 *
 * Sets which headers to trust from your reverse proxy.
 *
 * Common values are:
 * - \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_FOR
 * - \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_HOST
 * - \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PORT
 * - \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PROTO
 * - \Symfony\Component\HttpFoundation\Request::HEADER_FORWARDED
 *
 * Note the default value of
 * @code
 * \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_FOR | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_HOST | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PORT | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PROTO | \Symfony\Component\HttpFoundation\Request::HEADER_FORWARDED
 * @endcode
 * is not secure by default. The value should be set to only the specific
 * headers the reverse proxy uses. For example:
 * @code
 * \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_FOR | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_HOST | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PORT | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PROTO
 * @endcode
 * This would trust the following headers:
 * - X_FORWARDED_FOR
 * - X_FORWARDED_HOST
 * - X_FORWARDED_PROTO
 * - X_FORWARDED_PORT
 *
 * @see \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_FOR
 * @see \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_HOST
 * @see \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PORT
 * @see \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PROTO
 * @see \Symfony\Component\HttpFoundation\Request::HEADER_FORWARDED
 * @see \Symfony\Component\HttpFoundation\Request::setTrustedProxies
 */
# $settings['reverse_proxy_trusted_headers'] = \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_FOR | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_HOST | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PORT | \Symfony\Component\HttpFoundation\Request::HEADER_X_FORWARDED_PROTO | \Symfony\Component\HttpFoundation\Request::HEADER_FORWARDED;


/**
 * Page caching:
 *
 * By default, Drupal sends a "Vary: Cookie" HTTP header for anonymous page
 * views. This tells a HTTP proxy that it may return a page from its local
 * cache without contacting the web server, if the user sends the same Cookie
 * header as the user who originally requested the cached page. Without "Vary:
 * Cookie", authenticated users would also be served the anonymous page from
 * the cache. If the site has mostly anonymous users except a few known
 * editors/administrators, the Vary header can be omitted. This allows for
 * better caching in HTTP proxies (including reverse proxies), i.e. even if
 * clients send different cookies, they still get content served from the cache.
 * However, authenticated users should access the site directly (i.e. not use an
 * HTTP proxy, and bypass the reverse proxy if one is used) in order to avoid
 * getting cached pages from the proxy.
 */
# $settings['omit_vary_cookie'] = TRUE;


/**
 * Cache TTL for client error (4xx) responses.
 *
 * Items cached per-URL tend to result in a large number of cache items, and
 * this can be problematic on 404 pages which by their nature are unbounded. A
 * fixed TTL can be set for these items, defaulting to one hour, so that cache
 * backends which do not support LRU can purge older entries. To disable caching
 * of client error responses set the value to 0. Currently applies only to
 * page_cache module.
 */
# $settings['cache_ttl_4xx'] = 3600;

/**
 * Expiration of cached forms.
 *
 * Drupal's Form API stores details of forms in a cache and these entries are
 * kept for at least 6 hours by default. Expired entries are cleared by cron.
 *
 * @see \Drupal\Core\Form\FormCache::setCache()
 */
# $settings['form_cache_expiration'] = 21600;

/**
 * Class Loader.
 *
 * If the APCu extension is detected, the classloader will be optimized to use
 * it. Set to FALSE to disable this.
 *
 * @see https://getcomposer.org/doc/articles/autoloader-optimization.md
 */
# $settings['class_loader_auto_detect'] = FALSE;

/**
 * Authorized file system operations:
 *
 * The Update Manager module included with Drupal provides a mechanism for
 * site administrators to securely install missing updates for the site
 * directly through the web user interface. On securely-configured servers,
 * the Update manager will require the administrator to provide SSH or FTP
 * credentials before allowing the installation to proceed; this allows the
 * site to update the new files as the user who owns all the Drupal files,
 * instead of as the user the webserver is running as. On servers where the
 * webserver user is itself the owner of the Drupal files, the administrator
 * will not be prompted for SSH or FTP credentials (note that these server
 * setups are common on shared hosting, but are inherently insecure).
 *
 * Some sites might wish to disable the above functionality, and only update
 * the code directly via SSH or FTP themselves. This setting completely
 * disables all functionality related to these authorized file operations.
 *
 * @see https://www.drupal.org/node/244924
 *
 * Remove the leading hash signs to disable.
 */
# $settings['allow_authorize_operations'] = FALSE;

/**
 * Default mode for directories and files written by Drupal.
 *
 * Value should be in PHP Octal Notation, with leading zero.
 */
# $settings['file_chmod_directory'] = 0775;
# $settings['file_chmod_file'] = 0664;

/**
 * Optimized assets path:
 *
 * A local file system path where optimized assets will be stored. This directory
 * must exist and be writable by Drupal. This directory must be relative to
 * the Drupal installation directory and be accessible over the web.
 */
# $settings['file_assets_path'] = 'sites/default/files';

/**
 * Public file base URL:
 *
 * An alternative base URL to be used for serving public files. This must
 * include any leading directory path.
 *
 * A different value from the domain used by Drupal to be used for accessing
 * public files. This can be used for a simple CDN integration, or to improve
 * security by serving user-uploaded files from a different domain or subdomain
 * pointing to the same server. Do not include a trailing slash.
 */
# $settings['file_public_base_url'] = 'http://downloads.example.com/files';

/**
 * Public file path:
 *
 * A local file system path where public files will be stored. This directory
 * must exist and be writable by Drupal. This directory must be relative to
 * the Drupal installation directory and be accessible over the web.
 */
# $settings['file_public_path'] = 'sites/default/files';

/**
 * Additional public file schemes:
 *
 * Public schemes are URI schemes that allow download access to all users for
 * all files within that scheme.
 *
 * The "public" scheme is always public, and the "private" scheme is always
 * private, but other schemes, such as "https", "s3", "example", or others,
 * can be either public or private depending on the site. By default, they're
 * private, and access to individual files is controlled via
 * hook_file_download().
 *
 * Typically, if a scheme should be public, a module makes it public by
 * implementing hook_file_download(), and granting access to all users for all
 * files. This could be either the same module that provides the stream wrapper
 * for the scheme, or a different module that decides to make the scheme
 * public. However, in cases where a site needs to make a scheme public, but
 * is unable to add code in a module to do so, the scheme may be added to this
 * variable, the result of which is that system_file_download() grants public
 * access to all files within that scheme.
 */
# $settings['file_additional_public_schemes'] = ['example'];

/**
 * File schemes whose paths should not be normalized:
 *
 * Normally, Drupal normalizes '/./' and '/../' segments in file URIs in order
 * to prevent unintended file access. For example, 'private://css/../image.png'
 * is normalized to 'private://image.png' before checking access to the file.
 *
 * On Windows, Drupal also replaces '\' with '/' in URIs for the local
 * filesystem.
 *
 * If file URIs with one or more scheme should not be normalized like this, then
 * list the schemes here. For example, if 'porcelain://china/./plate.png' should
 * not be normalized to 'porcelain://china/plate.png', then add 'porcelain' to
 * this array. In this case, make sure that the module providing the 'porcelain'
 * scheme does not allow unintended file access when using '/../' to move up the
 * directory tree.
 */
# $settings['file_sa_core_2023_005_schemes'] = ['porcelain'];

/**
 * Configuration for phpinfo() admin status report.
 *
 * Drupal's admin UI includes a report at admin/reports/status/php which shows
 * the output of phpinfo(). The full output can contain sensitive information
 * so by default Drupal removes some sections.
 *
 * This behavior can be configured by setting this variable to a different
 * value corresponding to the flags parameter of phpinfo().
 *
 * If you need to expose more information in the report - for example to debug a
 * problem - consider doing so temporarily.
 *
 * @see https://www.php.net/manual/function.phpinfo.php
 */
# $settings['sa_core_2023_004_phpinfo_flags'] = ~ (INFO_VARIABLES | INFO_ENVIRONMENT);

/**
 * Private file path:
 *
 * A local file system path where private files will be stored. This directory
 * must be absolute, outside of the Drupal installation directory and not
 * accessible over the web.
 *
 * Note: Caches need to be cleared when this value is changed to make the
 * private:// stream wrapper available to the system.
 *
 * See https://www.drupal.org/documentation/modules/file for more information
 * about securing private files.
 */
# $settings['file_private_path'] = '';

/**
 * Temporary file path:
 *
 * A local file system path where temporary files will be stored. This directory
 * must be absolute, outside of the Drupal installation directory and not
 * accessible over the web.
 *
 * If this is not set, the default for the operating system will be used.
 *
 * @see \Drupal\Component\FileSystem\FileSystem::getOsTemporaryDirectory()
 */
# $settings['file_temp_path'] = '/tmp';

/**
 * Session write interval:
 *
 * Set the minimum interval between each session write to database.
 * For performance reasons it defaults to 180.
 */
# $settings['session_write_interval'] = 180;

/**
 * String overrides:
 *
 * To override specific strings on your site with or without enabling the Locale
 * module, add an entry to this list. This functionality allows you to change
 * a small number of your site's default English language interface strings.
 *
 * Remove the leading hash signs to enable.
 *
 * The "en" part of the variable name, is dynamic and can be any langcode of
 * any added language. (eg locale_custom_strings_de for german).
 */
# $settings['locale_custom_strings_en'][''] = [
#   'Home' => 'Front page',
#   '@count min' => '@count minutes',
# ];

/**
 * A custom theme for the offline page:
 *
 * This applies when the site is explicitly set to maintenance mode through the
 * administration page or when the database is inactive due to an error.
 * The template file should also be copied into the theme. It is located inside
 * 'core/modules/system/templates/maintenance-page.html.twig'.
 *
 * Note: This setting does not apply to installation and update pages.
 */
# $settings['maintenance_theme'] = 'claro';

/**
 * PHP settings:
 *
 * To see what PHP settings are possible, including whether they can be set at
 * runtime (by using ini_set()), read the PHP documentation:
 * http://php.net/manual/ini.list.php
 * See \Drupal\Core\DrupalKernel::bootEnvironment() for required runtime
 * settings and the .htaccess file for non-runtime settings.
 * Settings defined there should not be duplicated here so as to avoid conflict
 * issues.
 */

/**
 * If you encounter a situation where users post a large amount of text, and
 * the result is stripped out upon viewing but can still be edited, Drupal's
 * output filter may not have sufficient memory to process it.  If you
 * experience this issue, you may wish to uncomment the following two lines
 * and increase the limits of these variables.  For more information, see
 * http://php.net/manual/pcre.configuration.php.
 */
# ini_set('pcre.backtrack_limit', 200000);
# ini_set('pcre.recursion_limit', 200000);

/**
 * Configuration overrides.
 *
 * To globally override specific configuration values for this site,
 * set them here. You usually don't need to use this feature. This is
 * useful in a configuration file for a vhost or directory, rather than
 * the default settings.php.
 *
 * Note that any values you provide in these variable overrides will not be
 * viewable from the Drupal administration interface. The administration
 * interface displays the values stored in configuration so that you can stage
 * changes to other environments that don't have the overrides.
 *
 * There are particular configuration values that are risky to override. For
 * example, overriding the list of installed modules in 'core.extension' is not
 * supported as module install or uninstall has not occurred. Other examples
 * include field storage configuration, because it has effects on database
 * structure, and 'core.menu.static_menu_link_overrides' since this is cached in
 * a way that is not config override aware. Also, note that changing
 * configuration values in settings.php will not fire any of the configuration
 * change events.
 */
# $config['system.site']['name'] = 'My Drupal site';
# $config['user.settings']['anonymous'] = 'Visitor';

/**
 * Load services definition file.
 */
$settings['container_yamls'][] = $app_root . '/' . $site_path . '/services.yml';

/**
 * Override the default service container class.
 *
 * This is useful for example to trace the service container for performance
 * tracking purposes, for testing a service container with an error condition or
 * to test a service container that throws an exception.
 */
# $settings['container_base_class'] = '\Drupal\Core\DependencyInjection\Container';

/**
 * Override the default yaml parser class.
 *
 * Provide a fully qualified class name here if you would like to provide an
 * alternate implementation YAML parser. The class must implement the
 * \Drupal\Component\Serialization\SerializationInterface interface.
 */
# $settings['yaml_parser_class'] = NULL;

/**
 * Trusted host configuration.
 *
 * Drupal core can use the Symfony trusted host mechanism to prevent HTTP Host
 * header spoofing.
 *
 * To enable the trusted host mechanism, you enable your allowable hosts
 * in $settings['trusted_host_patterns']. This should be an array of regular
 * expression patterns, without delimiters, representing the hosts you would
 * like to allow.
 *
 * For example:
 * @code
 * $settings['trusted_host_patterns'] = [
 *   '^www\.example\.com$',
 * ];
 * @endcode
 * will allow the site to only run from www.example.com.
 *
 * If you are running multisite, or if you are running your site from
 * different domain names (eg, you don't redirect http://www.example.com to
 * http://example.com), you should specify all of the host patterns that are
 * allowed by your site.
 *
 * For example:
 * @code
 * $settings['trusted_host_patterns'] = [
 *   '^example\.com$',
 *   '^.+\.example\.com$',
 *   '^example\.org$',
 *   '^.+\.example\.org$',
 * ];
 * @endcode
 * will allow the site to run off of all variants of example.com and
 * example.org, with all subdomains included.
 *
 * @see https://www.drupal.org/docs/installing-drupal/trusted-host-settings
 */
# $settings['trusted_host_patterns'] = [];

/**
 * The default list of directories that will be ignored by Drupal's file API.
 *
 * By default ignore node_modules and bower_components folders to avoid issues
 * with common frontend tools and recursive scanning of directories looking for
 * extensions.
 *
 * @see \Drupal\Core\File\FileSystemInterface::scanDirectory()
 * @see \Drupal\Core\Extension\ExtensionDiscovery::scanDirectory()
 */
$settings['file_scan_ignore_directories'] = [
  'node_modules',
  'bower_components',
];

/**
 * The default number of entities to update in a batch process.
 *
 * This is used by update and post-update functions that need to go through and
 * change all the entities on a site, so it is useful to increase this number
 * if your hosting configuration (i.e. RAM allocation, CPU speed) allows for a
 * larger number of entities to be processed in a single batch run.
 */
$settings['entity_update_batch_size'] = 50;

/**
 * Entity update backup.
 *
 * This is used to inform the entity storage handler that the backup tables as
 * well as the original entity type and field storage definitions should be
 * retained after a successful entity update process.
 */
$settings['entity_update_backup'] = TRUE;

/**
 * Node migration type.
 *
 * This is used to force the migration system to use the classic node migrations
 * instead of the default complete node migrations. The migration system will
 * use the classic node migration only if there are existing migrate_map tables
 * for the classic node migrations and they contain data. These tables may not
 * exist if you are developing custom migrations and do not want to use the
 * complete node migrations. Set this to TRUE to force the use of the classic
 * node migrations.
 */
$settings['migrate_node_migrate_type_classic'] = FALSE;

/**
 * The default settings for migration sources.
 *
 * These settings are used as the default settings on the Credential form at
 * /upgrade/credentials.
 *
 * - migrate_source_version - The version of the source database. This can be
 *   '6' or '7'. Defaults to '7'.
 * - migrate_source_connection - The key in the $databases array for the source
 *   site.
 * - migrate_file_public_path - The location of the source Drupal 6 or Drupal 7
 *   public files. This can be a local file directory containing the source
 *   Drupal 6 or Drupal 7 site (e.g /var/www/docroot), or the site address
 *   (e.g http://example.com).
 * - migrate_file_private_path - The location of the source Drupal 7 private
 *   files. This can be a local file directory containing the source Drupal 7
 *   site (e.g /var/www/docroot), or empty to use the same value as Public
 *   files directory.
 *
 * Sample configuration for a drupal 6 source site with the source files in a
 * local directory.
 *
 * @code
 * $settings['migrate_source_version'] = '6';
 * $settings['migrate_source_connection'] = 'migrate';
 * $settings['migrate_file_public_path'] = '/var/www/drupal6';
 * @endcode
 *
 * Sample configuration for a drupal 7 source site with public source files on
 * the source site and the private files in a local directory.
 *
 * @code
 * $settings['migrate_source_version'] = '7';
 * $settings['migrate_source_connection'] = 'migrate';
 * $settings['migrate_file_public_path'] = 'https://drupal7.com';
 * $settings['migrate_file_private_path'] = '/var/www/drupal7';
 * @endcode
 */
# $settings['migrate_source_connection'] = '';
# $settings['migrate_source_version'] = '';
# $settings['migrate_file_public_path'] = '';
# $settings['migrate_file_private_path'] = '';

/**
 * Load local development override configuration, if available.
 *
 * Create a settings.local.php file to override variables on secondary (staging,
 * development, etc.) installations of this site.
 *
 * Typical uses of settings.local.php include:
 * - Disabling caching.
 * - Disabling JavaScript/CSS compression.
 * - Rerouting outgoing emails.
 *
 * Keep this code block at the end of this file to take full effect.
 */
#
# if (file_exists($app_root . '/' . $site_path . '/settings.local.php')) {
#   include $app_root . '/' . $site_path . '/settings.local.php';
# }
