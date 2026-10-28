package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fleet-management-poc/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "fleet.events"
const Queue = "geofence_alerts"

type Publisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	workerCh *amqp.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < 30; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err = ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	q, err := ch.QueueDeclare(Queue, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	if err = ch.QueueBind(q.Name, "geofence.entry", Exchange, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	workerCh, err := conn.Channel()
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &Publisher{conn: conn, ch: ch, workerCh: workerCh}, nil
}
func (p *Publisher) Publish(ctx context.Context, e domain.GeofenceEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, Exchange, "geofence.entry", false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Timestamp: time.Now(), Body: b})
}
func (p *Publisher) Consume(ctx context.Context) error {
	msgs, err := p.workerCh.Consume(Queue, "fleet-geofence-worker", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case m, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitmq consumer channel closed")
			}
			fmt.Printf("[geofence-worker] received: %s\n", string(m.Body))
			if err := m.Ack(false); err != nil {
				fmt.Printf("[geofence-worker] ack error: %v\n", err)
			}
		}
	}
}
func (p *Publisher) Close() {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.workerCh != nil {
		_ = p.workerCh.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}
