package app

// CheckNow fires a pending recheck immediately, or performs a regular check if none is pending.
// It lets tests drive consecutive probes deterministically in place of the recheck timer.
func (c *ConnectivityChecker) CheckNow() error {
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()

	return c.Check()
}
