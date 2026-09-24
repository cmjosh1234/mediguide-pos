package app

const swaggerChooserHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MediGuide Swagger</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f3f4f6;
      --panel: #ffffff;
      --border: #d1d5db;
      --text: #111827;
      --muted: #4b5563;
      --accent: #0f766e;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      background:
        radial-gradient(circle at top left, rgba(15,118,110,0.12), transparent 28rem),
        linear-gradient(180deg, #f8fafc 0%, var(--bg) 100%);
      color: var(--text);
      font-family: "IBM Plex Sans", "Segoe UI", sans-serif;
    }
    .panel {
      width: min(30rem, calc(100vw - 2rem));
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 18px;
      padding: 1.5rem;
      box-shadow: 0 18px 40px rgba(15, 23, 42, 0.08);
    }
    h1 {
      margin: 0 0 0.5rem;
      font-size: 1.4rem;
    }
    p {
      margin: 0 0 1rem;
      color: var(--muted);
      line-height: 1.5;
    }
    label {
      display: block;
      margin-bottom: 0.5rem;
      font-weight: 600;
    }
    select, button {
      width: 100%;
      border-radius: 12px;
      border: 1px solid var(--border);
      font: inherit;
    }
    select {
      padding: 0.9rem 1rem;
      background: #fff;
      margin-bottom: 0.9rem;
    }
    button {
      padding: 0.9rem 1rem;
      background: var(--accent);
      color: #fff;
      border-color: var(--accent);
      font-weight: 700;
      cursor: pointer;
    }
    .links {
      margin-top: 1rem;
      display: flex;
      gap: 0.75rem;
      flex-wrap: wrap;
    }
    a {
      color: var(--accent);
      text-decoration: none;
      font-weight: 600;
    }
  </style>
</head>
<body>
  <main class="panel">
    <h1>Swagger Docs</h1>
    <p>Select which route set you want to inspect. <code>v1</code> shows legacy compatibility APIs, while <code>v2</code> shows the current backend API.</p>
    <label for="swagger-version">API Version</label>
    <select id="swagger-version">
      <option value="/swagger/v1/index.html">v1 legacy routes</option>
      <option value="/swagger/v2/index.html" selected>v2 current routes</option>
      <option value="/swagger/all/index.html">all routes</option>
    </select>
    <button type="button" onclick="window.location.href=document.getElementById('swagger-version').value">Open Swagger UI</button>
    <div class="links">
      <a href="/swagger/v1/index.html">Open v1</a>
      <a href="/swagger/v2/index.html">Open v2</a>
      <a href="/swagger/all/index.html">Open all</a>
    </div>
  </main>
</body>
</html>`
