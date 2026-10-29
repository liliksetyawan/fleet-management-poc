# Fleet Management POC — Transjakarta Backend Engineer

POC implementasi requirement pada dokumen tes teknis: Go + Gin, MQTT Mosquitto, PostgreSQL, RabbitMQ, geofence radius 50 meter, Docker Compose, dan mock MQTT publisher setiap 2 detik. Struktur aplikasi memisahkan domain, service, repository, messaging, dan HTTP transport (Clean Architecture sederhana).

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

## Postman

Import `postman/Fleet Management POC.postman_collection.json` ke Postman. Collection menyediakan health, latest location, dan history request.
