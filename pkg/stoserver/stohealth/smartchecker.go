package stohealth

import (
	"fmt"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

// TODO: this is a placeholder and does not work yet
func NewSmartChecker(diskName string) HealthChecker {
	return &smartChecker{diskName}
}

type smartChecker struct {
	diskName string
}

func (s *smartChecker) CheckHealth() (*stoservertypes.Health, error) {
	return mkHealth(fmt.Sprintf("Disk %s SMART", s.diskName), stoservertypes.HealthStatusPass, "")
}
