# Product Requirements Document: Smart Network Monitoring System

**Versi 1.0 · 12 November 2026 · Target router: MikroTik RouterOS 6 · Status: Draft untuk tim pengembang**

> Catatan: prototipe PDF menyebut Laravel pada subjudulnya. PRD ini mengikuti arahan terbaru: aplikasi desktop tunggal berbasis Go dan Wails v2, tanpa server PHP terpisah.

## 0. Petunjuk untuk Pelaksana (Tim Developer / AI Agent)

Bagian ini wajib dibaca sebelum menulis kode.

- Baca seluruh dokumen dulu, terutama **5.7 (Clean Architecture)** dan **5.8 (hal berisiko)**.
- Kerjakan **per fase secara berurutan**, mulai Fase 0. Jangan membangun fitur fase berikutnya sebelum kriteria selesai fase berjalan terpenuhi dan dibuktikan dengan uji.
- Aturan yang tidak boleh dilanggar: (1) aplikasi **read-only** terhadap router, (2) kredensial dan token **terenkripsi**, (3) arah dependensi **Clean Architecture** (5.7).
- Bila ada hal yang tidak pasti (format log MikroTik, versi RouterOS, perilaku tray), **jangan menebak**: buat berkas uji atau konfigurasi yang bisa diubah, beri tanda `TODO-VERIFY`, dan catat di `docs/ASSUMPTIONS.md`.
- Data pada prototipe (mis. IP 192.168.10.25) hanya data simulasi untuk uji, bukan data produksi.
- Keluaran setiap fase: kode, uji otomatis, README singkat cara menjalankan, dan daftar hal yang masih belum diverifikasi.

## 1. Ringkasan Eksekutif & Tujuan Sistem

Instansi kedinasan membutuhkan pengawasan jaringan lokal (*on-premise*) yang terhubung ke router utama MikroTik RouterOS 6. Gangguan seperti konflik IP akibat perangkat liar atau kesalahan IP statis, dan perangkat vital yang mati (Access Point, Switch, Printer), saat ini baru diketahui setelah pengguna di unit pelayanan mengeluh.

**Smart Network Monitoring System** adalah aplikasi desktop (`.exe` Windows dan binary Linux) yang berjalan 24/7 di latar belakang. Aplikasi menerima log dari router, memperkaya data lewat RouterOS API, menyimpan riwayat di SQLite, dan mengirim peringatan ke smartphone tim IT lewat Telegram. Teknisi di lapangan memantau lewat antarmuka web responsif (PWA) dari browser HP.

### 1.1 Tujuan

1. Mendeteksi **IP Conflict** secara proaktif dari log `dhcp,warning` dan pencocokan silang data lease dengan ARP.
2. Mendeteksi **Device Down / Up** dari log `netwatch` beserta durasi *downtime*.
3. Mengirim notifikasi otomatis (Telegram (WhatsApp Gateway menyusul pasca-rilis)) dalam kurang dari 30 detik sejak kejadian diterima.
4. Menyediakan data investigasi (MAC lease vs MAC ARP, interface, waktu) dan panduan tindakan 5 langkah.
5. Menyimpan riwayat insiden sebagai bahan evaluasi dan pelaporan.

### 1.2 Bukan Tujuan (Non-Goals)

- Tidak mengubah konfigurasi router, memblokir MAC, menghapus lease, atau memperbaiki IP secara otomatis. Sistem **hanya membaca dan memperingatkan**.
- Tidak menggantikan sistem NMS skala besar (SNMP penuh, analisis NetFlow per aplikasi atau tujuan, dll.). Bandwidth hanya dipantau sebatas total pemakaian per IP (3.7).
- Tidak mendukung RouterOS 7 pada rilis awal (REST API RouterOS 7 masuk roadmap).
- Tidak diakses dari internet publik. Akses PWA hanya lewat jaringan lokal (LAN atau Wi-Fi internal), tanpa VPN pada v1.0.

### 1.3 Indikator Keberhasilan

| Metrik | Target |
| --- | --- |
| Waktu dari log diterima sampai notifikasi terkirim | ≤ 30 detik (P95) |
| Insiden IP Conflict terdeteksi dibanding yang terbukti di lapangan | ≥ 90% |
| Notifikasi duplikat dalam satu jendela cooldown | 0 |
| Ketersediaan proses latar belakang (sebulan) | ≥ 99,5% |
| Waktu rata-rata teknisi mulai investigasi (MTTA) | Turun ≥ 50% dari kondisi awal |

## 2. User Persona & Hak Akses

| Persona | Deskripsi | Kebutuhan Utama |
| --- | --- | --- |
| **Admin Jaringan** (Pak Syahrul, 27) | Penanggung jawab router, server, dan kebijakan IP. Bekerja di ruang server dengan PC Windows. | Konfigurasi koneksi router, kanal notifikasi, inventaris perangkat, penutupan insiden. |
| **Teknisi Lapangan** (Pak Syahrul, 27) | Menangani gangguan di unit pelayanan. Memakai HP Android. | Melihat peringatan dan detail insiden cepat, mengetahui lokasi dan MAC perangkat, menandai penanganan. |

### 2.1 Matriks Hak Akses

| Fitur | Admin Jaringan | Teknisi Lapangan |
| --- | --- | --- |
| Lihat dashboard, tabel DHCP/ARP, status perangkat | ✔ | ✔ |
| Lihat detail dan riwayat insiden | ✔ | ✔ |
| Mengakui (*acknowledge*) insiden | ✔ | ✔ |
| Mengonfirmasi insiden selesai | ✔ | ✔ (dengan catatan wajib) |
| Tambah/ubah/hapus perangkat terpantau | ✔ | ✘ |
| Ubah kredensial router, Bot Token, interval, cooldown | ✔ | ✘ |
| Kelola pengguna dan token akses PWA | ✔ | ✘ |
| Ekspor riwayat insiden | ✔ | ✘ |

Autentikasi: aplikasi desktop memakai login lokal admin. Akun admin pertama dibuat lewat form setup saat aplikasi pertama kali dijalankan (bagian dari wizard pengaturan awal), dan tidak ada kredensial bawaan. Kata sandi admin minimal 12 karakter. Akses PWA memakai akun teknisi (username + password, hash Argon2id) dengan sesi berbatas waktu.

## 3. Spesifikasi Kebutuhan Fungsional

### 3.1 Dashboard Ringkasan (Network Overview)

**Fungsi:** halaman utama yang menampilkan kondisi jaringan sekilas.

- Kartu metrik: router dipantau dan status API, total perangkat (online/offline), konflik IP aktif, persentase *uptime*.
- Grafik tren kejadian 7 hari terakhir (Senin–Minggu).
- Daftar *Recent Alerts* yang diperbarui real-time (maksimal 10 item).

**User Story:** Sebagai Admin Jaringan, saya ingin melihat ringkasan kondisi jaringan saat membuka aplikasi agar tahu apakah ada gangguan yang perlu ditangani.

**Acceptance Criteria**

- AC-1: Kartu metrik menampilkan nilai yang konsisten dengan data SQLite dan diperbarui otomatis tanpa reload (maks. 5 detik setelah perubahan data).
- AC-2: Jika koneksi RouterOS API gagal, kartu status router berubah menjadi *Terputus* (merah) dan banner peringatan muncul.
- AC-3: Grafik 7 hari menghitung jumlah insiden (konflik + device down) per hari, berdasarkan zona waktu lokal perangkat.
- AC-4: Klik item *Recent Alerts* membuka halaman detail insiden terkait.
- AC-5: Uptime = (total menit perangkat vital UP ÷ total menit pemantauan) × 100%, dihitung dalam jendela 30 hari.

### 3.2 Monitoring DHCP & ARP

**Fungsi:** tabel pemetaan langsung IP–MAC–hostname dari lease DHCP dan tabel ARP.

- Kolom: Alamat IP, Hostname, MAC Address, Sumber (`DHCP` / `Static`), Status (`Aktif` / `Indikasi`), Terakhir Terpantau.
- Pencarian cepat berdasarkan IP, MAC, atau hostname. Tab filter `SEMUA`, `AKTIF`, `KONFLIK`. Tombol *Refresh*.
- Catatan di bawah tabel: alamat yang sama dengan MAC berbeda perlu diverifikasi lewat log, lease, dan ARP.

**User Story:** Sebagai Teknisi Lapangan, saya ingin mencari IP atau MAC tertentu agar cepat tahu perangkat apa yang memakai alamat tersebut.

**Acceptance Criteria**

- AC-1: Data ditarik dari `/ip/arp` dan `/ip/dhcp-server/lease` secara berkala (default 60 detik) dan digabung berdasarkan IP.
- AC-2: Status `Indikasi` diberikan bila (a) MAC ARP berbeda dari MAC lease untuk IP yang sama, atau (b) ada log `dhcp,warning` untuk IP tersebut yang memenuhi syarat kandidat konflik (3.3 AC-1) dalam 24 jam terakhir.
- AC-3: Pencarian bersifat *case-insensitive*, mendukung sebagian string, dan hasil muncul dalam < 300 ms untuk 2.000 baris.
- AC-4: Tab `KONFLIK` hanya menampilkan baris berstatus `Indikasi`.
- AC-5: Sumber `Static` ditentukan dari entri ARP/lease yang bertanda statis atau tidak ada lease dinamis.
- AC-6: Baris yang tidak terlihat lagi selama > 15 menit ditandai redup dan tidak dihitung sebagai online.

### 3.3 Pusat Peringatan IP Conflict & Investigasi Insiden

**Fungsi:** daftar dan detail insiden konflik IP beserta alur penanganan manual.

- Detail insiden: alamat IP, sumber kejadian (`DHCP warning / Syslog`), status lease, **MAC dari DHCP Lease**, **MAC ARP teramati**, interface (mis. `ether2-LAN`), nama router, dan waktu.
- Status insiden: `Baru` → `Diakui` → `Dalam Penanganan` → `Selesai`.
- Panduan 5 langkah: (1) Tinjau log DHCP dan ARP, (2) Cocokkan MAC dan perangkat, (3) Periksa IP statis yang tumpang tindih, (4) Tangani penyebab konflik, (5) Konfirmasi jaringan kembali normal.
- Disclaimer permanen: aplikasi hanya memberi peringatan dan informasi. Perubahan konfigurasi dan perbaikan IP dilakukan petugas berwenang.

