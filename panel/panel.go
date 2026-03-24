package panel

// PanelType represents the type of panel (choose or filter).
type PanelType string

const (
	// PanelChoose is a choose-style panel where users select from a list.
	PanelChoose PanelType = "choose"
	// PanelFilter is a filter-style panel with fuzzy search.
	PanelFilter PanelType = "filter"
)

// Panel represents a single panel configuration.
type Panel struct {
	Type     PanelType
	Items    []string
	ModelIdx int
}
