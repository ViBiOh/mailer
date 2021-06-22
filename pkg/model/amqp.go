package model

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
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

	mutex sync.RWMutex
}

func createClientName() string {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		logger.Fatal(err)
		return "mailer"
	}

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
func GetAMQPClient(uri, exchangeName, queueName string) (client *AMQPClient, err error) {
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	if len(uri) == 0 {
		err = errors.New("URI is required")
		return
	}

	if len(exchangeName) == 0 {
		err = errors.New("exchange name is required")
		return
	}

	client = &AMQPClient{
		mutex:        sync.RWMutex{},
		exchangeName: exchangeName,
	}

	logger.Info("Dialing AMQP with 10 seconds timeout...")

	client.connection, err = amqp.DialConfig(uri, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
		Dial:      amqp.DefaultDial(10 * time.Second),
	})
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
func (a *AMQPClient) Enabled() bool {
	return a.connection != nil
}

// Ping checks if connection is live
func (a *AMQPClient) Ping() error {
	if !a.Enabled() {
		return errors.New("amqp client disabled")
	}

	if a.connection.IsClosed() {
		return errors.New("amqp client closed")
	}

	return nil
}

// QueueName returns queue name
func (a *AMQPClient) QueueName() string {
	return a.queue.Name
}

// ExchangeName returns exchange name
func (a *AMQPClient) ExchangeName() string {
	return a.exchangeName
}

// ClientName returns client name
func (a *AMQPClient) ClientName() string {
	return a.clientName
}

// Vhost returns connection Vhost
func (a *AMQPClient) Vhost() string {
	if a.connection == nil {
		return ""
	}

	return a.connection.Config.Vhost
}

// Send sends payload to the underlying exchange and queue
func (a *AMQPClient) Send(payload amqp.Publishing) error {
	err := a.handleClosed(func() error {
		a.mutex.RLock()
		defer a.mutex.RUnlock()

		return a.channel.Publish(a.exchangeName, "", false, false, payload)
	})
	if err != nil {
		return fmt.Errorf("unable to publish message: %s", err)
	}

	return nil
}

// Listen listens to queue
func (a *AMQPClient) Listen() (<-chan amqp.Delivery, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	messages, err := a.channel.Consume(a.queue.Name, a.clientName, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to consume queue `%s`: %s", a.queue.Name, err)
	}

	return messages, nil
}

// GetGarbage get a message from the garbage
func (a *AMQPClient) GetGarbage() (amqp.Delivery, bool, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.channel.Get(a.deadLetterQueue.Name, false)
}

// LoggedAck ack a message with error handling
func (a *AMQPClient) LoggedAck(message amqp.Delivery) {
	err := a.handleClosed(func() error {
		a.mutex.RLock()
		defer a.mutex.RUnlock()

		message.Acknowledger = a.channel
		return message.Ack(false)
	})
	if err != nil {
		logger.Error("unable to ack message: %s", err)
	}
}

// LoggedReject reject a message with error handling
func (a *AMQPClient) LoggedReject(message amqp.Delivery, requeue bool) {
	err := a.handleClosed(func() error {
		a.mutex.RLock()
		defer a.mutex.RUnlock()

		message.Acknowledger = a.channel
		return message.Reject(requeue)
	})

	if err != nil {
		logger.Error("unable to reject message: %s", err)
	}
}

// Close closes opened ressources
func (a *AMQPClient) Close() {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	a.closeChannel()

	if a.connection != nil {
		logger.WithField("vhost", a.Vhost()).Info("Closing AMQP connection")
		LoggedCloser(a.connection)
	}
}

func (a *AMQPClient) closeChannel() {
	if a.channel == nil {
		return
	}

	if len(a.queue.Name) != 0 {
		logger.WithField("name", a.clientName).Info("Canceling AMQP channel")
		if err := a.channel.Cancel(a.clientName, false); err != nil {
			logger.WithField("name", a.clientName).Error("unable to cancel consumer: %s", err)
		}
	}

	logger.Info("Closing AMQP channel")
	LoggedCloser(a.channel)
}

func (a *AMQPClient) handleClosed(action func() error) error {
	err := action()
	if err != amqp.ErrClosed {
		return err
	}

	logger.Warn("Channel was closed, closing first and trying to reopen...")
	a.closeChannel()

	newChannel, openErr := a.connection.Channel()
	if openErr != nil {
		return fmt.Errorf("unable to reopen closed channel: %s: %w", openErr, err)
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.channel = newChannel

	return action()
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
