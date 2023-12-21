package cache

import (
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
)

// ReportingStrategy is an interface representing a strategy to determine whether reporting is required or not.
type ReportingStrategy interface {
	// ReportRequired determines if report is required based on input information.
	ReportRequired(hasChanged bool, lastReported time.Time) bool
}

// ReportingStrategyFn is a function adapter that allows to use anonymous functions as reporting strategy.
type ReportingStrategyFn func(hasChanged bool, lastReported time.Time) bool

// ReportRequired determines if report is required based on input information.
func (f ReportingStrategyFn) ReportRequired(hasChanged bool, lastReported time.Time) bool {
	return f(hasChanged, lastReported)
}

// ReportAlways is a reporting strategy in which report is always sent.
func ReportAlways() ReportingStrategy {
	return ReportingStrategyFn(func(_ bool, _ time.Time) bool {
		return true
	})
}

// ReportOnChangeOnly is a reporting strategy in which report is sent only if value changed.
func ReportOnChangeOnly() ReportingStrategy {
	return ReportingStrategyFn(func(hasChanged bool, _ time.Time) bool {
		return hasChanged
	})
}

// ReportAtLeastEvery is a reporting strategy in which report is sent only if value changed or a specific time has passed.
func ReportAtLeastEvery(interval time.Duration) ReportingStrategy {
	return ReportingStrategyFn(func(hasChanged bool, lastReported time.Time) bool {
		if hasChanged {
			return true
		}

		return time.Since(lastReported) > interval
	})
}

// ReportingCache is a service responsible for storing reported values to allow determine if changes occurred.
type ReportingCache interface {
	// ReportRequired returns true if report for a provided key, sub key and value should be sent according to provided strategy.
	ReportRequired(strategy ReportingStrategy, key, subKey string, value interface{}) bool
	// HasChanged returns true if value for a provided key and sub key changed.
	HasChanged(key, subKey string, value interface{}) bool
	// Reported marks value for a provided key and sub key as reported.
	Reported(key, subKey string, value interface{})
}

// NewReportingCache creates new instance of a reporting cache.
func NewReportingCache() ReportingCache {
	return &reportingCache{
		lock:   &sync.RWMutex{},
		values: make(map[string]map[string]*value),
	}
}

// reportingCache is a private implementation of reporting cache service.
type reportingCache struct {
	lock   *sync.RWMutex
	values map[string]map[string]*value
}

// ReportRequired returns true if report for a provided key, sub key and value should be sent according to provided strategy.
func (c *reportingCache) ReportRequired(strategy ReportingStrategy, key, subKey string, val interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	v := c.values[key][subKey]
	if v == nil {
		return true
	}

	return strategy.ReportRequired(v.hasChanged(val), v.reported)
}

// HasChanged returns true if value for a provided key and sub key changed.
func (c *reportingCache) HasChanged(key, subKey string, val interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	v := c.values[key][subKey]
	if v == nil {
		return true
	}

	return v.hasChanged(val)
}

// Reported marks value for a provided key and sub key as reported.
func (c *reportingCache) Reported(key, subKey string, val interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.values[key]
	if !ok {
		c.values[key] = make(map[string]*value)
	}

	v, ok := c.values[key][subKey]
	if !ok {
		v = &value{}
		c.values[key][subKey] = v
	}

	v.reported = time.Now()
	v.value = val
}

// value is an object holding reporting value and time of last report.
type value struct {
	reported time.Time
	value    interface{}
}

// hasChanged returns true if value is different than provided one.
func (v *value) hasChanged(val interface{}) bool {
	return !cmp.Equal(v.value, val)
}
