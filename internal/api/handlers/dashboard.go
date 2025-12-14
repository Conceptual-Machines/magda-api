package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Dashboard(c *gin.Context) {
	//nolint:lll // SVG paths are inherently long and cannot be shortened
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AIDEAS API Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .header {
            text-align: center;
            color: white;
            margin-bottom: 40px;
        }
        .logo {
            width: 80px;
            height: 80px;
            margin: 0 auto 20px;
            display: block;
        }
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(0,0,0,0.3);
        }
        .header p {
            opacity: 0.9;
            font-size: 1.1em;
        }
        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .card {
            background: white;
            border-radius: 12px;
            padding: 24px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }
        .card:hover {
            transform: translateY(-4px);
            box-shadow: 0 8px 12px rgba(0,0,0,0.15);
        }
        .card h2 {
            color: #333;
            font-size: 1.2em;
            margin-bottom: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .card .value {
            font-size: 2em;
            font-weight: bold;
            color: #667eea;
            margin-bottom: 8px;
        }
        .card .label {
            color: #666;
            font-size: 0.9em;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
        }
        .status-healthy {
            background: #d4edda;
            color: #155724;
        }
        .status-error {
            background: #f8d7da;
            color: #721c24;
        }
        .metric-row {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #eee;
        }
        .metric-row:last-child {
            border-bottom: none;
        }
        .metric-label {
            color: #666;
        }
        .metric-value {
            font-weight: 600;
            color: #333;
        }
        .footer {
            text-align: center;
            color: white;
            margin-top: 40px;
            opacity: 0.8;
        }
        .loading {
            text-align: center;
            color: white;
            font-size: 1.2em;
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 16px;
            border-radius: 8px;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
            <div class="header">
                <svg class="logo" viewBox="0 0 476 462" xmlns="http://www.w3.org/2000/svg">
                    <style type="text/css">
                        .st0{fill:#080E18;}
                        .st1{fill:#1B8AF3;}
                        .st2{fill:#FFFFFF;}
                        .st3{fill:#323232;}
                    </style>
                    <g>
                        <g>
                            <g>
                                <path class="st2" d="M242.93,244.24H222.4l1.75,85.87h20.56L242.93,244.24z M240.92,147.98c-0.19-10.73-2.92-16.09-8.17-16.09
                                    h-21.14c-4.95,0-7.92,4.6-8.89,13.79l-12.84,99.68c-0.49,4.79-0.95,9.53-1.38,14.22c-0.44,4.7-0.9,9.44-1.39,14.23h-1.9
                                    c-0.49-4.79-0.92-9.53-1.31-14.23c-0.39-4.69-0.87-9.33-1.46-13.92l-12.98-99.98c-1.07-9.19-3.99-13.79-8.75-13.79h-20.85
                                    c-5.25,0-8.02,5.36-8.32,16.09l-3.65,182.13h20.56l3.07-164.61h2.48l16.33,123.81c0.98,9.19,3.94,13.79,8.9,13.79h14
                                    c4.96,0,7.87-4.6,8.75-13.79l16.19-123.81h2.62l1.15,56.64h20.52L240.92,147.98z"/>
                                <path class="st2" d="M266.54,244.24l7.92,33.29h52.21l-0.27-33.29H266.54z M318.93,143.38c-0.48-3.64-1.43-6.46-2.84-8.47
                                    c-1.41-2.02-3.04-3.02-4.89-3.02h-21.14c-1.84,0-3.47,1-4.88,3.02c-1.41,2.01-2.36,4.83-2.85,8.47l-12.3,78.76h21.1l5.35-37.97
                                    c0.49-3.06,0.89-6.27,1.24-9.62c0.33-3.34,0.65-6.56,0.94-9.62h3.94c0.29,3.07,0.64,6.28,1.02,9.62
                                    c0.39,3.35,0.77,6.57,1.17,9.62l8.54,60.62l4.61,32.75l7.4,52.58h22.75L318.93,143.38z M253.17,330.11h22.75l7.4-52.58
                                    l4.62-32.75l0.07-0.54h-21.44L253.17,330.11z"/>
                            </g>
                            <g>
                                <path class="st2" d="M351.17,387.1H124.83c-23.67,0-42.93-19.26-42.93-42.93V117.83c0-23.67,19.26-42.93,42.93-42.93h226.34
                                    c23.67,0,42.93,19.26,42.93,42.93v226.34C394.1,367.84,374.84,387.1,351.17,387.1z M124.83,93.32
                                    c-13.51,0-24.51,10.99-24.51,24.51v226.34c0,13.51,10.99,24.51,24.51,24.51h226.34c13.51,0,24.51-10.99,24.51-24.51V117.83
                                    c0-13.51-10.99-24.51-24.51-24.51H124.83z"/>
                            </g>
                        </g>
                    </g>
                </svg>
                <h1>🎵 AIDEAS API Dashboard</h1>
                <p>Real-time monitoring and metrics</p>
            </div>

        <div id="loading" class="loading">Loading metrics...</div>
        <div id="error" class="error" style="display: none;"></div>
        <div id="dashboard" style="display: none;">
            <div class="grid">
                <div class="card">
                    <h2>📊 Status</h2>
                    <div class="value">
                        <span id="status" class="status-badge status-healthy">Healthy</span>
                    </div>
                    <div class="label">Last updated: <span id="timestamp">-</span></div>
                </div>

                <div class="card">
                    <h2>⏱️ Uptime</h2>
                    <div class="value" id="uptime">-</div>
                    <div class="label">Since last restart</div>
                </div>

                <div class="card">
                    <h2>💾 Memory</h2>
                    <div class="value" id="memory">-</div>
                    <div class="label">Current allocation</div>
                </div>
            </div>

            <div class="grid">
                <div class="card">
                    <h2>🔄 Goroutines</h2>
                    <div class="value" id="goroutines">-</div>
                    <div class="label">Active concurrent tasks</div>
                </div>

                <div class="card">
                    <h2>🖥️ System Info</h2>
                    <div class="metric-row">
                        <span class="metric-label">Go Version</span>
                        <span class="metric-value" id="go-version">-</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Total Memory</span>
                        <span class="metric-value" id="mem-total">-</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">GC Runs</span>
                        <span class="metric-value" id="num-gc">-</span>
                    </div>
                </div>

                <div class="card">
                    <h2>🎵 MCP Server</h2>
                    <div class="metric-row">
                        <span class="metric-label">Status</span>
                        <span class="metric-value" id="mcp-enabled">-</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">URL</span>
                        <span class="metric-value" id="mcp-url" style="font-size: 0.85em;">-</span>
                    </div>
                </div>
            </div>

            <div class="grid">
                <div class="card">
                    <h2>📦 Version Info</h2>
                    <div class="metric-row">
                        <span class="metric-label">Version</span>
                        <span class="metric-value" id="app-version">-</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Started At</span>
                        <span class="metric-value" id="start-time">-</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Environment</span>
                        <span class="metric-value">Production</span>
                    </div>
                </div>

                <div class="card">
                    <h2>🔗 Quick Links</h2>
                    <div class="metric-row">
                        <span class="metric-label">Health Check</span>
                        <a href="/health" target="_blank" class="metric-value">View</a>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">MCP Status</span>
                        <a href="/mcp/status" target="_blank" class="metric-value">View</a>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Metrics API</span>
                        <a href="/api/metrics" target="_blank" class="metric-value">View JSON</a>
                    </div>
                </div>
            </div>
        </div>

        <div class="footer">
            <p>Auto-refreshes every 5 seconds</p>
        </div>
    </div>

    <script>
        async function fetchMetrics() {
            try {
                const response = await fetch('/api/metrics');
                if (!response.ok) throw new Error('Failed to fetch metrics');

                const data = await response.json();

                // Update UI
                document.getElementById('loading').style.display = 'none';
                document.getElementById('error').style.display = 'none';
                document.getElementById('dashboard').style.display = 'block';

                // Status
                const statusBadge = document.getElementById('status');
                statusBadge.textContent = data.status.charAt(0).toUpperCase() + data.status.slice(1);
                statusBadge.className = 'status-badge ' + (data.status === 'healthy' ? 'status-healthy' : 'status-error');

                // Timestamp
                const timestamp = new Date(data.timestamp);
                document.getElementById('timestamp').textContent = timestamp.toLocaleTimeString();

                // Uptime
                document.getElementById('uptime').textContent = data.uptime;

                // Memory
                document.getElementById('memory').textContent = data.system.mem_alloc_mb + ' MB';

                // Goroutines
                document.getElementById('goroutines').textContent = data.system.num_goroutine;

                // System info
                document.getElementById('go-version').textContent = data.system.go_version;
                document.getElementById('mem-total').textContent = data.system.mem_total_mb + ' MB';
                document.getElementById('num-gc').textContent = data.system.num_gc;

                // Version info
                document.getElementById('app-version').textContent = data.version.substring(0, 8);
                const startTime = new Date(data.start_time);
                document.getElementById('start-time').textContent = startTime.toLocaleString();

                // MCP info
                document.getElementById('mcp-enabled').textContent = data.api.mcp.enabled ? '✅ Enabled' : '❌ Disabled';
                document.getElementById('mcp-url').textContent = data.api.mcp.url || 'Not configured';

            } catch (error) {
                document.getElementById('loading').style.display = 'none';
                document.getElementById('dashboard').style.display = 'none';
                const errorDiv = document.getElementById('error');
                errorDiv.textContent = '❌ Error loading metrics: ' + error.message;
                errorDiv.style.display = 'block';
            }
        }

        // Initial load
        fetchMetrics();

        // Auto-refresh every 5 seconds
        setInterval(fetchMetrics, 5000);
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
