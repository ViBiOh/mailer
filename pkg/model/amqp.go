package model

import (
	"fmt"

	"github.com/streadway/amqp"
)

const (
	// ExchangeName is the exchange name
	ExchangeName = "mailer"
	// ExchangeType is the echange type
	ExchangeType = "direct"
)

// InitAMQP inits AMQP connection, channel and queue
func InitAMQP(uri string) (*amqp.Connection, *amqp.Channel, amqp.Queue, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, nil, amqp.Queue{}, fmt.Errorf("unable to connect to amqp: %s", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return conn, nil, amqp.Queue{}, fmt.Errorf("unable to open communication channel: %s", err)
	}

	queue, err := channel.QueueDeclare(ExchangeName, true, false, false, false, nil)
	if err != nil {
		return conn, channel, amqp.Queue{}, fmt.Errorf("unable to declare queue: %s", err)
	}

	return conn, channel, queue, nil
}
