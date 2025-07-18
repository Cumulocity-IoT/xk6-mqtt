package mqtt

import (
	"errors"

	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
)

type mqttMetrics struct {
	SentBytes        *metrics.Metric
	ReceivedBytes    *metrics.Metric
	SentMessages     *metrics.Metric
	ReceivedMessages *metrics.Metric
	SentDates        *metrics.Metric
	TagsAndMeta      *metrics.TagsAndMeta
}

type mqttMetricsLabels struct {
	SentBytesLabel             string
	ReceivedBytesLabel         string
	SentMessagesCountLabel     string
	ReceivedMessagesCountLabel string
	SentDatesLabel             string
}

// registerMetrics registers the metrics for the mqtt module in the metrics registry
func registerMetrics(vu modules.VU, labels mqttMetricsLabels) (mqttMetrics, error) {
	var err error
	m := mqttMetrics{}
	env := vu.InitEnv()
	if env == nil {
		return m, errors.New("missing env")
	}
	registry := env.Registry
	if registry == nil {
		return m, errors.New("missing registry")
	}

	m.SentBytes, err = registry.NewMetric(labels.SentBytesLabel, metrics.Counter)
	if err != nil {
		return m, err
	}

	m.ReceivedBytes, err = registry.NewMetric(labels.ReceivedBytesLabel, metrics.Counter)
	if err != nil {
		return m, err
	}

	m.SentMessages, err = registry.NewMetric(labels.SentMessagesCountLabel, metrics.Counter)
	if err != nil {
		return m, err
	}

	m.ReceivedMessages, err = registry.NewMetric(labels.ReceivedMessagesCountLabel, metrics.Counter)
	if err != nil {
		return m, err
	}
	m.SentDates, err = registry.NewMetric(labels.SentDatesLabel, metrics.Gauge)
	if err != nil {
		return m, err
	}
	m.TagsAndMeta = &metrics.TagsAndMeta{
		Tags: registry.RootTagSet(),
	}
	return m, nil
}
