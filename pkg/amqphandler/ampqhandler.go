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
type App struct {
	amqpClient    *model.AMQPClient
	done          chan struct{}
	mailerApp     mailer.App
	retryInterval time.Duration
	maxRetry      int64
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
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return App{}, nil
	}

	retryInterval, err := time.ParseDuration(strings.TrimSpace(*config.retryInterval))
	if err != nil {
		return App{}, fmt.Errorf("unable to parse retry duration: %s", err)
	}

	client, err := model.GetAMQPClient(url, strings.TrimSpace(*config.exchange), strings.TrimSpace(*config.queue))
	if err != nil {
		return App{}, fmt.Errorf("unable to create amqp client: %s", err)
	}

	return App{
		retryInterval: retryInterval,
		maxRetry:      int64(*config.maxRetry),
		mailerApp:     mailerApp,
		amqpClient:    client,
		done:          make(chan struct{}),
	}, nil
}

// Start amqp handler
func (a App) Start(done <-chan struct{}) {
	if !a.mailerApp.Enabled() || a.amqpClient == nil {
		return
	}

	defer close(a.done)

	if a.amqpClient.Ping() != nil {
		return
	}

	go a.startListener()
	go a.startGarbageCollector()

	<-done
}

func (a App) listen() (<-chan amqp.Delivery, error) {
	messages, err := a.amqpClient.Listen()
	if err != nil {
		return nil, fmt.Errorf("unable to listen on queue: %s", err)
	}

	return messages, nil
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

func (a App) startListener() {
	defer a.Close()

	logger.WithField("queue", a.amqpClient.QueueName()).WithField("vhost", a.amqpClient.Vhost()).Info("Listening as `%s`", a.amqpClient.ClientName())

	messages, err := a.listen()
	if err != nil {
		logger.Error("%s", err)
		return
	}

listener:
	for message := range messages {
		if err := a.sendEmail(message.Body); err != nil {
			logger.Error("unable to send email: %s", err)
			a.amqpClient.LoggedReject(message, false)
			continue
		}

		a.amqpClient.LoggedAck(message)
	}

	select {
	case <-a.done:
		return
	default:
	}

	for {
		if newMessages, err := a.reconnect(); err != nil {
			logger.Error("unable to reconnect: %s", err)

			logger.Info("Waiting one minute before attempting to reconnect again...")
			time.Sleep(time.Minute)
		} else {
			messages = newMessages
			goto listener
		}
	}
}

func (a App) reconnect() (<-chan amqp.Delivery, error) {
	if err := a.amqpClient.Reconnect(); err != nil {
		return nil, fmt.Errorf("unable to reconnect to amqp: %s", err)
	}

	messages, err := a.listen()
	if err != nil {
		return nil, fmt.Errorf("unable to reopen listener: %s", err)
	}

	return messages, nil
}

func (a App) startGarbageCollector() {
	logger.Info("Launching garbage collector every %s", a.retryInterval)
	defer logger.Info("Garbage collector cron is off")

	garbageCron := cron.New().Each(a.retryInterval).Now().OnError(func(err error) {
		logger.Error("error while running garbage collector: %s", err)
	})
	defer garbageCron.Shutdown()

	go garbageCron.Start(func(_ context.Context) error {
		return a.garbageCollector(a.done)
	}, a.done)

	signals := make(chan os.Signal, 1)
	defer close(signals)

	signal.Notify(signals, syscall.SIGUSR1)
	defer signal.Stop(signals)

	for {
		select {
		case <-a.done:
			return
		case <-signals:
			garbageCron.Now()
		}
	}
}

func (a App) garbageCollector(done <-chan struct{}) error {
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

// Ping amqp
func (a App) Ping() error {
	return a.amqpClient.Ping()
}

// Close amqp
func (a App) Close() {
	a.amqpClient.Close()
}
