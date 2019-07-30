// Health checks for Varasto server
package stohealth

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

type HealthChecker interface {
	CheckHealth() (*stoservertypes.Health, error)
}

type healthFolder struct {
	title    string
	children []HealthChecker
}

func NewHealthFolder(title string, children ...HealthChecker) HealthChecker {
	return &healthFolder{title, children}
}

func (h *healthFolder) CheckHealth() (*stoservertypes.Health, error) {
	return mkHealthWithChildren(h.title, stoservertypes.HealthStatusPass, "", h.children)
}

func mkHealth(title string, health stoservertypes.HealthStatus, details string) (*stoservertypes.Health, error) {
	return mkHealthWithChildren(title, health, details, []HealthChecker{})
}

func mkHealthWithChildren(title string, health stoservertypes.HealthStatus, details string, children []HealthChecker) (*stoservertypes.Health, error) {
	childDtos := []stoservertypes.Health{}

	for _, child := range children {
		childHealth, err := child.CheckHealth()
		if err != nil {
			return nil, err
		}

		childDtos = append(childDtos, *childHealth)
	}

	return &stoservertypes.Health{
		Title:    title,
		Health:   worstOf(childDtos, health),
		Details:  details,
		Children: childDtos,
	}, nil
}

func worstOf(list []stoservertypes.Health, initial stoservertypes.HealthStatus) stoservertypes.HealthStatus {
	worst := initial

	for _, item := range list {
		if statusWorse(item.Health, worst) {
			worst = item.Health
		}
	}

	return worst
}

func statusWorse(a stoservertypes.HealthStatus, b stoservertypes.HealthStatus) bool {
	return statusToInt(a) < statusToInt(b)
}

func statusToInt(status stoservertypes.HealthStatus) int {
	switch stoservertypes.HealthStatusExhaustive97fd15(status) {
	case stoservertypes.HealthStatusPass:
		return 3
	case stoservertypes.HealthStatusWarn:
		return 2
	case stoservertypes.HealthStatusFail:
		return 1
	default:
		panic("unknown")
	}
}
