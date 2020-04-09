package stohealth

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

func NewStaticHealthNode(
	title string,
	healthStatus stoservertypes.HealthStatus,
	descr string,
	kind *stoservertypes.HealthKind,
) HealthChecker {
	return &staticNode{title, healthStatus, descr, kind}
}

type staticNode struct {
	title        string
	healthStatus stoservertypes.HealthStatus
	descr        string
	kind         *stoservertypes.HealthKind
}

func (s *staticNode) CheckHealth() (*stoservertypes.Health, error) {
	return mkHealthWithChildren(s.title, s.healthStatus, s.descr, []HealthChecker{}, s.kind)
}