**User Story:** Sebagai Teknisi Lapangan, saya ingin melihat MAC mana yang sah dan mana yang liar untuk sebuah IP agar tahu perangkat apa yang harus dicabut atau diperbaiki.

**Acceptance Criteria**

- AC-1: Log `dhcp,warning` hanya menjadi **kandidat** konflik, bukan bukti. Pada contoh log nyata (5.8 butir 8), pesannya berbentuk `dhcp2 offering lease <IP> for <MAC> without success`. Kandidat menjadi insiden bila (a) tidak disusul log `assigned` untuk IP dan MAC yang sama dalam 60 detik, atau (b) berulang minimal 3 kali dalam 10 menit untuk IP yang sama, atau (c) MAC pada ARP berbeda dari MAC lease pada saat yang sama. Ambang waktu dan jumlah dapat diatur.
- AC-2: Satu IP hanya memiliki satu insiden terbuka. Kejadian berulang menambah penghitung dan `last_seen`, bukan membuat insiden baru.
- AC-3: Detail menampilkan kedua MAC berdampingan, dengan perbedaan disorot. Lookup vendor OUI ditampilkan bila tersedia.
- AC-4: Tombol *Konfirmasi Jaringan Normal* hanya aktif setelah langkah 1–4 dicentang. Sistem melakukan pemeriksaan ulang sebelum menutup. Jika masih berbeda, muncul peringatan dan pengguna boleh melanjutkan dengan catatan.
- AC-5: Setiap perubahan status dicatat di tabel audit (siapa, kapan, catatan).
- AC-6: Insiden otomatis ditutup dengan label *auto-resolved* bila MAC ARP kembali sama dengan MAC lease selama 3 siklus validasi berturut-turut. Notifikasi `DEVICE_RESOLVED`-style dikirim bila diaktifkan.
- AC-7: Tidak ada satu pun tombol atau API internal yang menulis ke router.

### 3.4 Monitoring Perangkat Mati / Hidup (Device Up/Down)

**Fungsi:** pemantauan perangkat vital berdasarkan log `netwatch` dari router.

- Inventaris perangkat vital: nama, IP, tipe (AP, Switch, Server, Printer), lokasi/ruangan, tingkat prioritas.
- Deteksi `DOWN` dan `UP` otomatis, pencatatan durasi *downtime*.
- Daftar status perangkat dengan indikator warna dan durasi sejak perubahan terakhir.

**User Story:** Sebagai Admin Jaringan, saya ingin diberi tahu begitu Access Point ruang pelayanan mati agar bisa bertindak sebelum layanan terganggu.

**Acceptance Criteria**

- AC-1: Log netwatch `up`/`down` dipetakan ke perangkat berdasarkan IP host. Log dari IP yang belum terdaftar dimasukkan ke daftar *Perangkat Tidak Dikenal* untuk ditinjau admin.
- AC-2: Saat `DOWN`, insiden dibuka dan notifikasi `DEVICE_DOWN` dikirim. Saat `UP`, insiden ditutup, durasi dihitung, dan `DEVICE_RESOLVED` dikirim.
- AC-3: Tersedia fallback *heartbeat* ICMP dari aplikasi (opsional, dapat dimatikan) bila tidak ada log netwatch selama 3× interval.
- AC-4: Flapping (≥ 4 perubahan status dalam 10 menit) digabung menjadi satu notifikasi ringkas bertanda *flapping*.
- AC-5: Admin dapat mengatur prioritas sehingga perangkat prioritas tinggi melewati jam tenang (*quiet hours*).

### 3.5 Pengaturan Notifikasi & Alerting Otomatis

- **Telegram Bot API** (wajib, satu-satunya kanal eksternal pada v1.0) untuk IP Conflict dan Device Down. Webhook WhatsApp Gateway ditunda ke pasca-rilis.
- Toggle: notifikasi dashboard, Telegram Bot, ringkasan email.
- Interval validasi status DHCP (default 60 detik, rentang 15–600) dan *cooldown* anti-spam (default 10 menit, rentang 1–120).
- Tombol *Kirim Pesan Uji* dan *Uji Koneksi Router*.

**User Story:** Sebagai Admin Jaringan, saya ingin mengatur kanal dan frekuensi notifikasi agar tim IT mendapat peringatan penting tanpa dibanjiri pesan.

**Acceptance Criteria**

- AC-1: Bot Token disimpan terenkripsi (AES-256-GCM) dan tidak pernah ditampilkan kembali setelah disimpan; kolomnya menampilkan placeholder "Isi hanya untuk mengganti token". Chat ID disimpan sebagai teks biasa sesuai skema bagian 6 dan ditampilkan utuh pada kartu ringkasan serta kolom isian yang terisi nilai tersimpan. URL proxy tidak ditampilkan kembali karena dapat memuat kredensial.
- AC-2: Cooldown dihitung per kombinasi `(jenis_kejadian, IP/perangkat)`. Kejadian yang ditahan tetap tercatat di riwayat dengan flag `suppressed`.
- AC-3: Kegagalan kirim (HTTP ≠ 200 atau timeout) diulang dengan *exponential backoff* (5 dtk, 30 dtk, 2 mnt, maks. 5 kali). Kegagalan permanen muncul sebagai peringatan di dashboard.
- AC-4: Perubahan pengaturan berlaku tanpa restart aplikasi.
- AC-5: Validasi input: interval dan cooldown di luar rentang ditolak dengan pesan jelas.
- AC-6: Ringkasan email dikirim harian pukul yang dikonfigurasi (SMTP internal), berisi jumlah insiden, insiden terbuka, dan perangkat dengan downtime terlama.

### 3.6 Tampilan Mobile (PWA)

- **Beranda:** kartu Perangkat Aktif (mis. “121 Normal”), Konflik IP, Status Router, dan daftar Peringatan Terbaru.
- **Detail Peringatan:** jenis, IP, status, sumber, router, interface, waktu, dan tombol *Lihat Detail Log*.
- Dapat dipasang ke layar utama (*Add to Home Screen*), *mobile-first*, mendukung mode gelap.

**User Story:** Sebagai Teknisi Lapangan, saya ingin membuka tautan dari pesan Telegram di HP dan langsung melihat detail insiden.

**Acceptance Criteria**

- AC-1: Tautan di pesan Telegram membuka halaman detail insiden (deep link) setelah login, tanpa kehilangan tujuan.
- AC-2: Halaman beranda termuat < 2 detik di jaringan Wi-Fi internal pada HP kelas menengah.
- AC-3: Data diperbarui lewat polling 10 detik atau SSE; indikator *Terhubung / Terputus* selalu terlihat.
- AC-4: Tata letak benar pada lebar layar 320–480 px.
- AC-5: Aksi *Akui* dan *Selesai* tersedia untuk teknisi. Pengaturan sistem tidak muncul.
- AC-6: Service worker menyimpan cache aset statis agar aplikasi tetap terbuka saat jaringan putus sesaat, dengan label data *basi* bila tidak terhubung.

### 3.7 Pengecekan Bandwidth per IP (permintaan tambahan admin)

**Status:** fitur tambahan, dikerjakan di Fase 3 setelah fitur inti stabil, dan hanya bila gerbang kelayakan sumber data di 5.9 lolos. Fitur inti (IP Conflict dan Device Down) tidak bergantung pada modul ini.

**Fungsi:**

- Halaman **Bandwidth** berisi tabel per IP: IP, hostname, MAC, total unduh, total unggah, laju saat ini (Mbps), laju puncak, dan persentase dari total.
- Filter periode (1 jam, 24 jam, 7 hari, 30 hari), pencarian IP atau hostname, dan urutan dari pemakaian terbesar.
- Kartu **Top 5 pengguna bandwidth** di dashboard, serta grafik garis pemakaian per IP pada halaman detail.
- Ambang batas opsional (nonaktif secara bawaan) yang memicu notifikasi `BANDWIDTH_HIGH`.
- Hanya volume yang dicatat. Tidak ada alamat tujuan, nama situs, atau isi trafik.

**User Story:** Sebagai Admin Jaringan, saya ingin melihat IP mana yang memakai bandwidth terbesar agar bisa menelusuri penyebab jaringan lambat.

**Acceptance Criteria**

- AC-1: Data dihitung per IP pada subnet lokal yang dikonfigurasi (mis. `10.10.2.0/24`). Unggah = IP sebagai sumber, unduh = IP sebagai tujuan, hanya untuk trafik yang melewati router.
- AC-2: Data diagregasi dalam bucket 5 menit dan tampilan diperbarui paling lambat 1 menit setelah bucket ditutup.
- AC-3: Total per IP sama dengan jumlah bucket-nya, dan selisihnya dengan pembanding independen pada uji kelayakan tidak lebih dari 5%.
- AC-4: Retensi: bucket 5 menit disimpan 7 hari, ringkasan per jam 90 hari, ringkasan per hari 2 tahun. Pengecilan data (downsampling) berjalan otomatis.
- AC-5: Hostname dan MAC dipetakan dari tabel `hosts`. IP yang tidak dikenal tetap muncul dengan label "tidak dikenal".
- AC-6: Tabel top-N untuk 254 IP tampil dalam waktu kurang dari 500 ms.
- AC-7: Bila tidak ada data aliran selama lebih dari 10 menit, halaman Diagnostik menampilkan "Sumber bandwidth tidak aktif" lengkap dengan saran pemeriksaan (konfigurasi router, firewall UDP 2055).
- AC-8: Ambang batas: bila laju rata-rata sebuah IP melebihi X Mbps selama N bucket berturut-turut, insiden `BANDWIDTH_HIGH` dibuat dan cooldown berlaku seperti notifikasi lain. Admin mengatur X dan N (nilai bawaan N = 3).
- AC-9: Admin dapat mengubah pengaturan dan ambang batas. Teknisi hanya melihat tabel dan grafik (di PWA cukup Top 10). Endpoint: `/api/v1/bandwidth/top?period=24h` dan `/api/v1/bandwidth/ip/{ip}`.
- AC-10: Aplikasi tetap read-only. Tidak ada queue, rule, atau konfigurasi lain yang dibuat di router oleh aplikasi.

