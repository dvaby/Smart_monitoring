# Asumsi dan hasil verifikasi Fase 0

Terakhir diperbarui: 2026-11-13

## Pilihan implementasi yang dikonfirmasi

- Akun admin pertama dibuat melalui formulir setup lokal saat aplikasi pertama kali dibuka; tidak ada password bawaan.
- Bot Token disimpan terenkripsi AES-256-GCM dan tidak pernah ditampilkan kembali setelah disimpan; kolomnya menggunakan placeholder "Isi hanya untuk mengganti token". Chat ID disimpan sebagai teks biasa dan ditampilkan utuh pada kartu ringkasan serta kolom isian yang terisi nilai tersimpan, sesuai PRD 3.5 AC-1. URL proxy tetap tersembunyi di UI karena dapat memuat kredensial.
- Pemilik proyek akan memasukkan kredensial Telegram sendiri pada UI lokal setelah aplikasi tersedia. Tidak ada rahasia Telegram yang diminta atau disimpan dalam repositori.
- Data contoh dari prototipe tidak digunakan. Belum ada koneksi atau perintah ke router MikroTik pada Fase 0.

## Hasil uji tray yang dilaporkan pemilik proyek

- Ubuntu 26.04, `wails dev -tags webkit2_41`, `getlantern/systray`: **GAGAL**, crash `SIGABRT` pada `github.com/getlantern/systray.nativeLoop` ketika Wails membuka window. Dugaan akar masalah dari pengujian: dua event loop GTK/AppIndicator dalam satu proses.
- Pustaka pengganti: `fyne.io/systray` v1.12.2. Dokumentasi upstream: <https://github.com/fyne-io/systray/tree/v1.12.2>. Dokumentasi menyatakan implementasi Linux memakai DBus SystemNotifier/AppIndicator (StatusNotifierItem), tanpa GTK, dan menyediakan `RunWithExternalLoop` untuk toolkit yang mengelola event loop.
- `fyne.io/systray` runtime di Linux dan Windows: **BELUM DIUJI** oleh pemilik proyek. Build/test otomatis tidak dihitung sebagai bukti tray berfungsi.

## TODO-VERIFY

- TODO-VERIFY: Tray pengganti `fyne.io/systray` v1.12.2 belum diuji saat runtime. Dokumentasi proyek menyebut Linux menggunakan DBus SystemNotifier/AppIndicator (StatusNotifierItem), dan fork ini menghapus dependensi GTK. Integrasi memakai `RunWithExternalLoop` untuk toolkit lain seperti Wails. Tray hanya dapat dinyatakan lulus setelah pemilik proyek menguji Ubuntu dan Windows.
- TODO-VERIFY: Pada desktop Ubuntu, jalankan aplikasi, pastikan ikon tray tampak, tutup jendela dengan tombol X, pastikan proses tetap hidup, lalu pilih "Buka Smart Network Monitoring" dan pastikan jendela muncul kembali. Pilih "Keluar" dari menu tray untuk memastikan proses berhenti.
- TODO-VERIFY: Pada Windows 10/11, ulangi skenario tray di atas, termasuk memastikan aplikasi tidak berhenti setelah jendela disembunyikan.
- TODO-VERIFY: Untuk memvalidasi setup admin, Wails IPC dan Telegram sebelum uji tray selesai, jalankan dengan `SNM_DISABLE_TRAY=1`; jendela X menutup aplikasi dalam mode ini.
- TODO-VERIFY: Pengiriman pesan Telegram nyata belum diuji. Setelah membuka aplikasi, buat/login admin, isi Bot Token dan Chat ID yang valid atau atur environment `SNM_TELEGRAM_BOT_TOKEN` dan `SNM_TELEGRAM_CHAT_ID`, lalu tekan "Kirim Pesan Uji". Validasi juga HTTP/SOCKS5 proxy pada jaringan yang memerlukannya. Hasil pengiriman baru boleh dicatat tanpa nilai kredensial.
- TODO-VERIFY: Uji otomatis membuktikan fallback file kunci Linux `0600` bekerja saat Secret Service tak tersedia; pemakaian Secret Service sebenarnya dan perilaku unlock setelah logout/login masih perlu dikonfirmasi pada desktop Ubuntu.
- TODO-VERIFY: Enkripsi kunci Windows DPAPI telah disediakan melalui implementasi OS-spesifik, tetapi perlu diuji pada runtime Windows dan dengan akun Windows yang berbeda untuk memastikan pembatasan scope kunci sesuai instalasi.
- TODO-VERIFY: Build lokal Linux pada Ubuntu 26.04 belum dijalankan karena dependency native belum terpasang. Workflow CI menargetkan Ubuntu 22.04 dan 24.04 dengan GTK3, WebKitGTK masing-masing, dan AppIndicator; hasil workflow CI perlu dilihat pada push/PR.
- TODO-VERIFY: Chat ID disimpan polos sesuai keputusan pemilik proyek, berbeda dari persyaratan eksplisit PRD yang meminta enkripsi Token dan Chat ID.
- TODO-VERIFY: Kata sandi admin memerlukan minimum 12 karakter (kebijakan fondasi awal), sedangkan PRD hanya menetapkan Argon2id dan penguncian setelah 5 kegagalan, tanpa panjang minimum.
- TODO-VERIFY: Skema Fase 0 mengikuti tabel awal PRD 6 dan memasukkan penyesuaian yang sudah disebut PRD untuk `dhcp_server` serta `is_randomized_mac`; tabel bandwidth Fase 3 sengaja belum dibuat.

