package stoserver

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

/*
	<=5 freezing (fail)
	5-45 => ok (pass)
	45-50 => uncomfortable  (warn)
	>=50 => too hot (fail)
*/
func temperatureToHealthStatus(tempC int) stoservertypes.HealthStatus {
	switch {
	case tempC <= 5: // freezing
		return stoservertypes.HealthStatusFail
	case tempC <= 45: // ok
		return stoservertypes.HealthStatusPass
	case tempC <= 50: // uncomfortable
		return stoservertypes.HealthStatusWarn
	default: // too hot
		return stoservertypes.HealthStatusFail
	}
}
