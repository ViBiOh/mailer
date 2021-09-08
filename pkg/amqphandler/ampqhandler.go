package amqphandler

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/streadway/amqp"
)

var (
	errNoDeathCount = errors.New("no death count")
)

// App of package
type App struct {
	amqpClient *model.AMQPClient
	done       chan struct{}
	queue      string
	exchange   string
	mailerApp  mailer.App
	maxRetry   int64
	retry      bool
}

// Config of package
type Config struct {
	url           *string
	queue         *string
	exchange      *string
	retryInterval *string
	maxRetry      *int
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:           flags.New(prefix, "amqp", "URL").Default("", nil).Label("Address in the form amqps?://<user>:<password>@<address>:<port>/<vhost>").ToString(fs),
		exchange:      flags.New(prefix, "amqp", "Exchange").Default("mailer", nil).Label("Exchange name").ToString(fs),
		queue:         flags.New(prefix, "amqp", "Queue").Default("mailer", nil).Label("Queue name").ToString(fs),
		retryInterval: flags.New(prefix, "amqp", "RetryInterval").Default("1h", nil).Label("Interval duration when send fails").ToString(fs),
		maxRetry:      flags.New(prefix, "amqp", "MaxRetry").Default(3, nil).Label("Max send retries").ToInt(fs),
	}
}

// New creates new App from Config
func New(config Config, mailerApp mailer.App) (App, error) {
	app := App{
		done:      make(chan struct{}),
		mailerApp: mailerApp,
		maxRetry:  int64(*config.maxRetry),
	}

	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return app, nil
	}

	retryInterval, err := time.ParseDuration(*config.retryInterval)
	if err != nil {
		return app, fmt.Errorf("unable to parse retry duration: %s", err)
	}

	client, err := model.GetAMQPClient(url)
	if err != nil {
		return app, fmt.Errorf("unable to create amqp client: %s", err)
	}

	queue := strings.TrimSpace(*config.queue)
	exchange := strings.TrimSpace(*config.exchange)
	if err := client.Consumer(queue, "", exchange, retryInterval); err != nil {
		client.Close()
		return app, fmt.Errorf("unable to configure consumer amqp: %s", err)
	}

	if err := client.Ping(); err != nil {
		client.Close()
		return app, fmt.Errorf("unable to ping amqp: %s", err)
	}

	app.amqpClient = client
	app.queue = queue
	app.exchange = exchange
	app.retry = retryInterval != 0

	return app, nil
}

// Enabled checks if requirements are met
func (a App) Enabled() bool {
	return a.amqpClient != nil
}

// Done returns the chan used for synchronization
func (a App) Done() <-chan struct{} {
	return a.done
}

// Start amqp handler
func (a App) Start(done <-chan struct{}) {
	defer close(a.done)
	defer a.Close()

	if !a.mailerApp.Enabled() || a.amqpClient == nil {
		return
	}

	messages, err := a.listen()
	if err != nil {
		logger.Error("%s", err)
		return
	}

	go func() {
		<-done
		a.amqpClient.StopChannel()
	}()

	a.startListener(done, messages)
}

func (a App) listen() (<-chan amqp.Delivery, error) {
	messages, err := a.amqpClient.Listen(a.queue)
	if err != nil {
		return nil, fmt.Errorf("unable to listen on queue: %s", err)
	}

	logger.WithField("queue", a.queue).WithField("vhost", a.amqpClient.Vhost()).Info("Listening as `%s`", a.amqpClient.ClientName())

	return messages, nil
}

func (a App) startListener(done <-chan struct{}, messages <-chan amqp.Delivery) {
	reconnectListener := a.amqpClient.ListenReconnect()

listener:
	for message := range messages {
		if err := a.sendEmail(message.Body); err != nil {
			logger.Error("unable to send email: %s", err)
			a.handleError(message)
		} else {
			a.amqpClient.LoggedAck(message)
		}
	}

	logger.WithField("queue", a.queue).WithField("vhost", a.amqpClient.Vhost()).Info("Listening stopped")

	select {
	case <-done:
		return
	case _, ok := <-reconnectListener:
		if !ok {
			return
		}
	}

	for {
		if newMessages, err := a.listen(); err != nil {
			logger.Error("unable to reopen listener: %s", err)

			logger.Info("Waiting 30 seconds before attempting to listen again...")
			time.Sleep(time.Second * 30)
		} else {
			messages = newMessages
			goto listener
		}
	}
}

func (a App) sendEmail(payload []byte) error {
	ctx := context.Background()

	var mailRequest model.MailRequest
	if err := json.Unmarshal(payload, &mailRequest); err != nil {
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	output, err := a.mailerApp.Render(ctx, mailRequest)
	if err != nil {
		return fmt.Errorf("unable to render email: %s", err)
	}

	return a.mailerApp.Send(ctx, mailRequest.ConvertToMail(output))
}

func (a App) handleError(message amqp.Delivery) {
	if !a.retry {
		logger.Error("message %s was rejected, content was `%s`", sha.New(message.Body), message.Body)
		a.amqpClient.LoggedAck(message)
		return
	}

	count, err := getDeathCount(message.Headers)
	if err != nil {
		if errors.Is(err, errNoDeathCount) {
			a.delayMessage(message)
			return
		}

		logger.Error("unable to get death count from message: %s", err)
		a.amqpClient.LoggedReject(message, false)
		return
	}

	if count >= a.maxRetry {
		logger.Error("message %s was rejected %d times, content was `%s`", sha.New(message.Body), a.maxRetry, message.Body)
		a.amqpClient.LoggedAck(message)
		return
	}

	a.delayMessage(message)
}

func (a App) delayMessage(message amqp.Delivery) {
	logger.Info("Delaying message treatment for %s...", sha.New(message.Body))

	if err := a.amqpClient.Publish(model.ConvertDeliveryToPublishing(message), a.exchange); err != nil {
		logger.Error("unable to re-send garbage message: %s", err)
		a.amqpClient.LoggedReject(message, true)
	}

	a.amqpClient.LoggedAck(message)
}

func getDeathCount(table amqp.Table) (int64, error) {
	rawDeath := table["x-death"]

	death, ok := rawDeath.([]interface{})
	if !ok {
		return 0, fmt.Errorf("`x-death` header in not an array: %w", errNoDeathCount)
	}

	if len(death) == 0 {
		return 0, fmt.Errorf("`x-death` is an empty array")
	}

	deathData, ok := death[0].(amqp.Table)
	if !ok {
		return 0, fmt.Errorf("`x-death` datas are not a map")
	}

	count, ok := deathData["count"].(int64)
	if !ok {
		return 0, fmt.Errorf("`count` is not an int")
	}

	return count, nil
}

// Ping amqp
func (a App) Ping() error {
	if a.amqpClient == nil {
		return nil
	}

	return a.amqpClient.Ping()
}

// Close amqp
func (a App) Close() {
	if a.amqpClient == nil {
		return
	}

	a.amqpClient.Close()
}