## 4. Spesifikasi Kebutuhan Non-Fungsional

### 4.1 Performa Polling dan Pemrosesan

| Parameter | Target |
| --- | --- |
| Interval polling `/ip/arp` dan `/ip/dhcp-server/lease` | Default 60 dtk, dapat 15–600 dtk |
| Skala didukung | Hingga 1 router, 2.000 entri lease/ARP, 500 perangkat terpantau |
| Penerimaan dan pemrosesan pesan syslog | ≥ 500 pesan/detik, tanpa kehilangan saat lonjakan (antrean *buffered channel* 10.000) |
| Latensi syslog → insiden tersimpan | ≤ 2 detik |
| Latensi insiden → pesan Telegram dikirim | ≤ 30 detik (P95) |
| Pemakaian sumber daya | RAM ≤ 150 MB, CPU rata-rata < 5% pada PC 4 core |
| Beban terhadap router | Satu koneksi API persisten, query terbatas pada `print` dengan `?` filter bila mungkin |

### 4.2 Keandalan Proses Latar Belakang

- Berjalan sebagai **System Tray** (Windows) dan dapat dipasang sebagai *service* (Windows Service / `systemd` di Linux) untuk auto-start saat boot.
- Menutup jendela hanya menyembunyikan UI. Keluar penuh lewat menu tray dengan konfirmasi.
- *Watchdog* internal: bila goroutine syslog, poller, atau notifier berhenti, otomatis dimulai ulang dan dicatat.
- Reconnect otomatis ke RouterOS API dengan *exponential backoff* (maks. 60 dtk). Status router dilaporkan *Terputus* setelah 3 kegagalan.
- Proteksi *single instance* agar tidak ada dua proses yang berebut port 514.
- SQLite memakai mode WAL dan transaksi atomik. Pencadangan otomatis harian (7 salinan terakhir).
- Retensi data: log mentah 90 hari (dhcp,info hanya 14 hari, lihat 5.8 butir 8), riwayat insiden 2 tahun (dapat dikonfigurasi). Pembersihan terjadwal setiap dini hari.
- Port 514 membutuhkan hak istimewa di Linux dan diblokir Windows Firewall secara bawaan di Windows, sehingga aturan firewall wajib dibuat (lihat 5.8). Sediakan opsi port alternatif (mis. 5514) atau `setcap cap_net_bind_service`.

### 4.3 Keamanan

| Area | Persyaratan |
| --- | --- |
| Kredensial router dan Bot Token | Dienkripsi **AES-256-GCM** di SQLite. Kunci diturunkan dari kunci mesin yang disimpan di **Windows DPAPI / Linux Secret Service** (fallback: berkas kunci dengan izin 0600). Tidak pernah masuk log atau UI utuh. |
| Akun RouterOS | User khusus **read-only** dengan policy `read, api` saja, dibatasi `address=` ke IP PC monitoring. Aplikasi menolak menyimpan kredensial bila user memiliki policy `write`, `policy`, atau `full`, dan menampilkan peringatan. |
| Transport API | Gunakan **API-SSL (8729)** bila tersedia, dengan opsi verifikasi sertifikat. API biasa (8728) hanya di segmen manajemen. |
| Syslog | Hanya menerima paket dari IP router yang terdaftar (*allowlist*). Paket lain dibuang dan dihitung. |
| Server HTTP lokal | Bind ke antarmuka LAN yang dipilih, bukan 0.0.0.0 secara default. HTTPS dengan sertifikat internal diwajibkan untuk PWA (lihat 5.8). Rate limit login, token sesi acak, cookie `HttpOnly` dan `SameSite=Strict`, header CSP. |
| Password pengguna | Argon2id. Penguncian sementara setelah 5 kali gagal. |
| Prinsip read-only | Klien API hanya mengizinkan perintah `/print` dan `/getall`. Perintah lain diblokir di lapisan klien (*allowlist* perintah) dan diuji otomatis. |
| Audit | Login, perubahan pengaturan, dan perubahan status insiden dicatat. |

### 4.4 Kebutuhan Lain

- **Kompatibilitas:** Windows 10/11 64-bit, Ubuntu 22.04+, RouterOS 6.49.7 (acuan uji, minimal 6.43 ke atas). Browser PWA: Chrome/Edge/Firefox/Safari terbaru.
- **Lokalisasi:** Bahasa Indonesia (utama), zona waktu WIB (dapat diubah).
- **Kemudahan pakai:** *Wizard* pengaturan awal (router → uji koneksi → kanal notifikasi → perangkat vital) kurang dari 10 menit.
- **Observabilitas:** log aplikasi terstruktur dengan rotasi dan halaman *Diagnostik* (status syslog, poller, antrean notifikasi).
- **Distribusi:** satu berkas eksekutif dengan basis data dan konfigurasi di folder data pengguna. Installer bertanda tangan digital bila memungkinkan.

## 5. Arsitektur Sistem & Alur Data

### 5.1 Komponen

| Komponen | Teknologi | Tanggung Jawab |
| --- | --- | --- |
| Syslog Receiver | Go `net.ListenUDP` port 514 | Menerima dan mengurai pesan RFC 3164/5424, menyimpan log dhcp (info dan warning) serta script (Netwatch), bukan hanya topik `dhcp,warning` dan `netwatch`. |
| RouterOS Poller | `go-routeros` (8728/8729) | Menarik `/ip/arp`, `/ip/dhcp-server/lease`, `/interface`, `/system/resource` secara berkala. |
| Event Processor | Go (goroutine + channel) | Normalisasi, korelasi log dengan ARP/lease, deduplikasi, pembuatan dan pembaruan insiden. |
| Notifier | Go (HTTP client) | Telegram dan email (webhook WhatsApp menyusul pasca-rilis). Cooldown, antrean, *retry*. |
| Storage | SQLite (WAL) | Konfigurasi, kredensial terenkripsi, inventaris, riwayat. |
| Wails Bridge | Wails v2 IPC | Frontend React memanggil fungsi Go (binding) dan menerima *event* real-time. |
| Local HTTP Server | Go `net/http` | REST API + SSE dan aset PWA untuk teknisi. |
| Frontend | React + TypeScript + Tailwind | UI desktop (7 halaman prototipe) dan PWA (kode dibagi bersama). |
| System Tray | Wails tray / library tray | Menjalankan aplikasi di latar belakang. |

### 5.2 Diagram Alur Data

```
 MikroTik RouterOS 6
  ├─ /system logging action=remote (dhcp,warning ; netwatch) ──UDP 514──┐
  └─ RouterOS API (8728 / 8729-SSL, user read-only) ◄──poll 60 dtk──┐   │
                                                                    │   ▼
                                    ┌───────────────────────────────┴───────────┐
                                    │ Aplikasi Go (satu proses)                 │
                                    │  Syslog Receiver ─┐                       │
                                    │  RouterOS Poller ─┼─► Event Processor     │
                                    │                   │     │  ▲              │
                                    │                   │     ▼  │              │
                                    │                   │   SQLite (WAL)        │
                                    │                   │     │                 │
                                    │            Notifier ◄───┤                 │
                                    │                         ├─► Wails IPC ──► UI Desktop (React)
                                    │                         └─► HTTP Server ──► PWA Teknisi (HP)
                                    └───────────────┬───────────────────────────┘
                                                    ▼
                                     Telegram Bot API / WhatsApp Gateway / SMTP
```

### 5.3 Alur Deteksi IP Conflict

1. Router mengirim log `dhcp,warning` (kandidat konflik, bukan bukti; lihat 3.3 AC-1) beserta log dhcp,info ke UDP 514.
2. Receiver memvalidasi IP sumber (allowlist), mengurai pesan, mengekstrak IP dan MAC, lalu mengirim ke Event Processor.
3. Processor membandingkan dengan snapshot terbaru lease dan ARP. Bila MAC berbeda, status menjadi `Indikasi`. Perbedaan MAC pada ARP saja diperlakukan sebagai **indikasi awal** dan wajib dicocokkan dengan lease dan log (bukan bukti tunggal).
4. Insiden dibuat atau diperbarui, lalu disimpan ke SQLite.
5. Cek cooldown. Bila lolos, Notifier mengirim `IP_CONFLICT` ke Telegram. UI desktop menerima *event* lewat Wails, PWA lewat SSE.
6. Poller berikutnya memvalidasi ulang. Bila normal selama 3 siklus, insiden *auto-resolved*.

**Konflik IP statis:** perangkat berIP statis tidak muncul di lease. Deteksi dilakukan lewat perbandingan ARP (MAC berubah pada IP yang sama dalam jendela waktu singkat) dan wajib ditandai *indikasi* dengan rekomendasi verifikasi manual. Deteksi lebih akurat (mis. ARP probe aktif) menjadi pengembangan lanjutan karena bersifat lebih intrusif.

### 5.4 Alur Device Down / Up

1. Admin mengatur `/tool netwatch` di router dengan skrip atau log `up`/`down` untuk host vital.
2. Router mengirim log ke UDP 514. Receiver mengurai nama host dan status.
3. Processor mencocokkan dengan inventaris, membuka atau menutup insiden, dan menghitung durasi.
4. Notifier mengirim `DEVICE_DOWN` / `DEVICE_RESOLVED`.

### 5.5 Konfigurasi RouterOS yang Disarankan (dilakukan manual oleh admin)

```
/user group add name=monitor-ro policy=read,api
/user add name=snm-api group=monitor-ro address=10.44.10.164/32 password=<kuat>
/system logging action add name=snm-remote target=remote remote=10.44.10.164 remote-port=514 bsd-syslog=yes
/system logging add topics=dhcp action=snm-remote
/system logging add topics=netwatch action=snm-remote
/ip service set api address=10.44.10.164/32
```

Aplikasi menyediakan halaman bantuan berisi perintah di atas sebagai referensi saja. Aplikasi tidak mengeksekusinya.

