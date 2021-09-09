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

	reconnectListeners []chan bool

	uri        string
	vhost      string
	clientName string

	mutex sync.RWMutex
}

// GetAMQPClient inits AMQP connection, channel and queue
func GetAMQPClient(uri string) (*AMQPClient, error) {
	if len(uri) == 0 {
		return nil, errors.New("URI is required")
	}

	client := &AMQPClient{
		uri: uri,
	}

	connection, channel, err := connect(uri, client.onDisconnect)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to amqp: %s", err)
	}

	client.connection = connection
	client.channel = channel
	client.vhost = connection.Config.Vhost

	logger.WithField("vhost", client.vhost).Info("Connected to AMQP!")

	return client, nil
}

func connect(uri string, onDisconnect func()) (*amqp.Connection, *amqp.Channel, error) {
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

	go func() {
		localAddr := connection.LocalAddr()
		logger.Warn("Listening close notifications %s", localAddr)
		defer logger.Warn("Close notifications are over for %s", localAddr)

		for range connection.NotifyClose(make(chan *amqp.Error)) {
			logger.Warn("Connection closed, trying to reconnect.")
			onDisconnect()
		}

	}()

	return connection, channel, nil
}

// ListenReconnect creates a chan notifier with a boolean when doing reconnection
func (a *AMQPClient) ListenReconnect() <-chan bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	listener := make(chan bool)
	a.reconnectListeners = append(a.reconnectListeners, listener)

	return listener
}

func (a *AMQPClient) closeListeners() {
	for _, listener := range a.reconnectListeners {
		close(listener)
	}
}

func (a *AMQPClient) notifyListeners() {
	for _, listener := range a.reconnectListeners {
		listener <- true
	}
}

// Consumer configures client for consumming from given queue, bind to given exchange
func (a *AMQPClient) Consumer(queueName, topic, exchangeName string, retryDelay time.Duration) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	queue, err := a.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("unable to declare queue: %s", err)
	}

	if err := a.channel.QueueBind(queue.Name, topic, exchangeName, false, nil); err != nil {
		return fmt.Errorf("unable to bind queue `%s` to `%s`: %s", queue.Name, exchangeName, err)
	}

	if retryDelay != 0 {
		err := a.declareExchange(getDelayedExchangeName(exchangeName), "direct", map[string]interface{}{
			"x-dead-letter-exchange": exchangeName,
			"x-message-ttl":          retryDelay.Milliseconds(),
		}, false)
		if err != nil {
			return fmt.Errorf("unable to declare delayed exchange: %s", getDelayedExchangeName(exchangeName))
		}
	}

	a.ensureClientName()

	return nil
}

// Publisher configures client for publishing to given exchange
func (a *AMQPClient) Publisher(exchangeName, exchangeType string, args amqp.Table) error {
	return a.declareExchange(exchangeName, exchangeType, args, true)
}

// Publisher configures client for publishing to given exchange
func (a *AMQPClient) declareExchange(exchangeName, exchangeType string, args amqp.Table, lock bool) error {
	if lock {
		a.mutex.RLock()
		defer a.mutex.RUnlock()
	}

	if err := a.channel.ExchangeDeclare(exchangeName, exchangeType, true, false, false, false, args); err != nil {
		return fmt.Errorf("unable to declare exchange `%s`: %s", exchangeName, err)
	}

	return nil
}

func getDelayedExchangeName(exchangeName string) string {
	return fmt.Sprintf("%s-delay", exchangeName)
}

func (a *AMQPClient) ensureClientName() {
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
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.channel.Publish(exchange, "", false, false, payload)
}

// Listen listens to configured queue
func (a *AMQPClient) Listen(queue string) (<-chan amqp.Delivery, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.ensureClientName()

	messages, err := a.channel.Consume(queue, a.clientName, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to consume queue: %s", err)
	}

	return messages, nil
}

// Ack ack a message with error handling
func (a *AMQPClient) Ack(message amqp.Delivery) {
	a.loggedConfirm(message, true, false)
}

// Reject reject a message with error handling
func (a *AMQPClient) Reject(message amqp.Delivery, requeue bool) {
	a.loggedConfirm(message, false, requeue)
}

func (a *AMQPClient) loggedConfirm(message amqp.Delivery, ack bool, value bool) {
	for {
		var err error

		if ack {
			err = message.Ack(value)
		} else {
			err = message.Reject(value)
		}

		if err == nil {
			return
		}

		if err != amqp.ErrClosed {
			logger.Error("unable to confirm message: %s", err)
			return
		}

		logger.Error("unable to confirm message due to a closed connection")

		logger.Info("Waiting 30 seconds before attempting to confirm message again...")
		time.Sleep(time.Second * 30)

		func() {
			a.mutex.RLock()
			defer a.mutex.RUnlock()

			message.Acknowledger = a.channel
		}()
	}
}

// Close closes opened ressources
func (a *AMQPClient) Close() {
	if err := a.close(false); err != nil {
		logger.Error("unable to close: %s", err)
	}
}

func (a *AMQPClient) close(reconnect bool) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.closeChannel()
	a.closeConnection()

	if !reconnect {
		a.closeListeners()
		return nil
	}

	newConnection, newChannel, err := connect(a.uri, a.onDisconnect)
	if err != nil {
		return fmt.Errorf("unable to reconnect to amqp: %s", err)
	}

	a.connection = newConnection
	a.channel = newChannel
	a.vhost = newConnection.Config.Vhost

	logger.Info("Connection reopened.")

	go a.notifyListeners()

	return nil
}

func (a *AMQPClient) onDisconnect() {
	for {
		if err := a.close(true); err != nil {
			logger.Error("unable to reconnect: %s", err)

			logger.Info("Waiting one minute before attempting to reconnect again...")
			time.Sleep(time.Minute)
		} else {
			return
		}
	}
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

	if a.connection.IsClosed() {
		return
	}

	logger.WithField("vhost", a.Vhost()).Info("Closing AMQP connection")
	LoggedCloser(a.connection)

	a.connection = nil
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
