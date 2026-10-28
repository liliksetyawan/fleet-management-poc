package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"fleet-management-poc/internal/domain"
	paho "github.com/eclipse/paho.mqtt.golang"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func main() {
	vehicle := env("VEHICLE_ID", "B1234XYZ")
	broker := env("MQTT_BROKER", "tcp://localhost:1883")
	lat, _ := strconv.ParseFloat(env("MOCK_LATITUDE", "-6.2088"), 64)
	lon, _ := strconv.ParseFloat(env("MOCK_LONGITUDE", "106.8456"), 64)
	opts := paho.NewClientOptions().AddBroker(broker).SetClientID("fleet-mock-publisher").SetAutoReconnect(true).SetConnectRetry(true)
	client := paho.NewClient(opts)
	if t := client.Connect(); !t.WaitTimeout(30*time.Second) || t.Error() != nil {
		log.Fatalf("mqtt connection failed: %v", t.Error())
	}
	defer client.Disconnect(250)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("publishing mock locations every 2 seconds to /fleet/vehicle/%s/location via %s", vehicle, broker)
	publish := func() {
		l := domain.Location{VehicleID: vehicle, Latitude: lat + (rand.Float64()-0.5)*0.0001, Longitude: lon + (rand.Float64()-0.5)*0.0001, Timestamp: time.Now().Unix()}
		b, _ := json.Marshal(l)
		topic := "/fleet/vehicle/" + vehicle + "/location"
		t := client.Publish(topic, 1, false, b)
		if !t.WaitTimeout(10*time.Second) || t.Error() != nil {
			log.Printf("publish failed: %v", t.Error())
			return
		}
		log.Printf("published %s", b)
	}
	publish()
	for {
		select {
		case <-sig:
			return
		case <-ticker.C:
			publish()
		}
	}
}
