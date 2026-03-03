package mockeddatabase

func (d *Database) WithGet(once, found bool, err error, args ...any) *Database {
	c := d.On("Get", args...).Return(found, err)

	if once {
		c.Once()
	}

	return d
}

func (d *Database) WithSet(once bool, err error, args ...any) *Database {
	c := d.On("Set", args...).Return(err)

	if once {
		c.Once()
	}

	return d
}
