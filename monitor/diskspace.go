package monitor

import (
	"errors"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/utils"
)

// DiskSpace represents a disk space monitor.
type DiskSpace interface {
	root.Service

	DiskFull() bool
}

type diskSpace struct {
	interval     time.Duration
	limitPercent float64
	used         float64

	closeCh   chan struct{}
	lock      *sync.Mutex
	dataLock  *sync.RWMutex
	waitGroup *sync.WaitGroup
}

// NewDiskSpace creates a new instance of disk space monitor.
// Limit percent value must be between 0 and 100.
func NewDiskSpace(interval time.Duration, limitPercent float64) DiskSpace {
	return &diskSpace{
		interval:     interval,
		limitPercent: limitPercent,
		lock:         &sync.Mutex{},
		dataLock:     &sync.RWMutex{},
		waitGroup:    &sync.WaitGroup{},
	}
}

// DiskFull returns true if the disk space is on limit.
func (d *diskSpace) DiskFull() bool {
	d.dataLock.RLock()
	used := d.used
	d.dataLock.RUnlock()

	if used == 0 {
		d.checkSpace()
	}

	d.dataLock.RLock()
	defer d.dataLock.RUnlock()

	return d.used >= d.limitPercent
}

// Start starts the disk space monitor.
func (d *diskSpace) Start() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.closeCh != nil {
		return errors.New("disk space monitor: already running")
	}

	d.closeCh = make(chan struct{})
	d.waitGroup.Add(1)

	go d.run()

	return nil
}

// Stop gracefully stops the disk space monitor.
func (d *diskSpace) Stop() error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.closeCh == nil {
		return errors.New("disk space monitor: already stopped")
	}

	close(d.closeCh)
	d.waitGroup.Wait()

	d.closeCh = nil

	return nil
}

func (d *diskSpace) run() {
	defer d.waitGroup.Done()
	defer utils.PrintStackOnRecover(false, "run")

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.closeCh:
			return
		case <-ticker.C:
			d.checkSpace()
		}
	}
}

func (d *diskSpace) checkSpace() {
	usage, err := disk.Usage("/")
	if err != nil {
		log.WithError(err).Error("disk space monitor: failed to get disk usage")

		return
	}

	d.dataLock.Lock()
	defer d.dataLock.Unlock()

	d.used = usage.UsedPercent
}
