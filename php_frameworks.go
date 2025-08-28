package main

import (
	"fmt"
	"os"
)

// SupportedPHPFrameworks lists all supported PHP frameworks
var SupportedPHPFrameworks = []string{
	"laravel",
	"symfony", 
	"wordpress",
	"drupal",
	"codeigniter",
	"slim",
	"lumen",
}

// detectPHPFramework auto-detects the PHP framework from project files
func detectPHPFramework(folder string) string {
	// Use the PHPConfigurator for framework detection
	configurator := NewPHPConfigurator()
	return configurator.DetectFramework(folder)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getNginxConfigForFramework returns the appropriate nginx configuration for a PHP framework
func getNginxConfigForFramework(serviceName, framework string) string {
	// Use the PHPConfigurator for nginx config generation
	configurator := NewPHPConfigurator()
	return configurator.GenerateNginxConfig(serviceName, framework)
}

// getNginxConfigForFrameworkWithVersion returns nginx config for framework with specific PHP version  
func getNginxConfigForFrameworkWithVersion(serviceName, framework, phpVersion string) string {
	// Suppress unused parameter warning - kept for compatibility
	_ = phpVersion
	// For compatibility, just use the standard method
	// The version is already handled in the runtime configuration
	return getNginxConfigForFramework(serviceName, framework)
}

// generateLaravelNginxConfig generates nginx config for Laravel
func generateLaravelNginxConfig(phpServiceName string) string {
	return fmt.Sprintf(`# Laravel configuration
server {
    listen 80;
    server_name _;
    root /var/www/html/public;
    
    index index.php index.html;
    
    charset utf-8;
    
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    
    location = /favicon.ico { access_log off; log_not_found off; }
    location = /robots.txt  { access_log off; log_not_found off; }
    
    error_page 404 /index.php;
    
    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass %s:9000;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        
        fastcgi_buffer_size 128k;
        fastcgi_buffers 256 16k;
        fastcgi_busy_buffers_size 256k;
    }
    
    location ~ /\.(?!well-known).* {
        deny all;
    }
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
}`, phpServiceName)
}

// generateSymfonyNginxConfig generates nginx config for Symfony
func generateSymfonyNginxConfig(phpServiceName string) string {
	return fmt.Sprintf(`# Symfony configuration
server {
    listen 80;
    server_name _;
    root /var/www/html/public;
    
    location / {
        try_files $uri /index.php$is_args$args;
    }
    
    location ~ ^/index\.php(/|$) {
        fastcgi_pass %s:9000;
        fastcgi_split_path_info ^(.+\.php)(/.*)$;
        include fastcgi_params;
        
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        fastcgi_param DOCUMENT_ROOT $realpath_root;
        
        # Prevents URIs that include the front controller
        internal;
    }
    
    # Return 404 for all other php files
    location ~ \.php$ {
        return 404;
    }
    
    # Security - hide .htaccess and .git
    location ~ /\.(ht|git|svn) {
        deny all;
    }
    
    # Assets
    location ~* \.(jpg|jpeg|gif|png|css|js|ico|xml)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
    
    error_log /var/log/nginx/symfony_error.log;
    access_log /var/log/nginx/symfony_access.log;
}`, phpServiceName)
}

// generateWordPressNginxConfig generates nginx config for WordPress
func generateWordPressNginxConfig(phpServiceName string) string {
	return fmt.Sprintf(`# WordPress configuration
server {
    listen 80;
    server_name _;
    root /var/www/html;
    
    index index.php index.html index.htm;
    
    # WordPress permalinks
    location / {
        try_files $uri $uri/ /index.php?$args;
    }
    
    # PHP handling
    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass %s:9000;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        
        # WordPress specific
        fastcgi_buffer_size 128k;
        fastcgi_buffers 256 16k;
        fastcgi_busy_buffers_size 256k;
        fastcgi_temp_file_write_size 256k;
        fastcgi_intercept_errors off;
    }
    
    # WordPress admin
    location ~* ^/wp-admin/.*\.php$ {
        try_files $uri =404;
        fastcgi_pass %s:9000;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
    }
    
    # Deny access to sensitive files
    location ~* /(?:uploads|files)/.*\.php$ {
        deny all;
    }
    
    location ~ /\.ht {
        deny all;
    }
    
    location = /xmlrpc.php {
        deny all;
    }
    
    # Media files
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires max;
        add_header Cache-Control "public, immutable";
        log_not_found off;
    }
    
    # Gzip
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
}`, phpServiceName, phpServiceName)
}

// generateDrupalNginxConfig generates nginx config for Drupal
func generateDrupalNginxConfig(phpServiceName string) string {
	return fmt.Sprintf(`# Drupal configuration
server {
    listen 80;
    server_name _;
    root /var/www/html;
    
    index index.php index.html;
    
    location = /favicon.ico {
        log_not_found off;
        access_log off;
    }
    
    location = /robots.txt {
        allow all;
        log_not_found off;
        access_log off;
    }
    
    # Block access to hidden files
    location ~ /\. {
        deny all;
        access_log off;
        log_not_found off;
    }
    
    location / {
        try_files $uri /index.php?$query_string;
    }
    
    location @rewrite {
        rewrite ^/(.*)$ /index.php?q=$1;
    }
    
    location ~ '\.php$|^/update.php' {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+?\.php)(|/.*)$;
        fastcgi_pass %s:9000;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        fastcgi_intercept_errors on;
    }
    
    # Fighting with Styles? This helps
    location ~ ^/sites/.*/files/styles/ {
        try_files $uri @rewrite;
    }
    
    # Handle private files
    location ~ ^(/[a-z\-]+)?/system/files/ {
        try_files $uri /index.php?$query_string;
    }
    
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        try_files $uri @rewrite;
        expires max;
        log_not_found off;
    }
}`, phpServiceName)
}

// generateCodeIgniterNginxConfig generates nginx config for CodeIgniter
func generateCodeIgniterNginxConfig(phpServiceName string) string {
	return fmt.Sprintf(`# CodeIgniter configuration
server {
    listen 80;
    server_name _;
    root /var/www/html/public;
    
    index index.php index.html;
    
    location / {
        try_files $uri $uri/ /index.php?/$request_uri;
    }
    
    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass %s:9000;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        
        fastcgi_buffer_size 128k;
        fastcgi_buffers 256 16k;
        fastcgi_busy_buffers_size 256k;
    }
    
    # Deny access to hidden files
    location ~ /\. {
        deny all;
        access_log off;
        log_not_found off;
    }
    
    # Security
    location ~* ^/(system|application|spark|tests|vendor)/.*\.(php|php3|php4|php5|phtml)$ {
        deny all;
    }
    
    # Static files
    location ~* \.(jpg|jpeg|gif|png|css|js|ico|xml)$ {
        expires 30d;
        add_header Cache-Control "public";
    }
}`, phpServiceName)
}

// generateSlimNginxConfig generates nginx config for Slim Framework
func generateSlimNginxConfig(phpServiceName string) string {
	return fmt.Sprintf(`# Slim configuration
server {
    listen 80;
    server_name _;
    root /var/www/html/public;
    
    index index.php;
    
    location / {
        try_files $uri /index.php$is_args$args;
    }
    
    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass %s:9000;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        
        fastcgi_buffer_size 128k;
        fastcgi_buffers 256 16k;
        fastcgi_busy_buffers_size 256k;
    }
    
    # Deny access to .htaccess
    location ~ /\.ht {
        deny all;
    }
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
}`, phpServiceName)
}