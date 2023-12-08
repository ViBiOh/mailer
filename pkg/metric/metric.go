package metric

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var metrics = make(map[string]metric.Int64Counter)

func Create(meterProvider metric.MeterProvider, name string) {
	if meterProvider == nil {
		return
	}

	meter := meterProvider.Meter("github.com/ViBiOh/mailer/pkg/metric")

	counter, err := meter.Int64Counter(name)
	if err != nil {
		slog.Error("create metric", "error", err, "name", name)
	}

	metrics[name] = counter
}

func Increase(ctx context.Context, name, state string) {
	if gauge, ok := metrics[name]; ok {
		gauge.Add(ctx, 1, metric.WithAttributes(
			attribute.String("state", state),
		))
	}
}
