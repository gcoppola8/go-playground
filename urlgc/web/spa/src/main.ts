import axios from 'axios'

const el = document.getElementById('app')!

el.innerHTML = `
  <style>
    :root{--bg:#0b1020;--card:#11162a;--muted:#cbd5e1;--fg:#e2e8f0;--accent:#8b5cf6;--accent2:#22c55e}
    body{margin:0;background:var(--bg);color:var(--fg);font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Ubuntu, Cantarell, Noto Sans, Helvetica, Arial, "Apple Color Emoji","Segoe UI Emoji"}
    .container{max-width:720px;margin:40px auto;padding:0 20px}
    .card{background:var(--card);border-radius:16px;padding:24px;border:1px solid #1f2847;box-shadow:0 10px 30px rgba(0,0,0,.2)}
    h1{margin:0 0 8px 0;font-weight:700;letter-spacing:.3px}
    p{margin:0 0 24px 0;color:var(--muted)}
    form{display:flex;gap:12px}
    input[type=url]{flex:1;border-radius:12px;padding:12px 14px;border:1px solid #2a3358;background:#0e1430;color:var(--fg);font-size:16px;outline:none}
    input[type=url]:focus{border-color:#4153a5;box-shadow:0 0 0 3px rgba(99,102,241,.25)}
    button{border:none;border-radius:12px;background:linear-gradient(135deg,#6366f1,#22c55e);color:white;font-weight:600;padding:12px 16px;cursor:pointer}
    .result{margin-top:18px;display:flex;gap:8px;align-items:center}
    .result a{color:#93c5fd}
    .copy{background:#0e1430;border:1px solid #2a3358;color:#cbd5e1}
    .small{font-size:12px;color:#94a3b8;margin-top:8px}
  </style>
  <div class="container">
    <div class="card">
      <h1>URL Shortener</h1>
      <p>Create a short link that redirects to your target URL.</p>
      <form id="f">
        <input id="url" type="url" placeholder="https://example.com/page" required />
        <button type="submit">Shorten</button>
      </form>
      <div id="res" class="result" style="display:none"></div>
      <div class="small">API: POST /api/urls with JSON { target: string }</div>
    </div>
  </div>
`

const form = document.getElementById('f') as HTMLFormElement
const urlInput = document.getElementById('url') as HTMLInputElement
const resDiv = document.getElementById('res') as HTMLDivElement

form.addEventListener('submit', async (e) => {
  e.preventDefault()
  const target = urlInput.value.trim()
  if (!target) return
  try {
    const { data } = await axios.post('/api/urls', { target })
    const shortUrl = data.shortUrl as string
    resDiv.style.display = 'flex'
    resDiv.innerHTML = `
      <span>Short URL:</span>
      <a href="${shortUrl}" target="_blank" rel="noreferrer">${shortUrl}</a>
      <button id="copy" class="copy">Copy</button>
    `
    const copyBtn = document.getElementById('copy') as HTMLButtonElement
    copyBtn.onclick = async () => {
      try { await navigator.clipboard.writeText(shortUrl); copyBtn.textContent = 'Copied!'; setTimeout(()=>copyBtn.textContent='Copy',1200) } catch {}
    }
  } catch (err: any) {
    alert('Failed to create short URL: ' + (err?.response?.data || err?.message || 'unknown error'))
  }
})
