package model

import (
	"errors"
	"fmt"

	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/streadway/amqp"
)

// AMQPClient wraps all object required for AMQP usage
type AMQPClient struct {
	connection   *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	clientName   string
	queue        amqp.Queue
}

// GetAMQPClient inits AMQP connection, channel and queue
func GetAMQPClient(uri, exchangeName, queueName, clientName string) (client AMQPClient, err error) {
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	client.clientName = clientName

	client.connection, err = amqp.Dial(uri)
	if err != nil {
		err = fmt.Errorf("unable to connect to amqp: %s", err)
		return
	}

	client.channel, err = client.connection.Channel()
	if err != nil {
		err = fmt.Errorf("unable to open communication channel: %s", err)
		return
	}

	if err = client.channel.ExchangeDeclare(exchangeName, "direct", true, false, false, false, nil); err != nil {
		err = fmt.Errorf("unable to declare exchange `%s`: %s", queueName, err)
		return
	}
	client.exchangeName = exchangeName

	client.queue, err = client.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		err = fmt.Errorf("unable to declare queue `%s`: %s", queueName, err)
		return
	}

	if err = client.channel.QueueBind(client.queue.Name, "", exchangeName, false, nil); err != nil {
		err = fmt.Errorf("unable to bind queue `%s` to `%s`: %s", queueName, exchangeName, err)
		return
	}

	return client, nil
}

// Enabled checks if connection is setup
func (a AMQPClient) Enabled() bool {
	return a.connection != nil
}

// Ping checks if connection is live
func (a AMQPClient) Ping() error {
	if !a.Enabled() || !a.connection.IsClosed() {
		return errors.New("amqp client closed")
	}

	return nil
}

// Vhost returns connection Vhost
func (a AMQPClient) Vhost() string {
	return a.connection.Config.Vhost
}

// Send sends payload to the underlying exchange and queue
func (a AMQPClient) Send(payload amqp.Publishing) error {
	if err := a.channel.Publish(a.exchangeName, "", false, false, payload); err != nil {
		return fmt.Errorf("unable to publish message: %s", err)
	}

	return nil
}

// Listen listen to queue as given client name
func (a AMQPClient) Listen() (<-chan amqp.Delivery, error) {
	messages, err := a.channel.Consume(a.queue.Name, a.clientName, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to consume queue `%s`: %s", a.queue.Name, err)
	}

	return messages, nil
}

// Close closes opened ressources
func (a AMQPClient) Close() {
	if a.channel != nil {
		if err := a.channel.Cancel(a.clientName, false); err != nil {
			logger.Error("unable to cancel consumer: %s", err)
		}

		LoggedCloser(a.channel)
	}

	if a.connection != nil {
		LoggedCloser(a.connection)
	}
}
