package amqphandler

import (
	"context"
	"encoding/json"
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
)

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
	queue         *string
	exchange      *string
	retryInterval *string
	maxRetry      *int
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		exchange:      flags.New(prefix, "amqp", "Exchange").Default("mailer", nil).Label("Exchange name").ToString(fs),
		queue:         flags.New(prefix, "amqp", "Queue").Default("mailer", nil).Label("Queue name").ToString(fs),
		retryInterval: flags.New(prefix, "amqp", "RetryInterval").Default("1h", nil).Label("Interval duration when send fails").ToString(fs),
		maxRetry:      flags.New(prefix, "amqp", "MaxRetry").Default(3, nil).Label("Max send retries").ToInt(fs),
	}
}

// New creates new App from Config
func New(config Config, mailerApp mailer.App, amqpClient *amqpclient.Client, prometheusRegisterer prometheus.Registerer) (App, error) {
	app := App{
		done:       make(chan struct{}),
		mailerApp:  mailerApp,
		maxRetry:   int64(*config.maxRetry),
		amqpClient: amqpClient,
	}

	retryInterval, err := time.ParseDuration(*config.retryInterval)
	if err != nil {
		return app, fmt.Errorf("unable to parse retry duration: %s", err)
	}
	app.retry = retryInterval != 0

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

			if status, err := a.amqpClient.Retry(message, a.maxRetry, a.delayExchange); err != nil {
				logger.Error("unable to retry message: %s", err)
			} else {
				messageSha := sha.New(message.Body)
				switch status {
				case amqpclient.DeliveryRejected:
					metric.Increase("amqp", "rejected")
					logger.Error("message %s was rejected, content was `%s`", messageSha, message.Body)
				case amqpclient.DeliveryDelayed:
					metric.Increase("amqp", "delayed")
					logger.Info("Delaying message `%s`...", messageSha)
				}
			}
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
