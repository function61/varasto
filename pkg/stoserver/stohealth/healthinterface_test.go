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
          "Title": "Disk Dummy 1 SMART"
        },
        {
          "Children": [],
          "Details": "",
          "Health": "pass",
          "Title": "Disk Dummy 2 SMART"
        }
      ],
      "Details": "",
      "Health": "pass",
      "Title": "SMART"
    }
  ],
  "Details": "",
  "Health": "pass",
  "Title": "Varasto"
}`)
}

func getTestGraph() (*stoservertypes.Health, error) {
	root := NewHealthFolder("Varasto", NewHealthFolder("SMART", NewSmartChecker("Dummy 1"), NewSmartChecker("Dummy 2")))

	return root.CheckHealth()
}
