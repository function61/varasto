package stoserver

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stoserver/stohealth"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"testing"
	"time"
)

func TestTemperatureToHealthStatus(t *testing.T) {
	// shorthands
	temp := temperatureToHealthStatus
	pass := stoservertypes.HealthStatusPass
	warn := stoservertypes.HealthStatusWarn
	fail := stoservertypes.HealthStatusFail

	assert.Assert(t, temp(0) == fail)
	assert.Assert(t, temp(1) == fail)
	assert.Assert(t, temp(2) == fail)
	assert.Assert(t, temp(3) == fail)
	assert.Assert(t, temp(4) == fail)
	assert.Assert(t, temp(5) == fail)
	assert.Assert(t, temp(6) == pass)
	assert.Assert(t, temp(7) == pass)
	assert.Assert(t, temp(8) == pass)
	assert.Assert(t, temp(9) == pass)
	assert.Assert(t, temp(10) == pass)
	assert.Assert(t, temp(11) == pass)
	assert.Assert(t, temp(12) == pass)
	assert.Assert(t, temp(13) == pass)
	assert.Assert(t, temp(14) == pass)
	assert.Assert(t, temp(15) == pass)
	assert.Assert(t, temp(16) == pass)
	assert.Assert(t, temp(17) == pass)
	assert.Assert(t, temp(18) == pass)
	assert.Assert(t, temp(19) == pass)
	assert.Assert(t, temp(20) == pass)
	assert.Assert(t, temp(21) == pass)
	assert.Assert(t, temp(22) == pass)
	assert.Assert(t, temp(23) == pass)
	assert.Assert(t, temp(24) == pass)
	assert.Assert(t, temp(25) == pass)
	assert.Assert(t, temp(26) == pass)
	assert.Assert(t, temp(27) == pass)
	assert.Assert(t, temp(28) == pass)
	assert.Assert(t, temp(29) == pass)
	assert.Assert(t, temp(30) == pass)
	assert.Assert(t, temp(31) == pass)
	assert.Assert(t, temp(32) == pass)
	assert.Assert(t, temp(33) == pass)
	assert.Assert(t, temp(34) == pass)
	assert.Assert(t, temp(35) == pass)
	assert.Assert(t, temp(36) == pass)
	assert.Assert(t, temp(37) == pass)
	assert.Assert(t, temp(38) == pass)
	assert.Assert(t, temp(39) == pass)
	assert.Assert(t, temp(40) == pass)
	assert.Assert(t, temp(41) == pass)
	assert.Assert(t, temp(42) == pass)
	assert.Assert(t, temp(43) == pass)
	assert.Assert(t, temp(44) == pass)
	assert.Assert(t, temp(45) == pass)
	assert.Assert(t, temp(46) == warn)
	assert.Assert(t, temp(47) == warn)
	assert.Assert(t, temp(48) == warn)
	assert.Assert(t, temp(49) == warn)
	assert.Assert(t, temp(50) == warn)
	assert.Assert(t, temp(51) == fail)
	assert.Assert(t, temp(52) == fail)
	assert.Assert(t, temp(53) == fail)
	assert.Assert(t, temp(54) == fail)
	assert.Assert(t, temp(55) == fail)
	assert.Assert(t, temp(56) == fail)
	assert.Assert(t, temp(57) == fail)
	assert.Assert(t, temp(58) == fail)
	assert.Assert(t, temp(59) == fail)
	assert.Assert(t, temp(60) == fail)
	assert.Assert(t, temp(61) == fail)
	assert.Assert(t, temp(62) == fail)
	assert.Assert(t, temp(63) == fail)
	assert.Assert(t, temp(64) == fail)
	assert.Assert(t, temp(65) == fail)
	assert.Assert(t, temp(66) == fail)
	assert.Assert(t, temp(67) == fail)
	assert.Assert(t, temp(68) == fail)
	assert.Assert(t, temp(69) == fail)
	assert.Assert(t, temp(70) == fail)
	assert.Assert(t, temp(71) == fail)
	assert.Assert(t, temp(72) == fail)
	assert.Assert(t, temp(73) == fail)
	assert.Assert(t, temp(74) == fail)
	assert.Assert(t, temp(75) == fail)
	assert.Assert(t, temp(76) == fail)
	assert.Assert(t, temp(77) == fail)
	assert.Assert(t, temp(78) == fail)
	assert.Assert(t, temp(79) == fail)
	assert.Assert(t, temp(80) == fail)
	assert.Assert(t, temp(81) == fail)
	assert.Assert(t, temp(82) == fail)
	assert.Assert(t, temp(83) == fail)
	assert.Assert(t, temp(84) == fail)
	assert.Assert(t, temp(85) == fail)
	assert.Assert(t, temp(86) == fail)
	assert.Assert(t, temp(87) == fail)
	assert.Assert(t, temp(88) == fail)
	assert.Assert(t, temp(89) == fail)
	assert.Assert(t, temp(90) == fail)
	assert.Assert(t, temp(91) == fail)
	assert.Assert(t, temp(92) == fail)
	assert.Assert(t, temp(93) == fail)
	assert.Assert(t, temp(94) == fail)
	assert.Assert(t, temp(95) == fail)
	assert.Assert(t, temp(96) == fail)
	assert.Assert(t, temp(97) == fail)
	assert.Assert(t, temp(98) == fail)
	assert.Assert(t, temp(99) == fail)
	assert.Assert(t, temp(100) == fail)
}

func TestServerCertHealth(t *testing.T) {
	t0 := time.Date(2019, 12, 16, 0, 0, 0, 0, time.UTC)

	nowBeforeT0 := func(dur time.Duration) time.Time {
		return t0.Add(-dur)
	}

	day := 24 * time.Hour // naive

	check := func(checker stohealth.HealthChecker, expected stoservertypes.HealthStatus) {
		t.Helper()

		actual, err := checker.CheckHealth()
		assert.Assert(t, err == nil)

		assert.EqualString(t, string(actual.Health), string(expected))
	}

	check(serverCertHealth(t0, "", nowBeforeT0(35*day)), stoservertypes.HealthStatusPass)
	check(serverCertHealth(t0, "", nowBeforeT0(20*day)), stoservertypes.HealthStatusWarn)
	check(serverCertHealth(t0, "", nowBeforeT0(5*day)), stoservertypes.HealthStatusFail)
	check(serverCertHealth(t0, "", nowBeforeT0(-5*day)), stoservertypes.HealthStatusFail)
}
