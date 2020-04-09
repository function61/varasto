package stoservertypes

// customizations on top of generated code

// needed since we can't take the address of const
func (h HealthKind) Ptr() *HealthKind {
	return &h
}
