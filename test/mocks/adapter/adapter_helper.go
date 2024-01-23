package mockedadapter

import "github.com/futurehomeno/cliffhanger/adapter"

func (a *Adapter) WithName(name string, once bool) *Adapter {
	c := a.On("Name").Return(name)

	if once {
		c.Once()
	}

	return a
}

func (a *Adapter) WithAddress(address string, once bool) *Adapter {
	c := a.On("Address").Return(address)

	if once {
		c.Once()
	}

	return a
}

func (a *Adapter) WithThingByAddress(address string, once bool, thing adapter.Thing) *Adapter {
	c := a.On("ThingByAddress", address).Return(thing)

	if once {
		c.Once()
	}

	return a
}

func (a *Adapter) WithThingByTopic(topic string, once bool, thing adapter.Thing) *Adapter {
	c := a.On("ThingByTopic", topic).Return(thing)

	if once {
		c.Once()
	}

	return a
}
