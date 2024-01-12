package mockedadapter

import "github.com/futurehomeno/cliffhanger/adapter"

func (a *Adapter) WithThingByTopic(topic string, once bool, thing adapter.Thing) *Adapter {
	c := a.On("ThingByTopic", topic).Return(thing)

	if once {
		c.Once()
	}

	return a
}
