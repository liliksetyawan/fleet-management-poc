-- The table name follows the technical specification exactly (vehicle_loctions).
CREATE TABLE IF NOT EXISTS vehicle_loctions (
  id BIGSERIAL PRIMARY KEY,
  vehicle_id VARCHAR(64) NOT NULL,
  latitude DOUBLE PRECISION NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
  longitude DOUBLE PRECISION NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
  "timestamp" BIGINT NOT NULL CHECK ("timestamp" > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vehicle_loctions_vehicle_timestamp ON vehicle_loctions(vehicle_id, "timestamp" DESC);
