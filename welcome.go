package routix

import (
	"fmt"
)

func GetWelcomePage(projectName string) string {
	if projectName == "" {
		projectName = "Routix"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Routix Framework</title>
    <style>
        :root {
            --primary: #00dc82;
            --primary-dark: #00b368;
            --bg-dark: #0f172a;
            --bg-card: #1e293b;
            --text: #e2e8f0;
            --text-muted: #94a3b8;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: var(--bg-dark);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            color: var(--text);
            overflow: hidden;
        }
        
        .bg-gradient {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: 
                radial-gradient(ellipse at top, rgba(0, 220, 130, 0.15) 0%%, transparent 50%%),
                radial-gradient(ellipse at bottom right, rgba(59, 130, 246, 0.1) 0%%, transparent 50%%);
            z-index: 0;
        }
        
        .container {
            position: relative;
            z-index: 1;
            text-align: center;
            padding: 2rem;
            max-width: 600px;
        }
        
        .logo {
            width: 120px;
            height: 120px;
            margin: 0 auto 2rem;
            background: linear-gradient(135deg, var(--primary) 0%%, var(--primary-dark) 100%%);
            border-radius: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 48px;
            font-weight: bold;
            color: white;
            box-shadow: 0 20px 40px rgba(0, 220, 130, 0.3);
            animation: float 3s ease-in-out infinite;
        }
        
        @keyframes float {
            0%%, 100%% { transform: translateY(0); }
            50%% { transform: translateY(-10px); }
        }
        
        h1 {
            font-size: 3rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
            background: linear-gradient(135deg, var(--primary), #3b82f6);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        
        .subtitle {
            font-size: 1.25rem;
            color: var(--text-muted);
            margin-bottom: 2rem;
        }
        
        .version {
            display: inline-block;
            background: var(--bg-card);
            padding: 0.5rem 1rem;
            border-radius: 9999px;
            font-size: 0.875rem;
            color: var(--primary);
            border: 1px solid rgba(0, 220, 130, 0.3);
            margin-bottom: 2rem;
        }
        
        .buttons {
            display: flex;
            gap: 1rem;
            justify-content: center;
            flex-wrap: wrap;
            margin-bottom: 3rem;
        }
        
        .btn {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.875rem 1.5rem;
            border-radius: 12px;
            font-size: 1rem;
            font-weight: 500;
            text-decoration: none;
            transition: all 0.2s ease;
        }
        
        .btn-primary {
            background: var(--primary);
            color: var(--bg-dark);
        }
        
        .btn-primary:hover {
            background: var(--primary-dark);
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(0, 220, 130, 0.3);
        }
        
        .btn-secondary {
            background: var(--bg-card);
            color: var(--text);
            border: 1px solid rgba(255, 255, 255, 0.1);
        }
        
        .btn-secondary:hover {
            background: #2d3a4f;
            transform: translateY(-2px);
        }
        
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
            gap: 1rem;
            margin-bottom: 3rem;
        }
        
        .feature {
            background: var(--bg-card);
            padding: 1.25rem;
            border-radius: 12px;
            border: 1px solid rgba(255, 255, 255, 0.05);
        }
        
        .feature-icon {
            font-size: 1.5rem;
            margin-bottom: 0.5rem;
        }
        
        .feature-title {
            font-size: 0.875rem;
            font-weight: 600;
            color: var(--text);
        }
        
        .footer {
            position: fixed;
            bottom: 2rem;
            left: 0;
            right: 0;
            text-align: center;
            z-index: 1;
        }
        
        .powered-by {
            font-size: 0.875rem;
            color: var(--text-muted);
        }
        
        .powered-by a {
            color: var(--primary);
            text-decoration: none;
            font-weight: 500;
        }
        
        .powered-by a:hover {
            text-decoration: underline;
        }
        
        @media (max-width: 480px) {
            h1 { font-size: 2rem; }
            .subtitle { font-size: 1rem; }
            .buttons { flex-direction: column; }
        }
    </style>
</head>
<body>
    <div class="bg-gradient"></div>
    
    <div class="container">
        <div class="logo">R</div>
        
        <h1>%s + Routix</h1>
        <p class="subtitle">High-Performance Go Web Framework</p>
        
        <span class="version">v0.3.8</span>
        
        <div class="buttons">
            <a href="https://github.com/ramusaaa/routix" class="btn btn-primary" target="_blank">
                📚 Read Docs
            </a>
            <a href="https://github.com/ramusaaa/routix" class="btn btn-secondary" target="_blank">
                ⭐ Star on GitHub
            </a>
        </div>
        
        <div class="features">
            <div class="feature">
                <div class="feature-icon">⚡</div>
                <div class="feature-title">Blazing Fast</div>
            </div>
            <div class="feature">
                <div class="feature-icon">🛠️</div>
                <div class="feature-title">Laravel-inspired CLI</div>
            </div>
            <div class="feature">
                <div class="feature-icon">🔒</div>
                <div class="feature-title">JWT Auth Ready</div>
            </div>
            <div class="feature">
                <div class="feature-icon">🗄️</div>
                <div class="feature-title">GORM Database</div>
            </div>
        </div>
    </div>
    
    <footer class="footer">
        <p class="powered-by">
            Powered by <a href="https://github.com/ramusaaa">Ramusa Software Corporation</a>
        </p>
    </footer>
</body>
</html>`, projectName, projectName)
}

func WelcomeHandler(projectName string) Handler {
	return func(c *Context) error {
		return c.HTML(200, GetWelcomePage(projectName))
	}
}
