package mqtt

import (
	"time"

	"github.com/grafana/sobek"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/metrics"
)

// Publish allow to publish one message
//
//nolint:gocognit
func (c *client) Publish(
	topic string,
	qos int,
	message string,
	retain bool,
	timeout uint,
	success func(sobek.Value) (sobek.Value, error),
	failure func(sobek.Value) (sobek.Value, error),
) error {
	// sync case no callback added
	if success == nil && failure == nil {
		return c.publishSync(topic, qos, message, retain, timeout)
	}

	// check timeout value
	timeoutValue, err := safeUintToInt64(timeout)
	if err != nil {
		rt := c.vu.Runtime()
		common.Throw(rt, ErrTimeoutToLong)
		return ErrTimeoutToLong
	}

	// async case
	callback := c.vu.RegisterCallback()
	go func() {
		if c.pahoClient == nil || !c.pahoClient.IsConnected() {
			callback(func() error {
				ev := c.newErrorEvent("publish not connected")
				if failure != nil {
					if _, err := failure(ev); err != nil {
						return err
					}
				}
				return nil
			})
			return
		}
		token := c.pahoClient.Publish(topic, byte(qos), retain, message)
		if !token.WaitTimeout(time.Duration(timeoutValue) * time.Millisecond) {
			callback(func() error {
				ev := c.newErrorEvent("publish timeout")

				if failure != nil {
					if _, err := failure(ev); err != nil {
						return err
					}
				}
				return nil
			})
			return
		}
		if err := token.Error(); err != nil {
			callback(func() error {
				ev := c.newErrorEvent(err.Error())
				if failure != nil {
					if _, err := failure(ev); err != nil {
						return err
					}
				}
				return nil
			})
			return
		}
		callback(func() error {
			err := c.publishMessageMetric(float64(len(message)))
			if err != nil {
				return err
			}
			ev := c.newPublishEvent(topic)
			if success != nil {
				if _, err := success(ev); err != nil {
					return err
				}
			}
			return nil
		})
	}()
	return nil
}

func (c *client) publishSync(
	topic string,
	qos int,
	message string,
	retain bool,
	timeout uint,
) error {
	if c.pahoClient == nil || !c.pahoClient.IsConnected() {
		rt := c.vu.Runtime()
		common.Throw(rt, ErrClient)
		return ErrClient
	}

	// check timeout value
	timeoutValue, err := safeUintToInt64(timeout)
	if err != nil {
		rt := c.vu.Runtime()
		common.Throw(rt, ErrTimeoutToLong)
		return ErrTimeoutToLong
	}

	token := c.pahoClient.Publish(topic, byte(qos), retain, message)
	// sync case
	if !token.WaitTimeout(time.Duration(timeoutValue) * time.Millisecond) {
		rt := c.vu.Runtime()
		common.Throw(rt, ErrTimeout)
		return ErrTimeout
	}

	if err = token.Error(); err != nil {
		rt := c.vu.Runtime()
		common.Throw(rt, ErrPublish)
		return ErrPublish
	}

	err = c.publishMessageMetric(float64(len(message)))
	if err != nil {
		return err
	}
	return nil
}

func (c *client) publishMessageMetric(msgLen float64) error {
	// publish metrics
	now := time.Now()
	state := c.vu.State()
	if state == nil {
		return ErrState
	}

	ctx := c.vu.Context()
	if ctx == nil {
		return ErrState
	}
	metrics.PushIfNotDone(ctx, state.Samples, metrics.Sample{
		TimeSeries: metrics.TimeSeries{Metric: c.metrics.SentMessages, Tags: c.metrics.TagsAndMeta.Tags},
		Time:       now,
		Value:      float64(1),
	})
	metrics.PushIfNotDone(ctx, state.Samples, metrics.Sample{
		TimeSeries: metrics.TimeSeries{Metric: c.metrics.SentBytes, Tags: c.metrics.TagsAndMeta.Tags},
		Time:       now,
		Value:      msgLen,
	})
	metrics.PushIfNotDone(ctx, state.Samples, metrics.Sample{
		TimeSeries: metrics.TimeSeries{Metric: c.metrics.SentDates, Tags: c.metrics.TagsAndMeta.Tags},
		Time:       now,
		Value:      float64(now.UnixMilli()),
	})
	return nil
}

//nolint:nosnakecase // their choice not mine
func (c *client) newPublishEvent(topic string) *sobek.Object {
	rt := c.vu.Runtime()
	o := rt.NewObject()
	must := func(err error) {
		if err != nil {
			common.Throw(rt, err)
		}
	}

	must(o.DefineDataProperty("type", rt.ToValue("publish"), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	must(o.DefineDataProperty("topic", rt.ToValue(topic), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	return o
}
