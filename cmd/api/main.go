package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"fleet-management-poc/internal/messaging/mqtt"
	"fleet-management-poc/internal/messaging/rabbitmq"
	"fleet-management-poc/internal/repository/postgres"
	"fleet-management-poc/internal/service"
	httptransport "fleet-management-poc/internal/transport/http"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func envFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, e := strconv.ParseFloat(v, 64)
	if e != nil {
		log.Fatalf("invalid %s: %v", key, e)
	}
	return n
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dsn := env("DATABASE_URL", "postgres://fleet:fleetpass@localhost:5432/fleet?sslmode=disable")
	db, err := postgres.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("postgres connection failed: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err = initSchema(ctx, db); err != nil {
		log.Fatalf("schema init failed: %v", err)
	}

	repo := postgres.NewLocationRepository(db)

	rabbit, err := rabbitmq.NewPublisher(env("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"))
	if err != nil {
		log.Fatalf("rabbitmq connection failed: %v", err)
	}
	defer rabbit.Close()

	svc := service.NewFleetService(repo, rabbit, envFloat("GEOFENCE_LAT", -6.2088), envFloat("GEOFENCE_LON", 106.8456), envFloat("GEOFENCE_RADIUS_METERS", 50))
	go func() {
		if err := rabbit.Consume(ctx); err != nil && ctx.Err() == nil {
			log.Printf("geofence worker stopped: %v", err)
		}
	}()

	go func() {
		if err := mqtt.Subscribe(ctx, env("MQTT_BROKER", "tcp://localhost:1883"), env("MQTT_CLIENT_ID", "fleet-backend"), svc); err != nil && ctx.Err() == nil {
			log.Printf("mqtt subscriber stopped: %v", err)
			cancel()
		}
	}()

	server := &http.Server{Addr: ":8080", Handler: httptransport.New(repo).Router(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("fleet API listening on %s; geofence center=(%f,%f), radius=%fm", server.Addr, envFloat("GEOFENCE_LAT", -6.2088), envFloat("GEOFENCE_LON", 106.8456), envFloat("GEOFENCE_RADIUS_METERS", 50))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server failed: %v", err)
	}
}
func initSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS vehicle_loctions (
		    id BIGSERIAL PRIMARY KEY, 
		    vehicle_id VARCHAR(64) NOT NULL, 
		    latitude DOUBLE PRECISION NOT NULL CHECK (latitude >= -90 AND latitude <= 90), 
		    longitude DOUBLE PRECISION NOT NULL CHECK (longitude >= -180 AND longitude <= 180), 
		    "timestamp" BIGINT NOT NULL CHECK ("timestamp" > 0), 
		    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_vehicle_loctions_vehicle_timestamp ON vehicle_loctions(vehicle_id, "timestamp" DESC)`)
	return err
}
