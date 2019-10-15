package stohealth

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

func NewStaticHealthNode(title string, healthStatus stoservertypes.HealthStatus, descr string) HealthChecker {
	return &staticNode{title, healthStatus, descr}
}

type staticNode struct {
	title        string
	healthStatus stoservertypes.HealthStatus
	descr        string
}

func (s *staticNode) CheckHealth() (*stoservertypes.Health, error) {
	return mkHealth(s.title, s.healthStatus, s.descr)
}
