package main

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/DataDog/datadog-process-agent/config"
	"github.com/DataDog/datadog-process-agent/model"
	"github.com/stretchr/testify/assert"
)

func TestUpdateRTStatus(t *testing.T) {
	assert := assert.New(t)
	cfg := config.NewDefaultAgentConfig()
	c, err := NewCollector(cfg)
	assert.NoError(err)
	// XXX: Give the collector a big channel so it never blocks.
	c.rtIntervalCh = make(chan time.Duration, 1000)

	// Validate that we switch to real-time if only one response says so.
	statuses := []*model.CollectorStatus{
		{ActiveClients: 0, Interval: 2},
		{ActiveClients: 3, Interval: 2},
		{ActiveClients: 0, Interval: 2},
	}
	c.updateStatus(statuses)
	assert.Equal(int64(1), atomic.LoadInt64(&c.realTimeEnabled))

	// Validate that we stay that way
	statuses = []*model.CollectorStatus{
		{ActiveClients: 0, Interval: 2},
		{ActiveClients: 3, Interval: 2},
		{ActiveClients: 0, Interval: 2},
	}
	c.updateStatus(statuses)
	assert.Equal(int64(1), atomic.LoadInt64(&c.realTimeEnabled))

	// And that it can turn back off
	statuses = []*model.CollectorStatus{
		{ActiveClients: 0, Interval: 2},
		{ActiveClients: 0, Interval: 2},
		{ActiveClients: 0, Interval: 2},
	}
	c.updateStatus(statuses)
	assert.Equal(int64(0), atomic.LoadInt64(&c.realTimeEnabled))
}

func TestUpdateRTInterval(t *testing.T) {
	assert := assert.New(t)
	cfg := config.NewDefaultAgentConfig()
	c, err := NewCollector(cfg)
	assert.NoError(err)
	// XXX: Give the collector a big channel so it never blocks.
	c.rtIntervalCh = make(chan time.Duration, 1000)

	// Validate that we pick the largest interval.
	statuses := []*model.CollectorStatus{
		{ActiveClients: 0, Interval: 3},
		{ActiveClients: 3, Interval: 2},
		{ActiveClients: 0, Interval: 10},
	}
	c.updateStatus(statuses)
	assert.Equal(int64(1), atomic.LoadInt64(&c.realTimeEnabled))
	assert.Equal(10*time.Second, c.realTimeInterval)
}
