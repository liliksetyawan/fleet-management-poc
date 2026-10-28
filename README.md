# Fleet Management POC — Transjakarta Backend Engineer

POC implementasi requirement pada dokumen tes teknis: Go + Gin, MQTT Mosquitto, PostgreSQL, RabbitMQ, geofence radius 50 meter, Docker Compose, dan mock MQTT publisher setiap 2 detik. Struktur aplikasi memisahkan domain, service, repository, messaging, dan HTTP transport (Clean Architecture sederhana).

## Arsitektur

1. `cmd/publisher` menerbitkan JSON lokasi ke `/fleet/vehicle/{vehicle_id}/location` setiap 2 detik.
2. Backend subscribe ke `/fleet/vehicle/+/location`, memvalidasi payload dan kecocokan `vehicle_id` dengan topic.
3. Lokasi valid disimpan sebagai baris baru di PostgreSQL, tabel `vehicle_loctions` (ejaan mengikuti dokumen requirement).
4. Service menghitung jarak Haversine ke titik geofence yang dikonfigurasi. Saat kendaraan bertransisi dari luar ke dalam radius 50 meter, event `geofence_entry` dipublish ke exchange `fleet.events` dengan routing key `geofence.entry`.
5. Worker di backend mengonsumsi queue `geofence_alerts` dan menampilkan pesan yang diterima di log.
6. Gin menyediakan API lokasi terakhir dan riwayat.

> Titik geofence tidak ditentukan di dokumen. POC memakai default Jakarta `(-6.2088, 106.8456)` agar publisher mock langsung bisa menguji geofence. Titik dan radius dapat diubah melalui environment variable.

## Menjalankan

Persyaratan: Docker dan Docker Compose v2.

```bash
docker compose up --build -d
# Jalankan mock publisher (mengirim data tiap 2 detik)
docker compose --profile publisher up --build -d publisher-cli

docker compose ps
docker compose logs -f backend publisher-cli
```

API tersedia di `http://localhost:8080`. Health check: `GET /health`.

### API

**Lokasi terakhir**

```bash
curl http://localhost:8080/vehicles/B1234XYZ/location
```

Mengembalikan `vehicle_id`, `latitude`, `longitude`, dan `timestamp`. Jika data belum tersedia, API mengembalikan HTTP 404.

**Riwayat berdasarkan rentang Unix timestamp**

```bash
START=$(($(date +%s)-3600))
END=$(date +%s)
curl "http://localhost:8080/vehicles/B1234XYZ/history?start=${START}&end=${END}"
```

Response history berbentuk `{ "vehicle_id": "...", "start": 0, "end": 0, "count": 1, "data": [...] }`. Rentang inklusif (`start <= timestamp <= end`), data diurutkan naik berdasarkan timestamp. `start` dan `end` wajib berupa Unix timestamp detik, `end >= start`.

### Menguji geofence

Default mock publisher menggunakan koordinat sekitar titik geofence. Lihat worker menerima event:

```bash
docker compose logs -f backend
```

Untuk mengubah titik pusat, atur sebelum menjalankan Compose:

```bash
GEOFENCE_LAT=-6.2088 GEOFENCE_LON=106.8456 GEOFENCE_RADIUS_METERS=50 docker compose up --build -d
```

Event yang dipublish mengikuti format:

```json
{
  "vehicle_id": "B1234XYZ",
  "event": "geofence_entry",
  "location": { "latitude": -6.2088, "longitude": 106.8456 },
  "timestamp": 1715003456
}
```

RabbitMQ Management UI: `http://localhost:15672` (user `fleet`, password `fleetpass`). Exchange `fleet.events`; queue `geofence_alerts`.

### Publish manual (opsional)

Jika ingin mengirim satu payload sendiri, gunakan `mosquitto_pub` bila terpasang:

```bash
mosquitto_pub -h localhost -p 1883 -q 1 \
  -t '/fleet/vehicle/B1234XYZ/location' \
  -m '{"vehicle_id":"B1234XYZ","latitude":-6.2088,"longitude":106.8456,"timestamp":'"$(date +%s)'"'}'
```

Publisher otomatis mengirim payload mock setiap 2 detik dan bisa dikonfigurasi melalui `VEHICLE_ID`, `MOCK_LATITUDE`, `MOCK_LONGITUDE`, dan `MQTT_BROKER`.

## Environment variables

| Variable | Default | Keterangan |
|---|---|---|
| `DATABASE_URL` | `postgres://fleet:fleetpass@postgres:5432/fleet?sslmode=disable` | PostgreSQL DSN |
| `RABBITMQ_URL` | `amqp://fleet:fleetpass@rabbitmq:5672/` | RabbitMQ AMQP URL |
| `MQTT_BROKER` | `tcp://mosquitto:1883` di Compose | Broker MQTT |
| `GEOFENCE_LAT` | `-6.2088` | Latitude titik geofence |
| `GEOFENCE_LON` | `106.8456` | Longitude titik geofence |
| `GEOFENCE_RADIUS_METERS` | `50` | Radius geofence dalam meter |

## Struktur

```text
cmd/api/                 Backend API + MQTT subscriber + RabbitMQ worker startup
cmd/publisher/           Mock MQTT publisher
internal/domain/         Entity dan validasi lokasi
internal/service/        Penyimpanan lokasi dan geofence logic
internal/repository/     PostgreSQL repository
internal/messaging/      MQTT subscriber dan RabbitMQ publisher/worker
internal/transport/http/ REST API handlers
 db/init.sql             Schema PostgreSQL
postman/                 Postman collection
```

## Postman

Import `postman/Fleet Management POC.postman_collection.json` ke Postman. Collection menyediakan health, latest location, dan history request.

## Catatan POC

- Validasi payload memastikan vehicle ID tidak kosong, koordinat berada dalam batas valid, dan timestamp positif.
- Event dikirim saat status kendaraan berubah dari di luar menjadi di dalam geofence; tidak mengirim event berulang untuk setiap pesan selama kendaraan tetap di dalam. Status transisi disimpan di memori proses, sehingga reset proses akan menginisialisasi ulang status.
- MQTT dibuat tanpa autentikasi untuk pengujian lokal. Jangan gunakan konfigurasi tersebut langsung di production.
- Untuk environment production, tambahkan autentikasi, TLS, observability, retry/dead-letter policy, dan state geofence persisten sesuai kebutuhan operasional.
