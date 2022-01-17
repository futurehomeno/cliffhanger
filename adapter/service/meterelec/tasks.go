package meterelec

import (
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskReporting creates a reporting task.
func TaskReporting(adapter adapter.Adapter, mqtt *fimpgo.MqttTransport, duration time.Duration, voters ...task.Voter) *task.Task {
	return task.New(HandleReporting(adapter, mqtt), duration, voters...)
}

// HandleReporting creates handler of a reporting task.
func HandleReporting(adapter adapter.Adapter, mqtt *fimpgo.MqttTransport) func() {
	return func() {
		for _, s := range adapter.Services(MeterElec) {
			meterElec, ok := s.(Service)
			if !ok {
				continue
			}

			address, err := fimpgo.NewAddressFromString(meterElec.Topic())
			if err != nil {
				log.WithError(err).Errorf("adapter: failed to parse service address from service topic: %s", meterElec.Topic())

				continue
			}

			address.MsgType = fimpgo.MsgTypeEvt

			if meterElec.SupportsExtendedReport() {
				reportExtended(mqtt, meterElec, address)

				continue
			}

			reportSimplified(mqtt, meterElec, address)
		}
	}
}

// reportExtended prepares and sends extended report.
func reportExtended(mqtt *fimpgo.MqttTransport, meterElec Service, address *fimpgo.Address) {
	report, err := meterElec.ExtendedReport()
	if err != nil {
		log.WithError(err).Error("adapter: failed to retrieve extended report")

		return
	}

	msg := fimpgo.NewFloatMapMessage(
		EvtMeterExtReport,
		MeterElec,
		report,
		nil,
		nil,
		nil,
	)

	err = mqtt.Publish(address, msg)
	if err != nil {
		log.WithError(err).Error("adapter: failed to publish extended report")
	}
}

// reportSimplified prepares and sends simplified report.
func reportSimplified(mqtt *fimpgo.MqttTransport, meterElec Service, address *fimpgo.Address) {
	for _, unit := range meterElec.SupportedUnits() {
		value, normalizedUnit, err := meterElec.Report(unit)
		if err != nil {
			log.WithError(err).Errorf("adapter: failed to retrieve report for unit: %s", unit)

			continue
		}

		msg := fimpgo.NewFloatMessage(
			EvtMeterReport,
			MeterElec,
			value,
			map[string]string{
				"unit": normalizedUnit,
			},
			nil,
			nil,
		)

		err = mqtt.Publish(address, msg)
		if err != nil {
			log.WithError(err).Errorf("adapter: failed to publish report for unit: %s", unit)
		}
	}
}