## Hasil uji yang tersedia

| Pemeriksaan | Hasil |
| --- | --- |
| Unit test Argon2id, enkripsi AES-GCM, migrasi SQLite/WAL, penyimpanan Bot Token terenkripsi | Lulus (Linux, `CGO_ENABLED=0` untuk subset backend) |
| Frontend React + TypeScript + Tailwind build | Lulus |
| Semua test backend tanpa dependensi desktop native (`CGO_ENABLED=0`) | Lulus: Argon2id, AES-GCM/DPAPI compile, fallback kunci Linux, migrasi/schema SQLite/WAL, token terenkripsi, diagnosis koneksi Telegram |
| `go test ./...` | Lulus |
| `go vet ./...` | Lulus |
| depguard seluruh paket `./...` | Lulus |
| `GOOS=windows GOARCH=amd64 go build .` | Lulus; executable belum diuji runtime di Windows |
| Build aplikasi Linux | Belum dijalankan: dependency GTK/WebKitGTK Wails tidak tersedia lokal; AppIndicator tidak lagi dibutuhkan oleh tray |
| UI Wails IPC | Binding dan bridge tersedia; `SNM_DISABLE_TRAY=1` memungkinkan pengujian UI tanpa membuat tray |
| Tray runtime Windows/Linux | `getlantern/systray`: GAGAL di Ubuntu (SIGABRT). `fyne.io/systray`: belum diuji; jangan menyatakan lulus sebelum uji manual |
| Kirim pesan uji Telegram | Belum diuji; menunggu pemilik proyek mengisi rahasia pada UI |

## Pustaka tray dan rencana cadangan

Pengganti yang dipilih adalah [`fyne.io/systray`](https://github.com/fyne-io/systray), versi v1.12.2. README proyek menjelaskan bahwa implementasi Linux memakai DBus SystemNotifier/AppIndicator (StatusNotifierItem), tanpa GTK, dan menyediakan `RunWithExternalLoop` untuk integrasi dengan toolkit yang mengelola main loop sendiri. Ini menghindari loop GTK/AppIndicator tray lama, tetapi tidak membuktikan integrasi runtime bersama Wails berhasil.

Jika `fyne.io/systray` juga gagal atau tidak stabil saat diuji, jangan menambahkan workaround native lain. Usulan cadangan: pisahkan aplikasi menjadi dua proses—inti pemantauan sebagai background service dan jendela Wails sebagai klien—yang berkomunikasi lewat loopback dengan autentikasi dan perlindungan replay. Perubahan arsitektur ini perlu persetujuan pemilik proyek sebelum dikerjakan.
