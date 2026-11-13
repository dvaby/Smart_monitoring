import { FormEvent, useCallback, useEffect, useState } from 'react';
import './App.css';
import { appBridge, AppState } from './bridge';

function App() {
  const [state, setState] = useState<AppState | null>(null);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [botToken, setBotToken] = useState('');
  const [chatId, setChatId] = useState('');
  const [proxyUrl, setProxyUrl] = useState('');

  const refreshState = useCallback(async () => {
    try {
      setState(await appBridge().GetState());
      setError('');
    } catch (reason) {
      setError(messageOf(reason));
    }
  }, []);

  useEffect(() => { void refreshState(); }, [refreshState]);

  useEffect(() => {
    if (state?.authenticated && state.chatId) setChatId(state.chatId);
  }, [state?.authenticated, state?.chatId]);

  async function submitAccount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    setNotice('');
    try {
      if (state?.adminConfigured) {
        await appBridge().Login(username, password);
        setNotice('Berhasil masuk.');
      } else {
        await appBridge().CreateAdmin(username, password);
        setNotice('Akun admin lokal berhasil dibuat.');
      }
      setPassword('');
      await refreshState();
    } catch (reason) {
      setError(messageOf(reason));
    } finally {
      setBusy(false);
    }
  }

  async function saveTelegram(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError('');
    setNotice('');
    try {
      await appBridge().SaveTelegramSettings(botToken, chatId, proxyUrl);
      setBotToken('');
      setNotice('Pengaturan Telegram disimpan. Bot Token tersimpan terenkripsi.');
      await refreshState();
    } catch (reason) {
      setError(messageOf(reason));
    } finally {
      setBusy(false);
    }
  }

  async function testTelegram() {
    setBusy(true);
    setError('');
    setNotice('');
    try {
      setNotice(await appBridge().TestTelegram(botToken, chatId, proxyUrl));
    } catch (reason) {
      setError(messageOf(reason));
    } finally {
      setBusy(false);
    }
  }

  async function logout() {
    await appBridge().Logout();
    setState(current => current ? { ...current, authenticated: false } : current);
    setNotice('Anda telah keluar.');
  }

  if (!state && !error) return <main className="loading">Memulai Smart Network Monitoring...</main>;
  const backendReady = Boolean(window.go?.wailsbind?.Bindings);

  return (
    <div className="app-shell min-h-screen">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark" aria-hidden="true">⌁</div>
          <div><strong>SMART NETWORK</strong><span>MONITORING SYSTEM</span></div>
        </div>
        <div className="side-label">FONDASI APLIKASI</div>
        <div className="nav-item active"><span aria-hidden="true">▦</span> Pengaturan awal</div>
        <div className="sidebar-status"><span className="status-dot" /> Aplikasi aktif</div>
      </aside>
      <main className="main-content">
        <header className="topbar">
          <div><p className="eyebrow">SMART NETWORK MONITORING</p><h1>{state?.authenticated ? 'Pengaturan Sistem' : 'Selamat Datang'}</h1></div>
          {state?.authenticated && <button className="button button-quiet" onClick={logout}>Keluar</button>}
        </header>
        <section className="content">
          {!backendReady && <div className="alert alert-warning">UI belum terhubung ke proses Wails. Jalankan aplikasi desktop, bukan halaman Vite mandiri.</div>}
          {error && <div className="alert alert-error" role="alert">{error}</div>}
          {notice && <div className="alert alert-success" role="status">{notice}</div>}
          {!state?.authenticated ? (
            <section className="panel auth-panel">
              <div className="panel-heading">
                <div className="icon-tile">{state?.adminConfigured ? '↗' : '✦'}</div>
                <div><h2>{state?.adminConfigured ? 'Login Admin Lokal' : 'Buat Admin Pertama'}</h2>
                  <p>{state?.adminConfigured ? 'Masuk untuk mengatur notifikasi dan menguji koneksi Telegram.' : 'Akun ini hanya tersimpan di komputer pemantau dan dilindungi Argon2id.'}</p>
                </div>
              </div>
              <form className="form-grid" onSubmit={submitAccount}>
                <label>Username<input value={username} onChange={event => setUsername(event.target.value)} autoComplete="username" required /></label>
                <label>Password<input value={password} onChange={event => setPassword(event.target.value)} autoComplete={state?.adminConfigured ? 'current-password' : 'new-password'} type="password" minLength={state?.adminConfigured ? undefined : 12} required /></label>
                {!state?.adminConfigured && <p className="field-note">Gunakan sedikitnya 12 karakter. Tidak ada password bawaan.</p>}
                <button className="button button-primary" disabled={busy || !backendReady || !state}>{busy ? 'Memproses...' : state?.adminConfigured ? 'Masuk' : 'Buat Akun Admin'}</button>
              </form>
            </section>
          ) : (
            <>
              <div className="summary-grid">
                <article className="summary-card"><span className="summary-icon blue">◉</span><div><p>Proses aplikasi</p><strong>Berjalan</strong><small>Siap bekerja di latar belakang</small></div></article>
                <article className="summary-card"><span className="summary-icon green">✓</span><div><p>Fitur tray</p><strong>Disiapkan</strong><small>Uji runtime desktop masih diperlukan</small></div></article>
                <article className="summary-card"><span className="summary-icon rose">✉</span><div><p>Telegram</p><strong>{state.telegramConfigured ? 'Dikonfigurasi' : 'Belum diatur'}</strong><small>{state.telegramConfigured ? `Chat ID: ${state.chatId}` : 'Isi kredensial secara lokal'}</small></div></article>
              </div>
              <section className="panel">
                <div className="panel-heading"><div className="icon-tile telegram-icon">✉</div><div><h2>Uji Koneksi Telegram</h2><p>Token disimpan terenkripsi di SQLite. Token yang tersimpan tidak pernah ditampilkan kembali.</p></div></div>
                <form className="form-grid two-columns" onSubmit={saveTelegram}>
                  <label>Bot Token<input value={botToken} onChange={event => setBotToken(event.target.value)} type="password" autoComplete="new-password" placeholder={state.telegramConfigured ? 'Isi hanya untuk mengganti token' : 'Token Bot Telegram'} /></label>
                  <label>Chat ID<input value={chatId} onChange={event => setChatId(event.target.value)} placeholder="Chat ID tujuan" /></label>
                  <label className="full-width">Proxy keluar (opsional)<input value={proxyUrl} onChange={event => setProxyUrl(event.target.value)} type="password" autoComplete="off" placeholder="http://host:port atau socks5://host:port" /></label>
                  <div className="form-actions full-width"><button className="button button-primary" disabled={busy || !backendReady}>Simpan Pengaturan</button><button className="button button-secondary" type="button" onClick={testTelegram} disabled={busy || !backendReady}>{busy ? 'Menguji...' : 'Kirim Pesan Uji'}</button></div>
                </form>
                <p className="privacy-note">Atau set SNM_TELEGRAM_BOT_TOKEN, SNM_TELEGRAM_CHAT_ID, dan SNM_TELEGRAM_PROXY_URL secara lokal untuk pengujian. Rahasia tidak perlu dikirim lewat chat.</p>
              </section>
              <section className="tray-note"><span aria-hidden="true">ⓘ</span><p>Jendela dapat disembunyikan dan dibuka kembali melalui ikon tray. Pilih <b>Keluar</b> dari menu tray untuk menghentikan proses.</p><button className="text-button" onClick={() => appBridge().HideWindow()}>Sembunyikan jendela</button></section>
            </>
          )}
        </section>
        <footer>Smart Network Monitoring <span>•</span> Fase 0 — Fondasi</footer>
      </main>
    </div>
  );
}

function messageOf(value: unknown): string {
  if (value instanceof Error) return value.message;
  return String(value);
}

export default App;
