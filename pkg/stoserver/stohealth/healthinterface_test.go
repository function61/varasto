package stohealth

import (
	"encoding/json"
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"testing"
)

func TestBasic(t *testing.T) {
	g, _ := getTestGraph()

	jsonBytes, _ := json.MarshalIndent(g, "", "  ")

	assert.EqualString(t, string(jsonBytes), `{
  "Children": [
    {
      "Children": [
        {
          "Children": [],
          "Details": "",
          "Health": "pass",
          "Title": "Dummy 1"
        },
        {
          "Children": [],
          "Details": "",
          "Health": "warn",
          "Title": "Dummy 2"
        }
      ],
      "Details": "",
      "Health": "warn",
      "Title": "SMART"
    }
  ],
  "Details": "",
  "Health": "warn",
  "Title": "Varasto"
}`)
}

func getTestGraph() (*stoservertypes.Health, error) {
	root := NewHealthFolder(
		"Varasto",
		NewHealthFolder("SMART",
			NewStaticHealthNode("Dummy 1", stoservertypes.HealthStatusPass, ""),
			NewStaticHealthNode("Dummy 2", stoservertypes.HealthStatusWarn, "")))

	return root.CheckHealth()
}
