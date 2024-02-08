package mockedadapter

func (s *Service) WithTopic(once bool, topic string) *Service {
	c := s.On("Topic").Return(topic)

	if once {
		c.Once()
	}

	return s
}
