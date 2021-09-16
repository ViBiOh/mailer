package metric

import (
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "mailer"
)

var (
	metrics = make(map[string]*prometheus.CounterVec)
)

// Create creates a metrics for the mailer
func Create(prometheusRegisterer prometheus.Registerer, name string) {
	if prometheusRegisterer == nil {
		return
	}

	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: name,
		Name:      "item",
	}, []string{"state"})

	if err := prometheusRegisterer.Register(counter); err != nil {
		logger.Error("unable to register `%s` metric: %s", name, err)
	}

	metrics[name] = counter
}

// Increase increases the given metric for given state
func Increase(name, state string) {
	if gauge, ok := metrics[name]; ok {
		gauge.With(prometheus.Labels{
			"state": state,
		}).Inc()
	}
}
