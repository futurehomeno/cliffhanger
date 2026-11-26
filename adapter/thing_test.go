package adapter_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
)

var errPublish = errors.New("publish failed")

type recordingPublisher struct {
	lock     sync.Mutex
	failNext bool
	messages []*fimpgo.FimpMessage
	events   []adapter.ThingEvent
}

func (p *recordingPublisher) PublishAdapterMessage(m *fimpgo.FimpMessage) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.failNext {
		p.failNext = false

		return errPublish
	}

	p.messages = append(p.messages, m)

	return nil
}

func (p *recordingPublisher) PublishThingMessage(_ adapter.Thing, m *fimpgo.FimpMessage) error {
	return p.PublishAdapterMessage(m)
}

func (p *recordingPublisher) PublishThingEvent(e adapter.ThingEvent) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.events = append(p.events, e)
}

func (p *recordingPublisher) PublishServiceMessage(adapter.Service, *fimpgo.FimpMessage) error {
	return nil
}

func (p *recordingPublisher) PublishServiceEvent(adapter.Service, adapter.ServiceEvent) {}

func (p *recordingPublisher) count(msgType string) int {
	p.lock.Lock()
	defer p.lock.Unlock()

	n := 0

	for _, m := range p.messages {
		if m.Interface == msgType {
			n++
		}
	}

	return n
}

type memThingState struct {
	checksum uint32
	failSet  bool
}

func (s *memThingState) ID() string                { return "A" }
func (s *memThingState) Address() string           { return "1" }
func (s *memThingState) Info(any) error            { return nil }
func (s *memThingState) State(any) error           { return nil }
func (s *memThingState) SetState(any) error        { return nil }
func (s *memThingState) InclusionChecksum() uint32 { return s.checksum }

func (s *memThingState) SetInclusionChecksum(c uint32) error {
	if s.failSet {
		return errors.New("disk full")
	}

	s.checksum = c

	return nil
}

func newTestThing(t *testing.T, p adapter.Publisher, ts adapter.ThingState, strategy cache.ReportingStrategy) adapter.Thing {
	t.Helper()

	return adapter.NewThing(p, ts, &adapter.ThingConfig{
		InclusionReport:               &fimptype.ThingInclusionReport{Address: "1"},
		Connector:                     mockedadapter.NewDefaultConnector(t),
		ConnectivityReportingStrategy: strategy,
	})
}

func TestThing_ForcedNodeReportRestartsDeduplicationWindow(t *testing.T) {
	t.Parallel()

	p := &recordingPublisher{}
	th := newTestThing(t, p, &memThingState{}, cache.ReportOnChangeOnly())

	sent, err := th.SendConnectivityReport(true)
	require.NoError(t, err)
	assert.True(t, sent)

	sent, err = th.SendConnectivityReport(false)
	require.NoError(t, err)
	assert.False(t, sent, "a forced report must be recorded in the reporting cache")
	assert.Equal(t, 1, p.count(adapter.EvtNetworkNodeReport))
}

func TestThing_FailedNodeReportIsNotCached(t *testing.T) {
	t.Parallel()

	p := &recordingPublisher{failNext: true}
	th := newTestThing(t, p, &memThingState{}, cache.ReportOnChangeOnly())

	sent, err := th.SendConnectivityReport(false)
	require.ErrorIs(t, err, errPublish)
	assert.False(t, sent)

	sent, err = th.SendConnectivityReport(false)
	require.NoError(t, err)
	assert.True(t, sent, "a report that never reached the hub must not count as reported")
}

func TestThing_NilConnectivityDetailsAreSanitised(t *testing.T) {
	t.Parallel()

	connector := mockedadapter.NewConnector(t).MockConnectivity(nil, false)
	th := adapter.NewThing(&recordingPublisher{}, &memThingState{}, &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{Address: "1"},
		Connector:       connector,
	})

	report := th.ConnectivityReport()

	require.NotNil(t, report.ConnectivityDetails)
	assert.Equal(t, adapter.ConnQualityUndefined, report.ConnQuality)
	assert.Equal(t, adapter.ConnTypeUnknown, report.ConnType)
	assert.NotNil(t, report.Operationability)
	assert.Empty(t, report.Operationability)
}

func TestThing_InclusionReportPublishFailureKeepsChecksum(t *testing.T) {
	t.Parallel()

	p := &recordingPublisher{failNext: true}
	ts := &memThingState{checksum: 7}
	th := newTestThing(t, p, ts, nil)

	sent, err := th.SendInclusionReport(false)
	require.ErrorIs(t, err, errPublish)
	assert.False(t, sent)
	assert.Equal(t, uint32(7), ts.checksum, "the checksum must only move once the report reached the hub")
	assert.Empty(t, p.events, "no inclusion_report_sent event for an unpublished report")

	sent, err = th.SendInclusionReport(false)
	require.NoError(t, err)
	assert.True(t, sent, "the retry must re-send the same report")
	assert.NotEqual(t, uint32(7), ts.checksum)
	require.Len(t, p.events, 1)
	assert.Equal(t, "1", p.events[0].Address())

	sent, err = th.SendInclusionReport(false)
	require.NoError(t, err)
	assert.False(t, sent, "an unchanged report must not be re-sent")
	assert.Equal(t, 1, p.count(adapter.EvtThingInclusionReport))
}

func TestThing_InclusionChecksumWriteFailureReportsError(t *testing.T) {
	t.Parallel()

	p := &recordingPublisher{}
	th := newTestThing(t, p, &memThingState{failSet: true}, nil)

	sent, err := th.SendInclusionReport(false)
	require.Error(t, err)
	assert.False(t, sent)
	assert.Equal(t, 1, p.count(adapter.EvtThingInclusionReport), "the report itself was published")
}

func TestThing_UpdateAddsAndRemovesServices(t *testing.T) {
	t.Parallel()

	const topic = "/rt:dev/rn:test_adapter/ad:1/sv:out_bin_switch/ad:1"

	p := &recordingPublisher{}
	svc := adapter.NewService(p, &fimptype.Service{Name: "out_bin_switch", Address: topic})
	th := newTestThing(t, p, &memThingState{}, nil)

	require.NoError(t, th.Update(adapter.ThingUpdateAddService(svc), adapter.ThingUpdateAddService(svc)))

	assert.Len(t, th.Services(""), 1)
	assert.Len(t, th.Services("out_bin_switch"), 1)
	assert.Empty(t, th.Services("meter_elec"))
	assert.Same(t, svc, th.ServiceByTopic("pt:j1/mt:cmd"+topic))
	assert.Len(t, th.InclusionReport().Services, 1, "adding the same topic twice must not duplicate the spec")

	require.NoError(t, th.Update(adapter.ThingUpdateRemoveService(svc)))

	assert.Empty(t, th.Services(""))
	assert.Nil(t, th.ServiceByTopic(topic))
	assert.Empty(t, th.InclusionReport().Services)
	assert.Empty(t, p.messages, "an update must not publish anything by itself")
}
