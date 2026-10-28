package mqtt

import (
	"context"
	"encoding/json"
	"fleet-management-poc/internal/domain"
	"fleet-management-poc/internal/service"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"strings"
	"time"
)

func Subscribe(ctx context.Context, broker, clientID string, svc *service.FleetService) error {
	opts := paho.NewClientOptions().AddBroker(broker).SetClientID(clientID).SetAutoReconnect(true).SetConnectRetry(true).SetConnectRetryInterval(2 * time.Second)
	client := paho.NewClient(opts)
	connectToken := client.Connect()
	if !connectToken.WaitTimeout(30 * time.Second) {
		return fmt.Errorf("mqtt connect timeout")
	}

	if connectToken.Error() != nil {
		return connectToken.Error()
	}

	token := client.Subscribe("/fleet/vehicle/+/location", 1, func(_ paho.Client, m paho.Message) {
		var l domain.Location
		if err := json.Unmarshal(m.Payload(), &l); err != nil {
			fmt.Printf("[mqtt] invalid JSON on %s: %v\n", m.Topic(), err)
			return
		}
		parts := strings.Split(m.Topic(), "/")
		if len(parts) == 5 && l.VehicleID != parts[3] {
			fmt.Printf("[mqtt] vehicle_id/topic mismatch: %s\n", m.Topic())
			return
		}
		if err := svc.Process(ctx, l); err != nil {
			fmt.Printf("[mqtt] processing message failed: %v\n", err)
		} else {
			fmt.Printf("[mqtt] stored location vehicle=%s lat=%f lon=%f timestamp=%d\n", l.VehicleID, l.Latitude, l.Longitude, l.Timestamp)
		}
	})

	if !token.WaitTimeout(15 * time.Second) {
		client.Disconnect(250)
		return fmt.Errorf("mqtt subscribe timeout")
	}
	if token.Error() != nil {
		client.Disconnect(250)
		return fmt.Errorf("mqtt subscribe: %w", token.Error())
	}
	<-ctx.Done()
	client.Disconnect(250)

	return nil
}