**Penting untuk Device Up/Down:** pada RouterOS 6, Netwatch tidak otomatis menulis log saat host mati atau hidup. Admin harus mengisi `up-script` dan `down-script` yang menulis log sendiri, lalu meneruskannya ke syslog dengan topik `script`. Tanpa langkah ini, fitur Device Up/Down tidak akan menerima data apa pun. Contoh (admin menyesuaikan IP dan nama untuk setiap perangkat):

```
/tool netwatch add host=192.168.10.52 interval=30s timeout=3s comment="AP-RUANG-1" up-script=":log info \"NETWATCH UP host=192.168.10.52 name=AP-RUANG-1\"" down-script=":log warning \"NETWATCH DOWN host=192.168.10.52 name=AP-RUANG-1\""
/system logging add topics=script action=snm-remote
```

Parser syslog aplikasi mengenali format `NETWATCH UP|DOWN host=<ip> name=<nama>` dari log bertopik `script`. Aplikasi menyediakan halaman bantuan yang menghasilkan perintah di atas untuk setiap perangkat di inventaris (hanya untuk disalin admin, tidak dieksekusi). Format tepatnya wajib diuji di router instansi sebelum Fase 2 (lihat 5.8).

### 5.6 Antarmuka HTTP Lokal (Ringkas)

| Endpoint | Metode | Fungsi |
| --- | --- | --- |
| `/api/v1/auth/login` | POST | Login teknisi, mengembalikan sesi. |
| `/api/v1/overview` | GET | Data kartu dashboard dan tren 7 hari. |
| `/api/v1/hosts?q=&filter=` | GET | Tabel DHCP/ARP. |
| `/api/v1/incidents?status=&type=` | GET | Daftar insiden. |
| `/api/v1/incidents/{id}` | GET | Detail insiden. |
| `/api/v1/incidents/{id}/ack` | POST | Akui insiden. |
| `/api/v1/incidents/{id}/resolve` | POST | Selesaikan insiden dengan catatan. |
| `/api/v1/devices` | GET | Status perangkat vital. |
| `/api/v1/stream` | GET (SSE) | Event real-time. |

Endpoint pengaturan (`/api/v1/settings/*`) hanya dapat diakses peran Admin dan secara default hanya dari aplikasi desktop (IPC).

### 5.7 Struktur Kode (Clean Architecture)

Belum ada di draf sebelumnya, jadi bagian ini ditambahkan. Kode Go wajib disusun dengan **Clean Architecture** agar logika bisnis (deteksi konflik, cooldown, auto-resolve) terpisah dari detail teknis (MikroTik, SQLite, Telegram, Wails). Hasilnya: logika inti bisa diuji tanpa router sungguhan, dan komponen teknis bisa diganti (mis. RouterOS 7 REST) tanpa menyentuh aturan bisnis.

**Aturan utama (Dependency Rule):** ketergantungan kode hanya boleh mengarah ke dalam. Lapisan dalam tidak boleh tahu apa pun tentang lapisan luar.

```
 Frameworks & Drivers   Wails, net/http, UDP 514, SQLite, go-routeros, klien Telegram
        │  mengimplementasikan port
        ▼
 Interface Adapters     handler IPC/HTTP, repository, parser syslog, presenter
        │  memanggil
        ▼
 Use Case               DetectIPConflict, HandleNetwatchEvent, AckIncident, SendNotification
        │  memakai
        ▼
 Domain (Entity)        Incident, Host, Device, CooldownPolicy, MACPair
```

### 5.7.1 Lapisan dan Isinya

| Lapisan | Isi | Boleh bergantung pada |
| --- | --- | --- |
| **Domain** | Entitas dan aturan bisnis murni: status insiden, perbandingan MAC lease vs ARP, aturan cooldown, hitung durasi downtime | Hanya pustaka standar Go |
| **Use Case** | Alur aplikasi satu per satu, plus interface (port) yang dibutuhkannya | Domain |
| **Adapter** | Implementasi port: klien RouterOS, penerima syslog, repository SQLite, pengirim Telegram, handler HTTP dan Wails | Use Case, Domain |
| **Platform / Main** | Konfigurasi, enkripsi kunci, tray, dan *composition root* yang merakit semua bagian | Semua lapisan |

### 5.7.2 Struktur Direktori

```
snm/
├── cmd/snm/main.go              # composition root: merakit dependensi, memulai goroutine
├── internal/
│   ├── domain/                  # entity + aturan bisnis (tanpa import eksternal)
│   │   ├── incident.go
│   │   ├── host.go
│   │   ├── device.go
│   │   └── cooldown.go
│   ├── usecase/
│   │   ├── port/                # interface (kontrak) untuk dunia luar
│   │   ├── detect_ip_conflict.go
│   │   ├── handle_netwatch_event.go
│   │   ├── sync_arp_lease.go
│   │   ├── manage_incident.go   # ack, resolve, auto-resolve
│   │   └── notify.go            # cooldown, retry, antrean
│   ├── adapter/
│   │   ├── routeros/            # go-routeros, hanya perintah baca
│   │   ├── syslog/              # UDP 514 + parser RFC 3164/5424
│   │   ├── sqlite/              # repository + migrasi
│   │   ├── notifier/            # telegram, email (whatsapp pasca-rilis)
│   │   ├── httpapi/             # REST + SSE + PWA
│   │   └── wailsbind/           # binding IPC tipis ke use case
│   └── platform/                # crypto (AES-GCM, DPAPI), config, tray, logger
├── frontend/                    # React + TypeScript + Tailwind
├── migrations/                  # berkas SQL berversi
└── build/                       # skrip Windows/Linux, installer
```

### 5.7.3 Port (Interface) Utama

Port didefinisikan di lapisan Use Case dan diimplementasikan di Adapter.

```go
// internal/usecase/port
type RouterReader interface {          // hanya metode baca
    ListARP(ctx context.Context) ([]domain.ARPEntry, error)
    ListLeases(ctx context.Context) ([]domain.Lease, error)
    Ping(ctx context.Context) error
}

type IncidentRepository interface {
    FindOpen(ctx context.Context, t domain.IncidentType, ip string) (*domain.Incident, error)
    Save(ctx context.Context, i *domain.Incident) error
    List(ctx context.Context, f domain.IncidentFilter) ([]domain.Incident, error)
}

type HostRepository interface { /* snapshot ARP + lease */ }
type DeviceRepository interface { /* inventaris perangkat vital */ }

type Notifier interface {
    Send(ctx context.Context, msg domain.Notification) error
}

type Clock interface { Now() time.Time }   // agar waktu bisa diuji
```

### 5.7.4 Aturan Penegakan

- **Read-only lewat desain:** `RouterReader` sengaja tidak punya metode tulis. Tidak ada jalur kode untuk mengubah router, sehingga prinsip read-only ditegakkan oleh compiler, bukan hanya konvensi.
- **Domain murni:** paket `domain` tidak boleh mengimpor paket di luar pustaka standar. Paket `usecase` hanya boleh mengimpor `domain`.
- **Adapter tipis:** handler Wails dan HTTP hanya menerjemahkan input/output (DTO) lalu memanggil use case. Tidak ada logika bisnis di sana.
- **DTO terpisah dari entity:** struktur JSON untuk UI/PWA dan baris SQLite tidak dipakai langsung sebagai entity domain.
- **Injeksi dependensi manual** di `main.go` (tanpa framework DI), dengan `context.Context` dan `Clock` dilewatkan eksplisit.
- **Penegakan otomatis di CI:** aturan impor diperiksa dengan `go-arch-lint` atau `depguard`. Build gagal bila ada pelanggaran arah dependensi.
- **Strategi uji per lapisan:** Domain dan Use Case diuji unit memakai *fake* (router dan Telegram tiruan, tanpa jaringan). Adapter diuji integrasi (SQLite sementara, simulator syslog UDP, mock API RouterOS).
- **Penambahan fitur:** fitur baru dimulai dari entity dan use case, baru adapter. Contoh dukungan RouterOS 7 cukup menambah satu adapter baru yang mengimplementasikan `RouterReader`.

### 5.8 Hal Berisiko yang Wajib Diputuskan atau Diverifikasi

Hal-hal berikut bisa membuat fitur inti tidak berfungsi bila dilewatkan, sehingga dijadikan syarat sebelum fase terkait dimulai.

**1. Format log RouterOS 6 harus diverifikasi dari router asli (gerbang Fase 1).** Pola log konflik IP pada AC modul 3.3 dan format log Netwatch masih perkiraan. Format `dhcp,warning` dapat berbeda antar versi RouterOS 6.

- Sebelum pengembangan parser dimulai, admin mengumpulkan minimal 20 contoh log nyata dari router instansi: log `dhcp,warning` (termasuk saat konflik diuji sengaja di segmen uji) dan log `script` dari Netwatch.
- Contoh log disimpan sebagai berkas uji di repositori dan menjadi dasar uji unit parser.
- Pola parser disimpan sebagai konfigurasi yang bisa diubah, bukan kode keras, agar bisa disesuaikan tanpa rilis ulang.
- Fase 1 baru dianggap selesai bila konflik IP yang dipicu sengaja di segmen uji terdeteksi dan terkirim ke Telegram.

**2. PWA butuh HTTPS agar fitur “pasang ke layar utama” dan cache offline berfungsi.** Browser hanya mengaktifkan service worker pada konteks aman (HTTPS atau `localhost`). Akses lewat `http://192.168.x.x` biasa akan tetap menampilkan halaman, tetapi tanpa instalasi PWA dan tanpa cache offline (AC-6 modul 3.6 tidak terpenuhi).

- **Keputusan:** HTTPS diwajibkan untuk server lokal mulai Fase 2, bukan lagi opsional di Fase 3.
- Aplikasi membuat sertifikat sendiri (CA internal + sertifikat server untuk IP/hostname PC). Admin memasang sertifikat CA itu di HP teknisi satu kali agar tidak muncul peringatan keamanan. Alternatifnya, admin memakai sertifikat dari CA instansi bila ada.
- **Mode cadangan:** bila HTTPS belum siap, server tetap dapat berjalan lewat HTTP. PWA berfungsi sebagai halaman web biasa dengan indikator “mode terbatas”, sedangkan fitur inti (lihat insiden, akui, selesaikan) tidak terpengaruh.
- Halaman bantuan di aplikasi menampilkan panduan pemasangan sertifikat untuk Android dan iOS.

