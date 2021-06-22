package amqphandler

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/streadway/amqp"
)

// App of package
type App interface {
	Start(<-chan struct{})
	Ping() error
	Close()
}

// Config of package
type Config struct {
	url           *string
	queue         *string
	exchange      *string
	retryInterval *string
	maxRetry      *int
}

type app struct {
	mailerApp     mailer.App
	amqpClient    *model.AMQPClient
	retryInterval time.Duration
	maxRetry      int64
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:           flags.New(prefix, "amqp").Name("URL").Default("").Label("Address in the form amqps?://<user>:<password>@<address>:<port>/<vhost>").ToString(fs),
		exchange:      flags.New(prefix, "amqp").Name("Exchange").Default("mailer").Label("Exchange name").ToString(fs),
		queue:         flags.New(prefix, "amqp").Name("Queue").Default("mailer").Label("Queue name").ToString(fs),
		retryInterval: flags.New(prefix, "amqp").Name("RetryInterval").Default("1h").Label("Interval duration when send fails").ToString(fs),
		maxRetry:      flags.New(prefix, "amqp").Name("MaxRetry").Default(3).Label("Max send retries").ToInt(fs),
	}
}

// New creates new App from Config
func New(config Config, mailerApp mailer.App) (App, error) {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return app{}, nil
	}

	retryInterval, err := time.ParseDuration(strings.TrimSpace(*config.retryInterval))
	if err != nil {
		return nil, fmt.Errorf("unable to parse retry duration: %s", err)
	}

	client, err := model.GetAMQPClient(url, strings.TrimSpace(*config.exchange), strings.TrimSpace(*config.queue))
	if err != nil {
		return nil, fmt.Errorf("unable to create amqp client: %s", err)
	}

	return app{
		retryInterval: retryInterval,
		maxRetry:      int64(*config.maxRetry),
		mailerApp:     mailerApp,
		amqpClient:    client,
	}, nil
}

func (a app) Start(done <-chan struct{}) {
	defer a.Close()

	if a.mailerApp == nil || a.amqpClient.Ping() != nil {
		return
	}

	messages, err := a.amqpClient.Listen()
	if err != nil {
		logger.Error("unable to listen on queue: %s", err)
		return
	}

	logger.WithField("queue", a.amqpClient.QueueName()).WithField("vhost", a.amqpClient.Vhost()).Info("Listening as `%s`", a.amqpClient.ClientName())

	go a.startGarbageCollector(done)

	for {
		select {
		case <-done:
			return
		case message := <-messages:
			if err := a.sendEmail(message.Body); err != nil {
				logger.Error("unable to send email: %s", err)
				a.amqpClient.LoggedReject(message, false)
				continue
			}

			a.amqpClient.LoggedAck(message)
		}
	}
}

func (a app) sendEmail(payload []byte) error {
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

func (a app) startGarbageCollector(done <-chan struct{}) {
	logger.Info("Launching garbage collector every %s", a.retryInterval)
	defer logger.Info("Garbage collector cron is off")

	garbageCron := cron.New().Each(a.retryInterval).Now().OnError(func(err error) {
		logger.Error("error while running garbage collector: %s", err)
	})
	defer garbageCron.Shutdown()

	go garbageCron.Start(func(_ context.Context) error {
		return a.garbageCollector(done)
	}, done)

	signals := make(chan os.Signal, 1)
	defer close(signals)

	signal.Notify(signals, syscall.SIGUSR1)
	defer signal.Stop(signals)

	for {
		select {
		case <-done:
			return
		case <-signals:
			garbageCron.Now()
		}
	}
}

func (a app) garbageCollector(done <-chan struct{}) error {
	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for !isDone() {
		garbage, ok, err := a.amqpClient.GetGarbage()
		if err != nil {
			return fmt.Errorf("unable to get garbage message: %s", err)
		}

		if !ok {
			return nil
		}

		count, err := getDeathCount(garbage.Headers)
		if err != nil {
			logger.Error("unable to get count from garbage: %s", err)
			a.amqpClient.LoggedReject(garbage, false)
			continue
		}

		if count > a.maxRetry {
			logger.Error("message was rejected %d times, content was `%s`", a.maxRetry, garbage.Body)
			a.amqpClient.LoggedAck(garbage)
			continue
		}

		logger.Info("Requeuing message from garbage with payload=`%s`", garbage.Body)

		if err := a.amqpClient.Send(model.ConvertDeliveryToPublishing(garbage)); err != nil {
			logger.Error("unable to re-send garbage message: %s", err)
			a.amqpClient.LoggedReject(garbage, true)
			continue
		}

		a.amqpClient.LoggedAck(garbage)
	}

	return nil
}

func getDeathCount(table amqp.Table) (int64, error) {
	rawDeath := table["x-death"]

	death, ok := rawDeath.([]interface{})
	if !ok {
		return 0, fmt.Errorf("`x-death` header in not an array")
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

func (a app) Ping() error {
	return a.amqpClient.Ping()
}

func (a app) Close() {
	a.amqpClient.Close()
}
