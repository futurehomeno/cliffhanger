package cache

import (
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
)

type ReportingStrategy interface {
	reportRequired(hasChanged bool, lastReported time.Time) bool
}

type reportingStrategyFn func(hasChanged bool, lastReported time.Time) bool

func (f reportingStrategyFn) reportRequired(hasChanged bool, lastReported time.Time) bool {
	return f(hasChanged, lastReported)
}

func ReportAlways() ReportingStrategy {
	return reportingStrategyFn(func(_ bool, _ time.Time) bool {
		return true
	})
}

func ReportOnChangeOnly() ReportingStrategy {
	return reportingStrategyFn(func(hasChanged bool, _ time.Time) bool {
		return hasChanged
	})
}

func ReportAtLeastEvery(interval time.Duration) ReportingStrategy {
	return reportingStrategyFn(func(hasChanged bool, lastReported time.Time) bool {
		if hasChanged {
			return true
		}

		return time.Since(lastReported) > interval
	})
}

type ReportingCache interface {
	ReportRequired(strategy ReportingStrategy, key, subKey string, value interface{}) bool
	HasChanged(key, subKey string, value interface{}) bool
	Reported(key, subKey string, value interface{})
}

func NewReportingCache() ReportingCache {
	return &cache{
		values: make(map[string]map[string]*value),
	}
}

type cache struct {
	lock   *sync.RWMutex
	values map[string]map[string]*value
}

func (c *cache) ReportRequired(strategy ReportingStrategy, key, subKey string, val interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	v := c.values[key][subKey]
	if v == nil {
		return true
	}

	return strategy.reportRequired(v.hasChanged(val), v.reported)
}

func (c *cache) HasChanged(key, subKey string, val interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	v := c.values[key][subKey]
	if v == nil {
		return true
	}

	return v.hasChanged(val)
}

func (c *cache) Reported(key, subKey string, val interface{}) {
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

type value struct {
	reported time.Time
	value    interface{}
}

func (v *value) hasChanged(val interface{}) bool {
	return !cmp.Equal(v.value, val)
}