**3. System Tray pada Wails v2 harus dibuktikan dulu (uji kelayakan Fase 0).** Wails v2 tidak punya dukungan tray bawaan (itu fitur Wails v3), sehingga dibutuhkan library tray tambahan yang kadang bentrok dengan event loop Wails, terutama di Linux.

- Buat prototipe kecil: aplikasi tampil, jendela disembunyikan, ikon tray tetap ada, proses dan goroutine latar belakang tetap berjalan, lalu jendela bisa dibuka lagi dari tray.
- Uji di Windows 10/11 dan Ubuntu. Hasil dicatat di `docs/ASSUMPTIONS.md`.
- **Rencana cadangan bila gagal:** pisahkan menjadi dua proses (inti pemantau tanpa UI sebagai background service, dan jendela Wails sebagai klien yang tersambung lewat server HTTP lokal), atau pindah ke Wails v3. Arsitektur 5.7 sudah mendukung pemisahan ini karena logika bisnis tidak bergantung pada Wails.

**4. Format syslog dari MikroTik tidak selalu standar.** Secara bawaan, pesan yang dikirim `/system logging action` tidak persis mengikuti format syslog BSD. Dianjurkan mengatur `bsd-syslog=yes` pada action remote (sudah dimasukkan ke contoh 5.5), tetapi parser tetap harus **toleran** terhadap dua bentuk: dengan header syslog lengkap dan hanya teks log. Bentuk yang sebenarnya dikonfirmasi dari contoh log nyata di gerbang Fase 1 (butir 1).

**5. Windows Firewall memblokir UDP 514 masuk secara bawaan.** Tanpa aturan masuk, paket syslog dari router tidak pernah sampai ke aplikasi dan tidak ada pesan error yang jelas.

- Installer atau panduan pengaturan awal membuat aturan firewall untuk UDP syslog (514 atau port alternatif) dan port server HTTP lokal, terbatas pada IP router dan segmen HP teknisi.
- Aplikasi menampilkan indikator **“Syslog: belum ada paket masuk”** di halaman Diagnostik bila tidak ada paket dari router selama 5 menit, lengkap dengan saran memeriksa firewall, IP tujuan di router, dan port.

**6. Telegram bisa diblokir di jaringan instansi.** Jaringan kedinasan sering memfilter akses keluar. Bila `api.telegram.org` tidak terjangkau dari PC monitoring, seluruh jalur peringatan mati.

- Wizard pengaturan awal menyertakan **uji koneksi ke Telegram** (kirim pesan uji) dan menampilkan penyebab kegagalan yang jelas (DNS, timeout, ditolak).
- Tersedia opsi **proxy keluar** (HTTP/SOCKS5) untuk klien Telegram.
- Kegagalan kirim yang berulang menampilkan banner merah di dashboard agar admin tahu peringatan tidak sampai ke HP.

**7. Keputusan teknis tambahan.**

- **SQLite tanpa CGO:** gunakan driver Go murni (mis. `modernc.org/sqlite`) agar `.exe` Windows bisa dibangun dan dilintaskan tanpa compiler C. Driver yang membutuhkan CGO tidak dipakai.
- **Allowlist syslog menerima beberapa IP per router:** router kadang mengirim dari IP interface lain, bukan IP yang dikira. Paket dari IP tak dikenal dibuang, dihitung, dan ditampilkan di Diagnostik beserta IP sumbernya agar admin dapat menambahkannya.
- **Waktu kejadian memakai jam PC monitoring** saat log diterima, sedangkan jam router disimpan sebagai informasi tambahan. Bila selisih keduanya lebih dari 2 menit, Diagnostik menampilkan peringatan agar admin memeriksa NTP.
- **Versi RouterOS minimal yang didukung** ditetapkan setelah pengecekan router instansi. Metode login API berubah sekitar RouterOS 6.43, sehingga klien API harus diuji pada versi yang dipakai. Router instansi memakai MikroTik CCR1009-7G-1C-1S+ dengan RouterOS 6.49.7 (stable), sudah di atas 6.43 sehingga memakai metode login baru. Versi ini menjadi acuan uji, dan versi lain dicantumkan di README bila diuji.

**8. Temuan dari contoh log nyata (tangkapan layar WinBox dari admin, 12 Nov 2026).** Contoh ini mengubah beberapa asumsi awal PRD. Bila ada pertentangan dengan bagian lain, butir ini yang berlaku.

- Format log: nomor, waktu, topik, pesan. Contoh ini berasal dari DHCP server `dhcp2` pada **LAN uji** `10.10.5.0/24` (ether4), bukan LAN produksi. Peta jaringan lengkap ada di bawah.
- Pola yang terkonfirmasi dari contoh:

```
dhcp, info      dhcp2 assigned 10.10.5.45 to DE:C9:0F:1A:0F:9E
dhcp, info      dhcp2 deassigned 10.10.5.99 from B2:14:02:90:64:97
dhcp, warning   dhcp2 offering lease 10.10.5.45 for DE:C9:0F:1A:0F:9E without success
```

- **Log `dhcp,warning` bukan bukti konflik.** Pesan "offering lease ... without success" berarti penawaran lease belum diterima klien. Pada contoh, peringatan untuk 10.10.5.45 pukul 12:10:58 langsung disusul `assigned 10.10.5.45` ke MAC yang sama pada detik yang sama, sehingga kemungkinan besar jinak. Pola konflik pada draf awal ("already in use", "duplicate") hanya perkiraan dan belum terbukti.
- **Konsekuensi desain:**
  1. Router meneruskan **semua log `dhcp`** (info dan warning), bukan hanya warning (sudah diperbarui di 5.5). Ini menggantikan keterangan "memfilter topik dhcp,warning" di 5.1.
  2. Peringatan menjadi insiden hanya bila memenuhi syarat 3.3 AC-1.
  3. Pola regex disimpan di konfigurasi dan menerima nama server DHCP apa pun, tidak hanya `dhcp2`.
- **MAC acak sangat umum.** Banyak MAC pada contoh (awalan 2E, DE, B2, 22, EA) bertanda *locally administered*, yaitu HP modern yang memakai MAC acak per jaringan. Aplikasi menandai MAC seperti ini (`is_randomized_mac`, bit 0x02 pada oktet pertama) dan tidak menganggap perubahan MAC antar waktu pada satu IP sebagai indikasi konflik. Yang dihitung hanya perbedaan MAC lease dan ARP **pada saat yang sama**.
- **Belum ada contoh konflik IP yang sungguhan.** Gerbang Fase 1 (butir 1) tetap berlaku: admin memicu konflik uji di segmen uji (mis. dua perangkat memakai IP statis yang sama) lalu mengirim log dan entri ARP-nya. Pola konflik disahkan setelah itu.
- Cuplikan log disimpan sebagai berkas uji `testdata/syslog/dhcp_20261112.txt`. Data simulasi prototipe (192.168.10.x) diganti 10.10.5.x pada berkas uji.

**Peta jaringan nyata (dari `/ip address print` dan `/ip dhcp-server print`, 12 Nov 2026):**

| Jaringan | Interface | Subnet | DHCP server | Catatan |
| --- | --- | --- | --- | --- |
| WAN | ether1-WAN | 10.1.81.148/30 | tidak ada | Jalur internet |
| **LAN produksi** | ether2-LAN | **10.10.2.0/24** | dhcp1 (pool dhcp\_pool0) | Jaringan yang dipantau |
| LAN uji | ether4 | 10.10.5.0/24 | dhcp2 (pool dhcp\_pool2) | Komentar router: "testing LAN Syahrul". Sumber log contoh di atas |
| Cadangan | ether7 | 10.1.95.116/30 | tidak ada | Dinonaktifkan ("BACKUP JOL") |

- **Log contoh berasal dari LAN uji.** Pola yang sama harus diverifikasi lagi pada LAN produksi (`dhcp1`), terutama untuk konflik IP yang sungguhan.
- **Lease time kedua DHCP server hanya 10 menit.** Akibatnya baris `assigned` dan `deassigned` jauh lebih sering. `deassigned` hanya berarti lease habis, **bukan** perangkat mati, sehingga tidak dipakai sebagai sinyal Device Down. Status online ditentukan dari ARP dan Netwatch. Log `dhcp,info` disimpan 14 hari, sedangkan `dhcp,warning` tetap 90 hari.
- **Router memakai dua DHCP server dan dua subnet.** Aplikasi memantau satu router dengan `local_subnets` berisi daftar subnet (produksi `10.10.2.0/24`, uji `10.10.5.0/24` hanya selama masa uji). Migrasi menambahkan kolom `dhcp_server` pada tabel `hosts` dan `incidents`.
- **Tidak ada interface bridge.** Nama interface nyata adalah `ether2-LAN` dan `ether4`, bukan `bridge-lan` seperti pada data simulasi prototipe. Nama interface selalu diambil dari router, tidak ditulis keras di kode.

**9. Penempatan PC monitoring (keputusan admin, 12 Nov 2026): `10.44.10.164`, bukan LAN produksi `10.10.2.x`.** Segmen `10.44.10.x` tidak muncul pada `/ip address print` router (yang terlihat hanya 10.1.81.148/30, 10.10.2.0/24, 10.10.5.0/24, dan ether7 yang nonaktif), sehingga PC dijangkau lewat routing, bukan terhubung langsung ke interface router. Contoh konfigurasi di 5.5 dan 5.9 sudah memakai `10.44.10.164`, dan `public_base_url` PWA diisi `https://10.44.10.164:<port>`. **Sudah dikonfirmasi admin:** HP teknisi boleh mengakses PC. **Juga sudah dikonfirmasi admin aman (12 Nov 2026):** keterjangkauan router ke PC, IP sumber paket syslog, API router dari PC (firewall `input`), akses internet keluar PC, dan IP `10.44.10.164` dibuat tetap (statis atau reservasi DHCP). Nilai IP sumber paket syslog yang sebenarnya dicatat di docs/ASSUMPTIONS.md dan ditampilkan Diagnostik dari paket pertama yang masuk.

