package parameters

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/event"
)

// NewInclusionReportSentEventHandler creates a new inclusion report sent event handler.
func NewInclusionReportSentEventHandler(thing adapter.ThingRegistry) *event.Handler {
	processor := event.ProcessorFn(func(e event.Event) {
		ep, ok := e.(adapter.ThingEvent)
		if !ok {
			return
		}

		thing := thing.ThingByAddress(ep.Address())

		if thing == nil {
			log.Errorf("inclusion report sent event: thing with address %s not found", ep.Address())

			return
		}

		parameterSrv, ok := getParametersService(thing)
		if !ok {
			return
		}

		if _, err := parameterSrv.SendSupportedParamsReport(true); err != nil {
			log.WithError(err)
		}
	})

	return event.NewHandler(
		processor,
		fmt.Sprintf("%s_inclusion_report_sent", Parameters),
		10,
		WaitForInclusionReportSent(),
	)
}

// WaitForInclusionReportSent creates a filter for a new inclusion report sent event.
func WaitForInclusionReportSent() event.Filter {
	return event.And(
		event.WaitForDomain(adapter.EventDomainAdapterThing),
		event.WaitForClass(adapter.EventClassInclusionReportSent),
	)
}

func getParametersService(thing adapter.Thing) (Service, bool) {
	for _, service := range thing.Services(Parameters) {
		if service, ok := service.(Service); ok {
			return service, true
		}
	}

	return nil, false
}
