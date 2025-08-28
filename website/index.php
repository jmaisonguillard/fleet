<?php
// Fleet PHP Demo Page
$php_version = phpversion();
$server_name = $_SERVER['SERVER_NAME'] ?? 'localhost';
$request_time = date('Y-m-d H:i:s');
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Fleet PHP - <?= htmlspecialchars($server_name) ?></title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            margin: 0;
            padding: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        .container {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 20px;
            padding: 2rem;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 600px;
            width: 90%;
        }
        h1 {
            color: #333;
            margin: 0 0 1rem 0;
            font-size: 2rem;
        }
        .info {
            background: #f7f7f7;
            padding: 1rem;
            border-radius: 10px;
            margin: 1rem 0;
        }
        .info-row {
            display: flex;
            justify-content: space-between;
            padding: 0.5rem 0;
            border-bottom: 1px solid #e0e0e0;
        }
        .info-row:last-child {
            border-bottom: none;
        }
        .label {
            font-weight: 600;
            color: #555;
        }
        .value {
            color: #764ba2;
            font-family: 'Courier New', monospace;
        }
        .success {
            background: #d4edda;
            color: #155724;
            padding: 1rem;
            border-radius: 5px;
            margin: 1rem 0;
        }
        footer {
            text-align: center;
            margin-top: 2rem;
            color: #777;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Fleet PHP is Running!</h1>
        
        <div class="success">
            ✅ PHP-FPM and nginx are successfully configured and serving your application
        </div>
        
        <div class="info">
            <div class="info-row">
                <span class="label">PHP Version:</span>
                <span class="value"><?= htmlspecialchars($php_version) ?></span>
            </div>
            <div class="info-row">
                <span class="label">Server Name:</span>
                <span class="value"><?= htmlspecialchars($server_name) ?></span>
            </div>
            <div class="info-row">
                <span class="label">Server Software:</span>
                <span class="value"><?= htmlspecialchars($_SERVER['SERVER_SOFTWARE'] ?? 'nginx/php-fpm') ?></span>
            </div>
            <div class="info-row">
                <span class="label">Request Time:</span>
                <span class="value"><?= htmlspecialchars($request_time) ?></span>
            </div>
            <div class="info-row">
                <span class="label">Document Root:</span>
                <span class="value"><?= htmlspecialchars($_SERVER['DOCUMENT_ROOT']) ?></span>
            </div>
        </div>
        
        <footer>
            Powered by Fleet - Simple Docker Service Orchestration<br>
            PHP <?= htmlspecialchars($php_version) ?> with nginx and PHP-FPM
        </footer>
    </div>
</body>
</html>