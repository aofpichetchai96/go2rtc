document.head.innerHTML += `
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="icon" type="image/png" href="cctv-icon.png">
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
<style>
    :root {
        --bg-color: #0d1117;
        --card-bg: rgba(22, 27, 34, 0.7);
        --text-primary: #e6edf3;
        --text-secondary: #8b949e;
        --accent: #00d9b5;
        --accent-hover: #00b396;
        --border-color: #30363d;
        --nav-bg: rgba(13, 17, 23, 0.8);
        --glass-border: rgba(255, 255, 255, 0.1);
        --danger: #f85149;
    }

    * {
        box-sizing: border-box;
    }

    body {
        background-color: var(--bg-color);
        color: var(--text-primary);
        display: flex;
        flex-direction: column;
        font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
        margin: 0;
        min-height: 100vh;
        line-height: 1.5;
    }

    /* navigation block */
    header {
        position: sticky;
        top: 0;
        z-index: 1000;
        backdrop-filter: blur(12px);
        -webkit-backdrop-filter: blur(12px);
        background-color: var(--nav-bg);
        border-bottom: 1px solid var(--border-color);
    }

    nav {
        max-width: 1200px;
        margin: 0 auto;
        display: flex;
        align-items: center;
        padding: 0 1rem;
    }

    nav a {
        display: block;
        color: var(--text-secondary);
        text-align: center;
        padding: 1.25rem 1rem;
        text-decoration: none;
        font-size: 0.95rem;
        font-weight: 500;
        transition: all 0.2s ease;
        border-bottom: 2px solid transparent;
    }

    nav a:hover {
        color: var(--text-primary);
    }

    nav a.active {
        color: var(--accent);
        border-bottom-color: var(--accent);
    }

    nav a b {
        color: var(--accent);
        font-size: 1.2rem;
        font-weight: 700;
        margin-right: 1.5rem;
    }

    /* main block */
    main {
        max-width: 1200px;
        margin: 0 auto;
        width: 100%;
        padding: 2rem 1rem;
        display: flex;
        flex-direction: column;
        gap: 2rem;
    }

    h1, h2, h3 {
        margin-top: 0;
        color: var(--text-primary);
    }

    /* checkbox */
    label {
        display: flex;
        gap: 8px;
        align-items: center;
        cursor: pointer;
        color: var(--text-secondary);
        font-size: 0.9rem;
    }

    input[type="checkbox"] {
        width: 18px;
        height: 18px;
        cursor: pointer;
        accent-color: var(--accent);
    }

    /* form */
    form {
        display: flex;
        flex-wrap: wrap;
        gap: 1rem;
    }

    input[type="text"], input[type="email"], input[type="password"], select {
        background-color: rgba(0, 0, 0, 0.2);
        color: var(--text-primary);
        padding: 0.75rem 1rem;
        border: 1px solid var(--border-color);
        border-radius: 8px;
        font-size: 0.95rem;
        outline: none;
        transition: border-color 0.2s;
    }

    input[type="text"]:focus {
        border-color: var(--accent);
    }

    button {
        background-color: var(--accent);
        color: #000;
        font-weight: 600;
        padding: 0.75rem 1.5rem;
        border: none;
        border-radius: 8px;
        cursor: pointer;
        font-size: 0.95rem;
        transition: all 0.2s ease;
    }

    button:hover {
        background-color: var(--accent-hover);
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(0, 217, 181, 0.2);
    }

    button.secondary {
        background-color: transparent;
        border: 1px solid var(--border-color);
        color: var(--text-primary);
    }

    button.secondary:hover {
        background-color: rgba(255, 255, 255, 0.05);
        border-color: var(--text-secondary);
    }

    button.danger {
        background-color: transparent;
        border: 1px solid var(--danger);
        color: var(--danger);
    }

    button.danger:hover {
        background-color: var(--danger);
        color: white;
    }

    /* table */
    .table-container {
        background: var(--card-bg);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        overflow: hidden;
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    th, td {
        padding: 1rem 1.25rem;
        text-align: left;
        border-bottom: 1px solid var(--border-color);
    }

    th {
        background-color: rgba(255, 255, 255, 0.03);
        color: var(--text-secondary);
        font-weight: 500;
        font-size: 0.85rem;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    tr:last-child td {
        border-bottom: none;
    }

    tr:hover {
        background-color: rgba(255, 255, 255, 0.02);
    }

    a {
        color: var(--accent);
        text-decoration: none;
    }

    a:hover {
        text-decoration: underline;
    }

    /* cards */
    .card {
        background: var(--card-bg);
        border: 1px solid var(--border-color);
        border-radius: 16px;
        padding: 1.5rem;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
    }

    /* table on mobile */
    @media (max-width: 480px) {
        nav {
            justify-content: center;
            flex-wrap: wrap;
        }
        nav a b {
            width: 100%;
            margin-right: 0;
            padding-bottom: 0.5rem;
        }
        nav a {
            padding: 0.75rem 0.5rem;
            font-size: 0.85rem;
        }
    }
</style>
`;

const currentPage = location.pathname.split('/').pop() || 'index.html';

document.body.innerHTML = `
<header>
    <nav>
        <a href="index.html"><img src="cctv-icon.png" alt="cctv-icon" style="width: 45px;"></a>
        <a href="index.html"><b>go2rtc</b></a>
        <a href="add.html" class="${currentPage === 'add.html' ? 'active' : ''}">Add</a>
        <a href="users.html" class="${currentPage === 'users.html' ? 'active' : ''}">Users</a>
        <a href="origins.html" class="${currentPage === 'origins.html' ? 'active' : ''}">Origins</a>
        <a href="tokens.html" class="${currentPage === 'tokens.html' ? 'active' : ''}">Tokens</a>
        <a href="types.html" class="${currentPage === 'types.html' ? 'active' : ''}">Types</a>
        <a href="#" class="btn-sm danger" id="logout-btn" style="margin-left: auto; color: var(--danger);">Logout</a>
    </nav>
</header>
` + document.body.innerHTML;

const originalFetch = window.fetch;
window.fetch = async function () {
    const response = await originalFetch.apply(this, arguments);
    if (response.status === 401 && !window.location.pathname.endsWith('login.html')) {
        window.location.href = 'login.html?redirect=' + encodeURIComponent(window.location.pathname + window.location.search);
    }
    return response;
};

document.addEventListener('click', async (ev) => {
    if (ev.target.id === 'logout-btn') {
        ev.preventDefault();
        try {
            await fetch('api/logout', { method: 'POST' });
        } catch (e) {
            console.error('Logout failed:', e);
        }
        window.location.href = 'login.html';
    }
});