| Pemeriksaan | Cara memeriksa di router | Alasan |
| --- | --- | --- |
| Router bisa menjangkau PC | `/ping <IP-PC-monitoring>` dan `/tool traceroute <IP-PC-monitoring>`, rute dilihat di `/ip route print` | Syslog dan NetFlow dikirim dari router ke PC |
| IP sumber paket syslog dan NetFlow | Router mengirim dari IP interface keluar menuju segmen itu, belum tentu `10.10.2.1`. IP ini dicatat dari paket pertama | Allowlist syslog dan NetFlow harus memuat IP tersebut (Diagnostik menampilkannya, 5.8 butir 7) |
| PC bisa menjangkau API router | `/ip firewall filter print where chain=input` dan `/ip service print` | Chain `input` bisa memblokir port 8728/8729 dari segmen lain. Pembatasan `/ip service set api address=` harus memakai IP PC |
| HP teknisi bisa menjangkau PC | Pastikan Wi-Fi teknisi dapat mencapai port HTTPS aplikasi di `10.44.10.x` (aturan antar-segmen) | Keputusan "lokal saja" berarti PWA hanya terbuka bila segmen teknisi boleh mengakses PC. Sertifikat HTTPS dibuat untuk IP PC ini |
| PC punya akses internet keluar | Uji dengan tombol Kirim Pesan Uji Telegram | Peringatan Telegram dikirim dari PC di segmen `10.44.10.x` |
| IP PC tetap | Atur IP statis atau reservasi DHCP | Allowlist, aturan firewall, dan sertifikat bergantung pada IP yang tidak berubah |

Diagnostik aplikasi menampilkan hasil pemeriksaan yang bisa dilakukan dari sisi PC (API terjangkau, paket syslog dan NetFlow masuk, Telegram terjangkau), sehingga penyebab "tidak ada data" bisa dilacak ke jalur jaringan yang tepat.

**9a. Detail konektivitas (pelengkap butir 9) untuk PC monitoring di jaringan 10.44.10.x (keputusan admin, bukan di 10.10.2.x).**

Jaringan `10.44.10.x` tidak terdaftar sebagai jaringan yang terhubung langsung ke router (daftar `/ip address print` hanya berisi 10.1.81.148/30, 10.10.2.0/24, 10.10.5.0/24, dan 10.1.95.116/30). Artinya semua komunikasi antara router dan PC lewat jalur routing, bukan satu segmen yang sama. **Admin sudah mengonfirmasi bahwa jalur ini aman (butir 9). Daftar periksa berikut menjadi pegangan saat pengujian, dan hasilnya dicatat di docs/ASSUMPTIONS.md**:

1. **Router ke PC:** dari router, `/ping` dan `/tool traceroute` ke IP PC (10.44.10.x) berhasil. Catat jalurnya.
2. **PC ke router (RouterOS API):** catat alamat router yang dijangkau PC (mis. alamat sisi WAN atau alamat lain). Pada router, `/ip service` untuk `api` atau `api-ssl` harus mengizinkan IP PC, dan firewall chain `input` harus mengizinkan TCP 8728 atau 8729 dari PC. Aplikasi tidak mengubah ini sendiri.
3. **Alamat sumber paket dari router:** karena PC berada di jaringan lain, router bisa mengirim syslog dan NetFlow dengan alamat sumber yang tidak terduga (mis. alamat WAN). Admin mengatur `src-address` pada `/system logging action` dan pada `/ip traffic-flow target` agar tetap, lalu IP itu dimasukkan ke allowlist aplikasi.
4. **Firewall di sepanjang jalur dan di PC:** UDP 514 (syslog) dan UDP 2055 (NetFlow) dari router ke PC, serta port server HTTP(S) lokal ke HP teknisi, harus dibuka. Aturan Windows Firewall mengikuti 5.8 butir 5.
5. **Akses internet dari 10.44.10.x:** PC harus bisa mencapai `api.telegram.org` (lihat 5.8 butir 6). Bila keluar lewat proxy, isi opsi proxy aplikasi.
6. **HP teknisi ke PC:** Wi-Fi atau VLAN yang dipakai teknisi harus bisa menjangkau 10.44.10.x pada port server HTTP(S) lokal. Bila tidak, PWA tidak bisa dibuka dan peringatan Telegram tetap menjadi jalur utama. Keputusan "lokal saja" bergantung pada butir ini.

Yang masih perlu dicatat: IP router yang dijangkau PC dan IP sumber paket syslog atau NetFlow yang sebenarnya. IP PC tetap 10.44.10.164 (bukan DHCP dinamis) agar allowlist dan tautan Telegram tidak rusak.

### 5.9 Sumber Data Bandwidth per IP (gerbang kelayakan untuk modul 3.7)

Router MikroTik tidak menyediakan "pemakaian per IP" siap pakai lewat API baca-saja, sehingga sumber datanya harus dipilih dan diuji dulu.

| Metode | Cara kerja | Kelebihan | Kekurangan |
| --- | --- | --- | --- |
| **NetFlow v5 (Traffic Flow), pilihan utama** | Router mengirim catatan aliran ke UDP 2055 di PC monitoring. Aplikasi menjumlahkan byte per IP. | Pasif dan sesuai prinsip read-only (admin yang mengaktifkannya di router). Akurat per IP. | Hanya trafik yang melewati router. FastTrack tidak aktif di router instansi (terkonfirmasi), uji ulang bila diaktifkan. Menambah sedikit beban CPU router. |
| Simple Queue per IP (cadangan) | Admin membuat queue tanpa batas per IP, aplikasi membaca counter `/queue/simple` lewat API. | Mudah dibaca. | Konfigurasi manual per IP. Salah atur bisa mengubah lalu lintas. |
| IP Accounting (cadangan, hanya RouterOS 6) | Router menyimpan snapshot per pasangan IP yang dibaca lewat HTTP. | Tanpa NetFlow. | Fitur dihapus di RouterOS 7. Perlu diverifikasi di router instansi. |

Contoh konfigurasi NetFlow yang dilakukan admin secara manual (nama interface dan IP disesuaikan):

```
/ip traffic-flow set enabled=yes interfaces=ether2-LAN active-flow-timeout=1m inactive-flow-timeout=15s cache-entries=64k
/ip traffic-flow target add dst-address=10.44.10.164 port=2055 version=5
```

Pilih hanya interface sisi LAN, jangan `all`, agar trafik tidak terhitung ganda. Arah dihitung dari IP lokal: **unggah** bila IP lokal menjadi sumber, **unduh** bila IP lokal menjadi tujuan.

**Gerbang kelayakan (sebelum modul 3.7 dikerjakan):**

1. **Terkonfirmasi (screenshot admin, 12 Nov 2026):** `/ip traffic-flow` tersedia di router instansi (MikroTik CCR1009-7G-1C-1S+, RouterOS 6.49.7). Saat ini belum aktif (`enabled=no`) dan belum ada target. Beban router aman: CPU 5% dan memori bebas sekitar 1,7 GB dari 2 GB. Dua pengaturan bawaan harus diubah admin: `interfaces=all` diganti hanya interface LAN produksi (ether2-LAN, bukan ether4 yang untuk uji) agar trafik tidak terhitung ganda, dan `active-flow-timeout=30m` diganti `1m` (agar data masuk tepat waktu ke bucket 5 menit).
2. **Terkonfirmasi:** FastTrack tidak aktif. Perintah `/ip firewall filter print where action=fasttrack-connection` tidak menghasilkan aturan apa pun. Bila admin menambahkan FastTrack di kemudian hari, butir 3 diulang.
3. Admin mengaktifkan Traffic Flow ke PC monitoring selama minimal 1 jam. Total byte satu IP dari penerima NetFlow dibandingkan dengan pembanding independen (counter interface atau Torch). Selisih harus 5% atau kurang. Hasil dicatat di `docs/ASSUMPTIONS.md`.
4. Bila uji butir 3 gagal, pilihannya: (a) beralih ke Simple Queue per IP, atau (b) halaman Bandwidth menampilkan label "data mungkin lebih rendah dari pemakaian sebenarnya". Bila tidak satu pun dapat diterima, modul ini ditunda. Fitur inti tidak terpengaruh.

**Penempatan di Clean Architecture (5.7):** entity `TrafficSample`, port `TrafficSource` dan `TrafficRepository`, use case `AggregateTraffic` (bucket 5 menit) dan `QueryTopUsers`, adapter `adapter/netflow/` (UDP 2055, parser v5) dan `adapter/sqlite/traffic`. Sumber lain cukup mengimplementasikan `TrafficSource`.

**Beban dan privasi:**

- Penerima mampu memproses minimal 5.000 catatan per detik. Agregasi dilakukan di memori dan ditulis ke SQLite tiap 60 detik secara batch.
- Hanya paket dari IP router terdaftar yang diterima. Firewall PC membuka UDP 2055 khusus dari IP router.
- Hanya volume per IP yang disimpan, tanpa alamat tujuan, nama situs, atau isi trafik. Kebijakan pemantauan ini disepakati dengan Tim IT sebelum diaktifkan.

## 6. Rancangan Skema Database SQLite

Pengaturan: `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`. Semua waktu disimpan sebagai UTC (`INTEGER` epoch detik) dan ditampilkan sesuai zona waktu pengguna. Migrasi memakai nomor versi (`schema_migrations`).

