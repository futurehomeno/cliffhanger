package mockedadapter

import (
	"github.com/stretchr/testify/mock"
)

func (t *Thing) WithUpdate(force, once bool, err error) *Thing {
	c := t.On("Update", force, mock.Anything).Return(err)

	if once {
		c.Once()
	}

	return t
}
