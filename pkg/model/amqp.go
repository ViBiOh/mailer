package model

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/streadway/amqp"
)

// AMQPClient wraps all object required for AMQP usage
type AMQPClient struct {
	connection *amqp.Connection
	channel    *amqp.Channel

	exchangeName string
	clientName   string

	queue           amqp.Queue
	deadLetterQueue amqp.Queue
}

func createClientName() string {
	raw := make([]byte, 4)
	rand.New(rand.NewSource(time.Now().UnixNano())).Read(raw)
	return fmt.Sprintf("%x", raw)
}

func createExchangeAndQueue(channel *amqp.Channel, exchangeName, queueName string, internal bool, args amqp.Table) (amqp.Queue, error) {
	if err := channel.ExchangeDeclare(exchangeName, "direct", true, false, false, internal, nil); err != nil {
		return amqp.Queue{}, fmt.Errorf("unable to declare exchange `%s`: %s", queueName, err)
	}

	queue, err := channel.QueueDeclare(queueName, true, false, false, false, args)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("unable to declare queue `%s`: %s", queueName, err)
	}

	if err := channel.QueueBind(queue.Name, "", exchangeName, false, nil); err != nil {
		return queue, fmt.Errorf("unable to bind queue `%s` to `%s`: %s", queueName, exchangeName, err)
	}

	return queue, nil
}

// GetAMQPClient inits AMQP connection, channel and queue
func GetAMQPClient(uri, exchangeName, queueName string) (client AMQPClient, err error) {
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	client.exchangeName = exchangeName

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

	if len(queueName) == 0 {
		return client, nil
	}

	client.clientName = createClientName()

	if err = client.channel.Qos(1, 0, false); err != nil {
		err = fmt.Errorf("unable to configure QoS on channel: %s", err)
		return client, nil
	}

	deadLetterExchange := fmt.Sprintf("%s-garbage", exchangeName)
	deadLetterQueue := fmt.Sprintf("%s-garbage", queueName)
	client.deadLetterQueue, err = createExchangeAndQueue(client.channel, deadLetterExchange, deadLetterQueue, true, nil)
	if err != nil {
		err = fmt.Errorf("unable to create dead letter: %s", err)
		return
	}

	client.queue, err = createExchangeAndQueue(client.channel, exchangeName, queueName, false, map[string]interface{}{
		"x-dead-letter-exchange": deadLetterExchange,
	})
	if err != nil {
		err = fmt.Errorf("unable to create queue: %s", err)
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

// QueueName returns queue name
func (a AMQPClient) QueueName() string {
	return a.queue.Name
}

// ExchangeName returns exchange name
func (a AMQPClient) ExchangeName() string {
	return a.exchangeName
}

// Vhost returns connection Vhost
func (a AMQPClient) Vhost() string {
	if a.connection == nil {
		return ""
	}

	return a.connection.Config.Vhost
}

// Send sends payload to the underlying exchange and queue
func (a AMQPClient) Send(payload amqp.Publishing) error {
	if err := a.channel.Confirm(false); err != nil {
		return fmt.Errorf("unable to put channel in confirm mode: %s", err)
	}

	notifyPublish := a.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	if err := a.channel.Publish(a.exchangeName, "", false, false, payload); err != nil {
		return fmt.Errorf("unable to publish message: %s", err)
	}

	timeout := time.NewTicker(time.Second * 15)
	defer timeout.Stop()

	select {
	case <-timeout.C:
		return errors.New("timeout while waiting for delivery confirmation")
	case confirmed := <-notifyPublish:
		if confirmed.Ack {
			logger.Info("Delivery confirmed with tag %d", confirmed.DeliveryTag)
			return nil
		}

		return fmt.Errorf("unable to confirme delivery with tag %d", confirmed.DeliveryTag)
	}
}

// Listen listen to queue
func (a AMQPClient) Listen() (<-chan amqp.Delivery, error) {
	messages, err := a.channel.Consume(a.queue.Name, a.clientName, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to consume queue `%s`: %s", a.queue.Name, err)
	}

	return messages, nil
}

// GetGarbage get a message from the garbage
func (a AMQPClient) GetGarbage() (amqp.Delivery, bool, error) {
	return a.channel.Get(a.deadLetterQueue.Name, false)
}

// Close closes opened ressources
func (a AMQPClient) Close() {
	if a.channel != nil {
		if len(a.queue.Name) != 0 {
			logger.Info("Closing channel for %s", a.clientName)
			if err := a.channel.Cancel(a.clientName, false); err != nil {
				logger.Error("unable to cancel consumer `%s`: %s", a.clientName, err)
			}
		}

		LoggedCloser(a.channel)
	}

	if a.connection != nil {
		logger.Info("Closing connection on %s", a.Vhost())
		LoggedCloser(a.connection)
	}
}

// LoggedAck ack a message with error handling
func LoggedAck(message amqp.Delivery) {
	if err := message.Ack(false); err != nil {
		logger.Error("unable to ack message: %s", err)
	}
}

// LoggedReject reject a message with error handling
func LoggedReject(message amqp.Delivery, requeue bool) {
	if err := message.Reject(requeue); err != nil {
		logger.Error("unable to reject message: %s", err)
	}
}

// ConvertDeliveryToPublishing convert a delivery to a publishing, for requeuing
func ConvertDeliveryToPublishing(message amqp.Delivery) amqp.Publishing {
	return amqp.Publishing{
		Headers:         message.Headers,
		ContentType:     message.ContentType,
		ContentEncoding: message.ContentEncoding,
		DeliveryMode:    message.DeliveryMode,
		Priority:        message.Priority,
		CorrelationId:   message.CorrelationId,
		ReplyTo:         message.ReplyTo,
		Expiration:      message.Expiration,
		MessageId:       message.MessageId,
		Timestamp:       message.Timestamp,
		Type:            message.Type,
		UserId:          message.UserId,
		AppId:           message.AppId,
		Body:            message.Body,
	}
}