```sql
-- Router yang dipantau
CREATE TABLE routers (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,                 -- 'Router Kantor Utama'
  host TEXT NOT NULL,
  api_port INTEGER NOT NULL DEFAULT 8728,
  use_tls INTEGER NOT NULL DEFAULT 0,
  verify_cert INTEGER NOT NULL DEFAULT 0,
  username TEXT NOT NULL,
  password_enc BLOB NOT NULL,         -- AES-256-GCM (nonce||ciphertext)
  syslog_source_ip TEXT NOT NULL,     -- allowlist syslog
  status TEXT NOT NULL DEFAULT 'unknown', -- connected|disconnected|unknown
  last_poll_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

-- Pengguna aplikasi (desktop & PWA)
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,        -- Argon2id
  role TEXT NOT NULL CHECK (role IN ('admin','technician')),
  is_active INTEGER NOT NULL DEFAULT 1,
  failed_attempts INTEGER NOT NULL DEFAULT 0,
  locked_until INTEGER,
  created_at INTEGER NOT NULL
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);

-- Inventaris perangkat vital (Device Up/Down)
CREATE TABLE devices (
  id INTEGER PRIMARY KEY,
  router_id INTEGER NOT NULL REFERENCES routers(id),
  name TEXT NOT NULL,                 -- 'AP-RUANG-1'
  ip_address TEXT NOT NULL,
  mac_address TEXT,
  type TEXT NOT NULL CHECK (type IN ('access_point','switch','server','printer','other')),
  location TEXT,
  priority TEXT NOT NULL DEFAULT 'normal' CHECK (priority IN ('low','normal','high')),
  current_state TEXT NOT NULL DEFAULT 'unknown' CHECK (current_state IN ('up','down','unknown')),
  state_since INTEGER,
  is_monitored INTEGER NOT NULL DEFAULT 1,
  UNIQUE (router_id, ip_address)
);

-- Snapshot ARP + lease terkini (satu baris per IP/MAC)
CREATE TABLE hosts (
  id INTEGER PRIMARY KEY,
  router_id INTEGER NOT NULL REFERENCES routers(id),
  ip_address TEXT NOT NULL,
  hostname TEXT,
  lease_mac TEXT,
  arp_mac TEXT,
  source TEXT NOT NULL CHECK (source IN ('dhcp','static')),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','indication')),
  interface TEXT,
  lease_status TEXT,                  -- bound|waiting|offered
  first_seen INTEGER NOT NULL,
  last_seen INTEGER NOT NULL,
  UNIQUE (router_id, ip_address)
);
CREATE INDEX idx_hosts_mac ON hosts(lease_mac, arp_mac);

-- Riwayat perubahan ARP/lease (untuk analisis dan bukti)
CREATE TABLE arp_lease_log (
  id INTEGER PRIMARY KEY,
  router_id INTEGER NOT NULL REFERENCES routers(id),
  ip_address TEXT NOT NULL,
  lease_mac TEXT,
  arp_mac TEXT,
  interface TEXT,
  change_type TEXT NOT NULL,          -- new|mac_changed|expired|removed
  observed_at INTEGER NOT NULL
);
CREATE INDEX idx_arplog_ip_time ON arp_lease_log(ip_address, observed_at);

-- Log syslog mentah (disaring: hanya topik relevan)
CREATE TABLE syslog_events (
  id INTEGER PRIMARY KEY,
  router_id INTEGER REFERENCES routers(id),
  topics TEXT NOT NULL,               -- 'dhcp,warning' | 'netwatch,info'
  message TEXT NOT NULL,
  parsed_ip TEXT,
  parsed_mac TEXT,
  received_at INTEGER NOT NULL
);
CREATE INDEX idx_syslog_time ON syslog_events(received_at);

-- Insiden (IP Conflict & Device Down)
CREATE TABLE incidents (
  id INTEGER PRIMARY KEY,
  type TEXT NOT NULL CHECK (type IN ('IP_CONFLICT','DEVICE_DOWN')),
  router_id INTEGER NOT NULL REFERENCES routers(id),
  device_id INTEGER REFERENCES devices(id),   -- untuk DEVICE_DOWN
  ip_address TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('info','warning','critical')),
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new','acknowledged','in_progress','resolved')),
  source TEXT NOT NULL,               -- 'DHCP warning / Syslog' | 'Netwatch' | 'ARP poll'
  lease_mac TEXT,
  arp_mac TEXT,
  interface TEXT,
  lease_status TEXT,
  occurrence_count INTEGER NOT NULL DEFAULT 1,
  first_detected_at INTEGER NOT NULL,
  last_detected_at INTEGER NOT NULL,
  acknowledged_by INTEGER REFERENCES users(id),
  acknowledged_at INTEGER,
  resolved_by INTEGER REFERENCES users(id),
  resolved_at INTEGER,
  resolution TEXT,                    -- manual|auto|timeout
  resolution_note TEXT,
  downtime_seconds INTEGER,           -- untuk DEVICE_DOWN
  checklist_state TEXT                -- JSON: langkah 1-5
);
CREATE UNIQUE INDEX uq_open_incident
  ON incidents(type, router_id, ip_address) WHERE status <> 'resolved';
CREATE INDEX idx_incidents_status ON incidents(status, last_detected_at);

CREATE TABLE incident_events (        -- audit & linimasa insiden
  id INTEGER PRIMARY KEY,
  incident_id INTEGER NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  user_id INTEGER REFERENCES users(id),
  action TEXT NOT NULL,               -- created|updated|ack|note|resolved|auto_resolved|notified|suppressed
  detail TEXT,
  created_at INTEGER NOT NULL
);

-- Pengaturan notifikasi (satu baris)
CREATE TABLE notification_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  dashboard_enabled INTEGER NOT NULL DEFAULT 1,
  telegram_enabled INTEGER NOT NULL DEFAULT 0,
  telegram_bot_token_enc BLOB,
  telegram_chat_id TEXT,
  whatsapp_enabled INTEGER NOT NULL DEFAULT 0,
  whatsapp_webhook_url TEXT,
  whatsapp_secret_enc BLOB,
  email_enabled INTEGER NOT NULL DEFAULT 0,
  smtp_config_enc BLOB,               -- JSON terenkripsi (host, port, user, pass, to)
  email_digest_time TEXT DEFAULT '07:30',
  validation_interval_sec INTEGER NOT NULL DEFAULT 60 CHECK (validation_interval_sec BETWEEN 15 AND 600),
  cooldown_minutes INTEGER NOT NULL DEFAULT 10 CHECK (cooldown_minutes BETWEEN 1 AND 120),
  quiet_hours_start TEXT,
  quiet_hours_end TEXT,
  public_base_url TEXT                -- untuk deep link ke PWA
);

-- Antrean & riwayat notifikasi (cooldown, retry, bukti kirim)
CREATE TABLE notification_log (
  id INTEGER PRIMARY KEY,
  incident_id INTEGER REFERENCES incidents(id),
  channel TEXT NOT NULL,              -- telegram|whatsapp|email|dashboard
  template TEXT NOT NULL,             -- IP_CONFLICT|DEVICE_DOWN|DEVICE_RESOLVED
  dedup_key TEXT NOT NULL,            -- 'IP_CONFLICT:192.168.10.25'
  status TEXT NOT NULL CHECK (status IN ('queued','sent','failed','suppressed')),
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at INTEGER NOT NULL,
  sent_at INTEGER
);
CREATE INDEX idx_notif_dedup ON notification_log(dedup_key, created_at);

CREATE TABLE app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL                 -- retensi, zona waktu, port HTTP, dll.
);
```

### 6.1 Aturan Data

- Retensi default: `syslog_events` dan `arp_lease_log` 90 hari, `notification_log` 180 hari, `incidents` 2 tahun.
- Tren 7 hari dihitung dari `incidents.first_detected_at` dikelompokkan per hari lokal.
- `uq_open_incident` menjamin hanya satu insiden terbuka per jenis dan IP (AC-2 modul 3.3).
- Kolom terenkripsi tidak pernah diekspor dalam ekspor CSV atau pencadangan tanpa kata sandi pencadangan.

### 6.2 Tabel Tambahan untuk Bandwidth per IP (modul 3.7)

```sql
-- Agregasi pemakaian per IP (bucket 5 menit)
CREATE TABLE traffic_5m (
  router_id INTEGER NOT NULL REFERENCES routers(id),
  ip_address TEXT NOT NULL,
  bucket_start INTEGER NOT NULL,       -- epoch detik, kelipatan 300
  bytes_in INTEGER NOT NULL DEFAULT 0,   -- unduh (IP sebagai tujuan)
  bytes_out INTEGER NOT NULL DEFAULT 0,  -- unggah (IP sebagai sumber)
  packets_in INTEGER NOT NULL DEFAULT 0,
  packets_out INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (router_id, ip_address, bucket_start)
) WITHOUT ROWID;

-- Ringkasan per jam dan per hari: kolom sama, bucket_start kelipatan 3600 / 86400
CREATE TABLE traffic_1h ( /* sama dengan traffic_5m */ );
CREATE TABLE traffic_1d ( /* sama dengan traffic_5m */ );

CREATE TABLE bandwidth_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  enabled INTEGER NOT NULL DEFAULT 0,
  source TEXT NOT NULL DEFAULT 'netflow_v5' CHECK (source IN ('netflow_v5','simple_queue','ip_accounting')),
  flow_port INTEGER NOT NULL DEFAULT 2055,
  local_subnets TEXT NOT NULL,         -- JSON array CIDR, mis. ["10.10.2.0/24"]
  alert_enabled INTEGER NOT NULL DEFAULT 0,
  alert_threshold_mbps REAL,
  alert_consecutive_buckets INTEGER NOT NULL DEFAULT 3
);
```

Migrasi juga menambahkan nilai `BANDWIDTH_HIGH` pada batasan `incidents.type` dan menambahkan kolom `is_randomized_mac INTEGER NOT NULL DEFAULT 0` pada tabel `hosts` (lihat 5.8 butir 8).

## 7. Format Payload Notifikasi Telegram

Pesan dikirim lewat `POST https://api.telegram.org/bot<TOKEN>/sendMessage` dengan `parse_mode=HTML`, `disable_web_page_preview=true`. Semua nilai dinamis wajib di-*escape* HTML. Tombol *inline* membuka tautan detail ke PWA.

### 7.1 Template `IP_CONFLICT`

```
🚨 <b>SMART NETWORK ALERT</b>
<b>IP CONFLICT</b>

Router: {router_name}
IP: <code>{ip}</code>
Status: Perlu verifikasi
MAC (lease): <code>{lease_mac}</code>
MAC (ARP): <code>{arp_mac}</code>
Interface: {interface}
Sumber: {source}
Terdeteksi: {event_time} WIB
Kejadian: {occurrence_count}×

🔎 Periksa DHCP lease dan ARP.
Catatan: perubahan MAC adalah indikasi awal, verifikasi manual diperlukan.
```

