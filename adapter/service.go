package adapter

import (
	"github.com/futurehomeno/fimpgo/fimptype"
)

type Service interface {
	Name() string
	Topic() string
	Specification() *fimptype.Service
}

func NewService(inclusionReport *fimptype.Service) Service {
	return &service{
		inclusionReport: inclusionReport,
	}
}

type service struct {
	inclusionReport *fimptype.Service
}

func (s *service) Name() string {
	return s.inclusionReport.Name
}

func (s *service) Topic() string {
	return s.inclusionReport.Address
}

func (s *service) Specification() *fimptype.Service {
	return s.inclusionReport
}
