package cache

import (
	"math"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// ReportingStrategy is an interface representing a strategy to determine whether reporting is required or not.
type ReportingStrategy interface {
	// ReportRequired determines if report is required based on input information.
	ReportRequired(prevVal, newVal any, lastReported time.Time) bool
}

// ReportingStrategyFn is a function adapter that allows to use anonymous functions as reporting strategy.
type ReportingStrategyFn func(prevVal, newVal any, lastReported time.Time) bool

// ReportRequired determines if report is required based on input information.
func (f ReportingStrategyFn) ReportRequired(prevVal, newVal any, lastReported time.Time) bool {
	return f(prevVal, newVal, lastReported)
}

// ReportAlways is a reporting strategy in which report is always sent.
func ReportAlways() ReportingStrategy {
	return ReportingStrategyFn(func(_, _ any, _ time.Time) bool {
		return true
	})
}

// ReportOnChangeOnly is a reporting strategy in which report is sent only if value changed.
func ReportOnChangeOnly() ReportingStrategy {
	return ReportingStrategyFn(func(prevVal, newVal any, _ time.Time) bool {
		return !reflect.DeepEqual(prevVal, newVal)
	})
}

// ReportAtLeastEvery is a reporting strategy in which report is sent only if value changed or a specific time has passed.
func ReportAtLeastEvery(interval time.Duration) ReportingStrategy {
	return ReportingStrategyFn(func(prevVal, newVal any, lastReported time.Time) bool {
		if !reflect.DeepEqual(prevVal, newVal) {
			return true
		}

		return time.Since(lastReported) > interval
	})
}

func ReportMinimalChangeAtLeastEvery(minimalChange float64, interval, lastReported time.Duration) ReportingStrategy {
	return ReportingStrategyFn(func(prevVal, newVal any, lastReported time.Time) bool {
		if time.Since(lastReported) > interval {
			return true
		}

		switch pv := prevVal.(type) {
		case int:
			nv, ok := newVal.(int)
			return !ok || math.Abs(float64(pv)-float64(nv)) > minimalChange
		case int64:
			nv, ok := newVal.(int64)
			return !ok || math.Abs(float64(pv)-float64(nv)) > minimalChange
		case uint:
			nv, ok := newVal.(uint)
			return !ok || math.Abs(float64(pv)-float64(nv)) > minimalChange
		case float64:
			nv, ok := newVal.(float64)
			return !ok || math.Abs(pv-nv) > minimalChange
		default:
			log.Infof("Unsupported type %T", prevVal)
		}

		return true
	})
}

// ReportingCache is a service responsible for storing reported values to allow determine if changes occurred.
type ReportingCache interface {
	// ReportRequired returns true if report for a provided key, sub key and value should be sent according to provided strategy.
	ReportRequired(strategy ReportingStrategy, key, subKey string, value any) bool
	// HasChanged returns true if value for a provided key and sub key changed.
	HasChanged(key, subKey string, value any) bool
	// Reported marks value for a provided key and sub key as reported.
	Reported(key, subKey string, value any)
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
func (c *reportingCache) ReportRequired(strategy ReportingStrategy, key, subKey string, val any) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	prevVal := c.values[key][subKey]
	if prevVal == nil {
		return true
	}

	return strategy.ReportRequired(prevVal, val, v.reported)
}

// HasChanged returns true if value for a provided key and sub key changed.
func (c *reportingCache) HasChanged(key, subKey string, val any) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	v := c.values[key][subKey]
	if v == nil {
		return true
	}

	return v.hasChanged(val)
}

// Reported marks value for a provided key and sub key as reported.
func (c *reportingCache) Reported(key, subKey string, val any) {
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
	value    any
}

// hasChanged returns true if value is different than provided one.
func (v *value) hasChanged(val any) bool {
	return !reflect.DeepEqual(v.value, val)
}