Contoh:

```
🚨 SMART NETWORK ALERT
IP CONFLICT

Router: Kantor Utama
IP: 192.168.10.25
Status: Perlu verifikasi
MAC (lease): 00:1B:21:AA:3C:11
MAC (ARP): F8:2C:18:4D:AA:03
Interface: bridge-lan
Sumber: DHCP warning / Syslog
Terdeteksi: 11:16 WIB
Kejadian: 1×

🔎 Periksa DHCP lease dan ARP.
```

### 7.2 Template `DEVICE_DOWN`

```
🔴 <b>SMART NETWORK ALERT</b>
<b>DEVICE DOWN</b>

Perangkat: <b>{device_name}</b> ({device_type})
IP: <code>{ip}</code>
Lokasi: {location}
Router: {router_name}
Prioritas: {priority}
Mati sejak: {down_since} WIB
Sumber: Netwatch

⚠️ Periksa daya, kabel, dan port switch perangkat.
```

### 7.3 Template `DEVICE_RESOLVED`

Dipakai untuk perangkat yang kembali UP maupun konflik IP yang terselesaikan.

```
✅ <b>SMART NETWORK RESOLVED</b>
<b>{incident_type_label}</b>

{device_or_ip_label}: <b>{name_or_ip}</b>
IP: <code>{ip}</code>
Status: Kembali normal
Durasi gangguan: {duration}
Diselesaikan oleh: {resolved_by_or_"otomatis"}
Selesai: {resolved_time} WIB
```

Contoh: `Durasi gangguan: 12 menit 40 detik`.

### 7.4 Aturan Payload

| Aturan | Detail |
| --- | --- |
| Tombol inline | `[Buka Detail]` → `{public_base_url}/#/incidents/{id}`, `[Akui]` → callback `ack:{id}` (opsional, bila bot webhook/long-poll aktif) |
| Panjang pesan | Maks. 4.096 karakter, dipotong dengan penanda “…” |
| Dedup | Kunci `{TEMPLATE}:{ip atau device_id}`. Ditahan selama `cooldown_minutes`. Hasil ditahan dicatat `suppressed` |
| Resolved | Tidak terkena cooldown, tetapi hanya dikirim bila notifikasi awal pernah terkirim |
| Rate limit Telegram | Maks. 1 pesan/dtk per chat. Antrean menyesuaikan, balasan 429 menghormati `retry_after` |
| Privasi | Tidak menyertakan kredensial, token, atau data pribadi di luar IP, MAC, dan nama perangkat |
| Webhook WhatsApp | Isi pesan sama dalam JSON `{"template","incident_id","text","url"}` dengan tanda tangan HMAC-SHA256 di header `X-Signature` |

### 7.5 Template `BANDWIDTH_HIGH` (opsional, Fase 3)

```
📶 <b>SMART NETWORK ALERT</b>
<b>BANDWIDTH TINGGI</b>

IP: <code>{ip}</code> ({hostname})
Rata-rata 5 menit: {rate_mbps} Mbps (ambang {threshold_mbps} Mbps)
Pemakaian 1 jam: unduh {down_gb} GB / unggah {up_gb} GB
Router: {router_name}
Terdeteksi: {event_time} WIB

Periksa perangkat atau aplikasi yang memakai bandwidth besar.
```

Kunci dedup: `BANDWIDTH_HIGH:{ip}`. Cooldown dan aturan payload lain sama dengan 7.4.

## 8. Roadmap Pengembangan Bertahap

| Fase | Perkiraan | Cakupan | Kriteria Selesai |
| --- | --- | --- | --- |
| **Fase 0: Fondasi** | Minggu 1–2 | Kerangka Go + Wails v2, React/Tailwind, SQLite + migrasi, enkripsi kredensial, login admin, System Tray (uji kelayakan, 5.8), uji koneksi Telegram, logging | Aplikasi berjalan di tray, UI kosong terhubung ke Go lewat IPC, kredensial tersimpan terenkripsi, tray tetap hidup saat jendela disembunyikan (Windows dan Linux), pesan uji Telegram terkirim |
| **Fase 1: MVP** | Minggu 3–8 | Koneksi router + uji kredensial (cek policy read-only), penerima syslog UDP 514, poller ARP/lease, tabel DHCP & ARP, deteksi dan riwayat IP Conflict, notifikasi dashboard dan Telegram dengan cooldown, dashboard dasar | Skenario uji konflik IP terdeteksi dan sampai ke Telegram ≤ 30 dtk, tanpa tulis ke router |
| **Fase 2: Device Up/Down & PWA** | Minggu 9–12 | Inventaris perangkat, parsing netwatch, durasi downtime, `DEVICE_DOWN` dan `DEVICE_RESOLVED`, server HTTP lokal, PWA (beranda + detail), akun teknisi, SSE, deep link, HTTPS lokal dengan sertifikat internal, panduan script Netwatch | Teknisi dapat membuka tautan dari Telegram dan mengakui insiden dari HP |
| **Fase 3: Penguatan** | Minggu 13–16 | Alur investigasi 5 langkah lengkap, auto-resolve, anti-flapping, quiet hours, ringkasan email, ekspor CSV/PDF, backup otomatis, halaman diagnostik, auto-start (service), Bandwidth per IP (3.7, bila gerbang kelayakan 5.9 lolos) | Uji ketahanan 7 hari tanpa henti, restart otomatis bekerja, pencadangan terverifikasi, total bandwidth per IP selisih 5% atau kurang dari pembanding |
| **Fase 4: Rilis Penuh (v1.0)** | Minggu 17–20 | UAT di lingkungan instansi, uji keamanan (pentest ringan, audit allowlist perintah), dokumentasi pengguna dan panduan konfigurasi RouterOS, installer bertanda tangan, pelatihan admin | Persetujuan UAT, seluruh Acceptance Criteria lulus, tidak ada temuan keamanan tingkat tinggi |
| **Pasca-rilis** | Setelah v1.0 | Webhook WhatsApp Gateway, multi-router, dukungan RouterOS 7 (REST API), deteksi konflik IP statis lanjutan (ARP probe terkontrol), laporan berkala, integrasi tiket helpdesk | Dinilai berdasarkan umpan balik penggunaan |

### 8.1 Strategi Pengujian

- **Unit test:** parser syslog (kumpulan contoh log RouterOS 6 nyata), logika korelasi lease/ARP, cooldown, dan enkripsi.
- **Integrasi:** emulator API RouterOS (mock) dan simulator syslog UDP untuk skenario konflik, flapping, dan router terputus.
- **Uji keamanan otomatis:** memastikan klien API menolak semua perintah selain `print`/`getall`.
- **Uji lapangan:** router uji di segmen terpisah sebelum produksi.

### 8.2 Risiko dan Mitigasi

| Risiko | Dampak | Mitigasi |
| --- | --- | --- |
| Format log `dhcp,warning` bervariasi antar versi RouterOS 6 | Konflik tidak terdeteksi | Parser berbasis pola yang dapat dikonfigurasi, koleksi sampel log dari router instansi sejak Fase 0 |
| Konflik IP statis tidak tercatat di log DHCP | Konflik terlewat | Pencocokan ARP dengan baseline inventaris, ditandai sebagai indikasi, dan rencana ARP probe di fase lanjutan |
| Port 514 butuh hak istimewa atau sudah dipakai | Syslog tidak diterima | Port alternatif yang dapat dikonfigurasi, pemeriksaan saat startup, panduan `setcap` |
| Kebocoran kredensial | Risiko keamanan | Enkripsi at-rest, user router read-only, allowlist IP, peringatan bila policy terlalu luas |
| Kelelahan notifikasi (*alert fatigue*) | Notifikasi diabaikan | Cooldown, penggabungan flapping, prioritas perangkat, jam tenang |
| Router mati total sehingga syslog dan polling berhenti | Tidak ada peringatan | Deteksi *silence* (tidak ada data > 3 siklus) memicu peringatan *Router Terputus* |

### 8.3 Keputusan Proyek (jawaban pertanyaan terbuka)

| Pertanyaan | Keputusan | Dampak pada PRD |
| --- | --- | --- |
| Jumlah router atau lokasi awal | **1 router** | Cakupan v1.0 = satu router. Skema tetap memakai `router_id` agar multi-router bisa ditambah pasca-rilis tanpa migrasi besar. |
| PC monitoring 24/7 dan terpasang sebagai service | **Opsional** | Aplikasi tetap berjalan sebagai System Tray. Mode service (Windows Service / `systemd`) menjadi pilihan, bukan syarat. Bila PC mati, tidak ada pemantauan dan tidak ada notifikasi. Admin perlu tahu batas ini. Peringatan *Router Terputus* dan data yang tertinggal hanya dikejar saat aplikasi hidup kembali (poller menyinkronkan ulang ARP/lease). |
| Kanal notifikasi MVP | **Telegram saja** | WhatsApp Gateway dikeluarkan dari Fase 3 dan dipindah ke pasca-rilis. Kolom WhatsApp di skema tetap disiapkan tetapi tidak aktif. |
| Akses PWA lewat VPN | **Jaringan lokal saja, tanpa VPN.** | PWA hanya dibuka dari LAN atau Wi-Fi internal. VPN tidak termasuk v1.0. |
| Pemilik keputusan retensi data | **Tim IT** | Nilai bawaan (log mentah 90 hari kecuali dhcp,info 14 hari, insiden 2 tahun) berlaku sampai Tim IT menetapkan lain. Pengaturan retensi dapat diubah admin. |

**Catatan akses PWA.** Peringatan Telegram dikirim aplikasi secara *outbound* dari PC monitoring, jadi tetap sampai ke HP teknisi di mana pun selama PC punya internet. Tautan **Buka Detail** ke PWA hanya berfungsi bila HP tersambung ke jaringan lokal kantor. Di luar kantor, teknisi tetap mendapat ringkasan insiden (IP, MAC, interface, waktu) dari pesan Telegram dan membuka detail lengkap setelah kembali ke Wi-Fi internal. Admin mengisi `public_base_url` dengan IP atau hostname lokal PC monitoring. Aplikasi tidak membuka port ke internet publik.
