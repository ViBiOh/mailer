package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/streadway/amqp"
)

// AMQPClient wraps all object required for AMQP usage
type AMQPClient struct {
	connection *amqp.Connection
	channel    *amqp.Channel

	uri        string
	vhost      string
	clientName string

	mutex sync.RWMutex
}

// GetAMQPClient inits AMQP connection, channel and queue
func GetAMQPClient(uri string) (client *AMQPClient, err error) {
	if len(uri) == 0 {
		return nil, errors.New("URI is required")
	}

	connection, channel, err := connect(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to amqp: %s", err)
	}

	return &AMQPClient{
		mutex:      sync.RWMutex{},
		uri:        uri,
		connection: connection,
		channel:    channel,
		vhost:      connection.Config.Vhost,
	}, nil
}

func connect(uri string) (*amqp.Connection, *amqp.Channel, error) {
	logger.Info("Dialing AMQP with 10 seconds timeout...")

	connection, err := amqp.DialConfig(uri, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
		Dial:      amqp.DefaultDial(10 * time.Second),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect to amqp: %s", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		err := fmt.Errorf("unable to open communication channel: %s", err)

		if closeErr := connection.Close(); closeErr != nil {
			err = fmt.Errorf("%s: %w", err, closeErr)
		}

		return nil, nil, err
	}

	if err := channel.Qos(1, 0, false); err != nil {
		err := fmt.Errorf("unable to configure QoS on channel: %s", err)

		if closeErr := channel.Close(); closeErr != nil {
			err = fmt.Errorf("%s: %w", err, closeErr)
		}

		if closeErr := connection.Close(); closeErr != nil {
			err = fmt.Errorf("%s: %w", err, closeErr)
		}

		return nil, nil, err
	}

	return connection, channel, nil
}

// Publisher configures client for publishing to given exchange
func (a *AMQPClient) Publisher(exchangeName, exchangeType string, args amqp.Table) error {
	if err := a.channel.ExchangeDeclare(exchangeName, exchangeType, true, false, false, false, args); err != nil {
		return fmt.Errorf("unable to declare exchange `%s`: %s", exchangeName, err)
	}

	return nil
}

// Consumer configures client for consumming from given queue, bind to given exchange
func (a *AMQPClient) Consumer(queueName, topic, exchangeName string, retryDelay time.Duration) error {
	queue, err := a.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("unable to declare queue: %s", err)
	}

	if err := a.channel.QueueBind(queue.Name, topic, exchangeName, false, nil); err != nil {
		return fmt.Errorf("unable to bind queue `%s` to `%s`: %s", queue.Name, exchangeName, err)
	}

	if retryDelay != 0 {
		err := a.Publisher(getDelayedExchangeName(exchangeName), "direct", map[string]interface{}{
			"x-dead-letter-exchange": exchangeName,
			"x-message-ttl":          retryDelay.Milliseconds(),
		})
		if err != nil {
			return fmt.Errorf("unable to declare delayed exchange: %s", getDelayedExchangeName(exchangeName))
		}
	}

	a.ensureClientName()

	return nil
}

func getDelayedExchangeName(exchangeName string) string {
	return fmt.Sprintf("%s-delay", exchangeName)
}

func (a *AMQPClient) ensureClientName() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if len(a.clientName) != 0 {
		return
	}

	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		logger.Fatal(err)
		a.clientName = "mailer"
	}

	a.clientName = fmt.Sprintf("%x", raw)
}

// Enabled checks if connection is setup
func (a *AMQPClient) Enabled() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.connection != nil
}

// Ping checks if connection is live
func (a *AMQPClient) Ping() error {
	if !a.Enabled() {
		return errors.New("amqp client disabled")
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.connection.IsClosed() {
		return errors.New("amqp client closed")
	}

	return nil
}

// ClientName returns client name
func (a *AMQPClient) ClientName() string {
	return a.clientName
}

// Vhost returns connection Vhost
func (a *AMQPClient) Vhost() string {
	return a.vhost
}

// Publish sends payload to the underlying exchange
func (a *AMQPClient) Publish(payload amqp.Publishing, exchange string) error {
	err := a.handleClosed(func() error {
		a.mutex.RLock()
		defer a.mutex.RUnlock()

		return a.channel.Publish(exchange, "", false, false, payload)
	})
	if err != nil {
		return fmt.Errorf("unable to publish message: %s", err)
	}

	return nil
}

// Listen listens to configured queue
func (a *AMQPClient) Listen(queue string) (<-chan amqp.Delivery, error) {
	a.ensureClientName()

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	messages, err := a.channel.Consume(queue, a.clientName, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to consume queue: %s", err)
	}

	return messages, nil
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
	if err := a.close(false); err != nil {
		logger.Error("unable to close: %s", err)
	}
}

// Reconnect to amqp
func (a *AMQPClient) Reconnect() error {
	if err := a.close(true); err != nil {
		return fmt.Errorf("unable to reconnect: %s", err)
	}
	return nil
}

func (a *AMQPClient) close(reconnect bool) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.closeChannel()
	a.closeConnection()

	if !reconnect {
		return nil
	}

	newConnection, newChannel, err := connect(a.uri)
	if err != nil {
		return fmt.Errorf("unable to reconnect to amqp: %s", err)
	}

	a.connection = newConnection
	a.channel = newChannel
	a.vhost = newConnection.Config.Vhost

	logger.Info("Connection reopened.")

	return nil
}

// StopChannel cancel existing channel
func (a *AMQPClient) StopChannel() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.channel == nil {
		return
	}

	a.closeChannel()
}

func (a *AMQPClient) closeChannel() {
	if a.channel == nil {
		return
	}

	if len(a.clientName) != 0 {
		log := logger.WithField("name", a.clientName)

		log.Info("Canceling AMQP channel")
		if err := a.channel.Cancel(a.clientName, false); err != nil {
			log.Error("unable to cancel consumer: %s", err)
		}
	}

	logger.Info("Closing AMQP channel")
	LoggedCloser(a.channel)

	a.channel = nil
}

func (a *AMQPClient) closeConnection() {
	if a.connection == nil {
		return
	}

	logger.WithField("vhost", a.Vhost()).Info("Closing AMQP connection")
	LoggedCloser(a.connection)

	a.connection = nil
}

func (a *AMQPClient) handleClosed(action func() error) error {
	err := action()
	if err != amqp.ErrClosed {
		return err
	}

	logger.Warn("Channel was closed, trying to reconnect...")
	if err := a.Reconnect(); err != nil {
		logger.Error("unable to reconnect: %s", err)
	}

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
