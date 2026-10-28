# Panduan Video Demo (durasi target 6–8 menit)

Dokumen ini adalah naskah untuk direkam; video tidak dibuat otomatis oleh project.

1. **Arsitektur (1 menit):** jelaskan publisher mock → Mosquitto → subscriber Go → PostgreSQL; service geofence → RabbitMQ exchange `fleet.events` → queue `geofence_alerts` → worker; Gin API membaca PostgreSQL.
2. **Jalankan stack (1 menit):** terminal: `docker compose up --build -d`, lalu `docker compose ps` dan `docker compose logs -f backend`.
3. **Jalankan publisher (30 detik):** terminal lain: `docker compose --profile publisher up --build -d publisher-cli`; tunjukkan log publish setiap 2 detik.
4. **Buktikan penyimpanan (1 menit):** panggil `curl http://localhost:8080/vehicles/B1234XYZ/location`; opsional query DB: `docker compose exec postgres psql -U fleet -d fleet -c 'SELECT vehicle_id, latitude, longitude, "timestamp" FROM vehicle_loctions ORDER BY id DESC LIMIT 5;'`.
5. **Uji history (1 menit):** hitung `START=$(($(date +%s)-3600)); END=$(date +%s)` lalu `curl "http://localhost:8080/vehicles/B1234XYZ/history?start=$START&end=$END"`.
6. **Buktikan geofence (1 menit):** tunjukkan log `[geofence-worker] received` melalui `docker compose logs --since=5m backend`; buka RabbitMQ UI `http://localhost:15672` dengan `fleet/fleetpass`, lalu tunjukkan exchange dan queue.
7. **Penutup (30 detik):** tunjukkan Postman Collection dan README, sebutkan konfigurasi geofence bisa diubah lewat environment variable.
