package smart

type SmartCtlJsonReport struct {
	JsonFormatVersion  []int `json:"json_format_version"`
	AtaSmartAttributes struct {
		Revision int                 `json:"revision"`
		Table    []AtaSmartAttribute `json:"table"`
	} `json:"ata_smart_attributes"`
	SmartStatus struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	PowerCycleCount int `json:"power_cycle_count"`
	PowerOnTime     struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
	Temperature struct {
		Current int `json:"current"`
	} `json:"temperature"`
}

type AtaSmartAttribute struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Value  int    `json:"value"`
	Worst  int    `json:"worst"`
	Thresh int    `json:"thresh"`
	Raw    struct {
		Value  int    `json:"value"`
		String string `json:"string"`
	} `json:"raw"`
}

func (s *SmartCtlJsonReport) FindSmartAttributeByName(name string) *AtaSmartAttribute {
	for _, item := range s.AtaSmartAttributes.Table {
		if item.Name == name {
			return &item
		}
	}

	return nil
}
