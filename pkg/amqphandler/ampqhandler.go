package amqphandler

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	amqpclient "github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/metric"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
)

var errNoDeathCount = errors.New("no death count")

// App of package
type App struct {
	amqpClient    *amqpclient.Client
	done          chan struct{}
	queue         string
	delayExchange string
	mailerApp     mailer.App
	maxRetry      int64
	retry         bool
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
func New(config Config, mailerApp mailer.App, prometheusRegisterer prometheus.Registerer) (App, error) {
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
	app.retry = retryInterval != 0

	app.amqpClient, err = amqpclient.New(url, prometheusRegisterer)
	if err != nil {
		return app, fmt.Errorf("unable to create amqp client: %s", err)
	}

	app.queue = strings.TrimSpace(*config.queue)
	app.delayExchange, err = app.amqpClient.Consumer(app.queue, "", strings.TrimSpace(*config.exchange), retryInterval)
	if err != nil {
		app.amqpClient.Close()
		return app, fmt.Errorf("unable to configure consumer amqp: %s", err)
	}

	if err = app.amqpClient.Ping(); err != nil {
		app.amqpClient.Close()
		return app, fmt.Errorf("unable to ping amqp: %s", err)
	}

	metric.Create(prometheusRegisterer, "amqp")

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

	consumerName, messages, err := a.amqpClient.Listen(a.queue)
	if err != nil {
		logger.Error("unable to listen `%s`: %s", a.queue, err)
		return
	}

	go func() {
		<-done
		a.amqpClient.StopListener(consumerName)
	}()

	log := logger.WithField("queue", a.queue).WithField("vhost", a.amqpClient.Vhost())
	log.Info("Listening messages - started")
	defer log.Info("Listening messages - ended")

	for message := range messages {
		if err := a.sendEmail(message.Body); err != nil {
			logger.Error("unable to send email: %s", err)
			a.handleError(message)
		} else {
			a.amqpClient.Ack(message)
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
		metric.Increase("amqp", "rejected")
		logger.Error("message %s was rejected, content was `%s`", sha.New(message.Body), message.Body)
		a.amqpClient.Ack(message)
		return
	}

	count, err := getDeathCount(message.Headers)
	if err != nil {
		if errors.Is(err, errNoDeathCount) {
			a.delayMessage(message)
			return
		}

		metric.Increase("amqp", "lost")
		logger.Error("unable to get death count from message: %s", err)
		a.amqpClient.Reject(message, false)
		return
	}

	if count >= a.maxRetry {
		metric.Increase("amqp", "rejected")
		logger.Error("message %s was rejected %d times, content was `%s`", sha.New(message.Body), a.maxRetry, message.Body)
		a.amqpClient.Ack(message)
		return
	}

	a.delayMessage(message)
}

func (a App) delayMessage(message amqp.Delivery) {
	messageSha := sha.New(message.Body)

	logger.Info("Delaying message `%s`...", messageSha)

	if err := a.amqpClient.Publish(amqpclient.ConvertDeliveryToPublishing(message), a.delayExchange); err != nil {
		logger.Error("unable to delay message `%s`: %s", messageSha, err)
		a.amqpClient.Reject(message, true)
		return
	}

	metric.Increase("amqp", "delayed")

	a.amqpClient.Ack(message)
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
