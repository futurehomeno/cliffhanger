package han

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

func TaskReporting(adapter adapter.Adapter, mqtt *fimpgo.MqttTransport, duration time.Duration, voters ...task.Voter) *task.Task {
	return task.New(HandleReporting(adapter, mqtt), duration, voters...)
}

func HandleReporting(adapter adapter.Adapter, mqtt *fimpgo.MqttTransport) func() {
	return func() {
		for _, thing := range adapter.Things() {
			meter, ok := thing.(HAN)
			if !ok {
				continue
			}

			addr := &fimpgo.Address{
				MsgType:         fimpgo.MsgTypeEvt,
				ResourceType:    fimpgo.ResourceTypeDevice,
				ResourceName:    adapter.Name(),
				ResourceAddress: adapter.Address(),
				ServiceName:     MeterElec,
				ServiceAddress:  meter.GetAddress(),
			}

			if meter.SupportsExtendedReport() {
				report, err := meter.ExtendedReport()
				if err != nil {
					log.WithError(err).Error("adapter: failed to retrieve extended report from HAN")

					continue
				}

				msg := fimpgo.NewFloatMapMessage(
					EvtMeterExtReport,
					MeterElec,
					report,
					nil,
					nil,
					nil,
				)

				err = mqtt.Publish(addr, msg)
				if err != nil {
					log.WithError(err).Error("adapter: failed to publish extended report from HAN")
				}

				continue
			}

			for _, unit := range meter.SupportedUnits() {
				value, err := meter.Report(unit)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to retrieve report from HAN for unit: %s", unit)

					continue
				}

				msg := fimpgo.NewFloatMessage(
					EvtMeterReport,
					MeterElec,
					value,
					map[string]string{
						"unit": unit,
					},
					nil,
					nil,
				)

				err = mqtt.Publish(addr, msg)
				if err != nil {
					log.WithError(err).Errorf("adapter: failed to publish report from HAN for unit: %s", unit)
				}
			}
		}
	}
}
