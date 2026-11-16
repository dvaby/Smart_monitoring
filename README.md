# Smart Network Monitoring System

Fondasi desktop Fase 0: Go + Wails v2, React/TypeScript/Tailwind, SQLite WAL, admin lokal, enkripsi kredensial, tray, dan uji Telegram. Aplikasi belum terhubung ke router.

## Prasyarat

- Go sesuai versi di `go.mod`, Node.js 24+ dan npm.
- Wails CLI v2.
- Linux: GTK3 and WebKitGTK for Wails. Ubuntu 22.04 uses WebKitGTK 4.0; Ubuntu 24.04 uses 4.1 (`libgtk-3-dev libwebkit2gtk-4.x-dev pkg-config`). The tray uses DBus StatusNotifierItem and does not require GTK/AppIndicator.
- Windows 10/11: WebView2 Runtime.

## Jalankan

```sh
cd frontend && npm ci && npm run build
cd ..
wails dev
```

For desktop/IPC testing without a tray icon, set `SNM_DISABLE_TRAY=1` before launching. In this mode, closing the window exits the app normally; the tray menu is unavailable.

Ubuntu:

```sh
SNM_DISABLE_TRAY=1 wails dev -tags webkit2_41
```

PowerShell:

```powershell
$env:SNM_DISABLE_TRAY = "1"
wails dev
```

Saat pertama dibuka, buat admin lokal dengan password minimal 12 karakter. Login admin untuk mengisi Bot Token dan Chat ID, simpan pengaturan, lalu tekan **Kirim Pesan Uji**. Bot Token dan URL proxy disimpan terenkripsi dan tidak ditampilkan kembali; Chat ID disimpan sebagai teks biasa, ditampilkan utuh di kartu ringkasan, dan dimuat kembali pada kolom isian. Untuk mengganti Bot Token, isi kolom dengan placeholder **Isi hanya untuk mengganti token**. Alternatif uji lokal menggunakan environment `SNM_TELEGRAM_BOT_TOKEN`, `SNM_TELEGRAM_CHAT_ID`, dan `SNM_TELEGRAM_PROXY_URL`.

Menutup jendela menyembunyikan aplikasi. Gunakan ikon tray untuk membuka jendela lagi atau memilih **Keluar**.

## Build dan uji

```sh
make test
make architecture
make build-linux
make build-windows
```

Pada Ubuntu 24.04, gunakan `make build-linux TAGS=webkit2_41`; Ubuntu 22.04 memakai default WebKitGTK 4.0. Build Windows harus dijalankan pada Windows. CI membangun untuk Windows serta Ubuntu 22.04/24.04. Tray memakai `fyne.io/systray` melalui DBus StatusNotifierItem di Linux, tanpa dependency GTK/AppIndicator tersendiri. Lihat [docs/ASSUMPTIONS.md](docs/ASSUMPTIONS.md) untuk uji tray, status uji Telegram, dan hal yang masih harus diverifikasi.

davin and tegar
