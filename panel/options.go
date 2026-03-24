// Package panel provides a multi-panel TUI for switching between
// choose and filter panels side by side.
//
// Example:
//
//	$ gum panel choose:a:b:c filter:x:y:z
//
// This opens two panels. Navigate between them with tab/shift+tab or arrow keys.
package panel

import (
	"time"

	"github.com/charmbracelet/gum/style"
)

// Options for the panel command.
type Options struct {
	Panel []string `arg:"" help:"Panel configuration (type items...)"`

	Vertical bool `help:"Arrange panels vertically instead of horizontally" group:"Layout"`

	Stacked   bool   `help:"Stack outputs with delimiter (vs sequential)" default:"true" negatable:""`
	Delimiter string `help:"Delimiter between panel outputs" default:"---"`

	Timeout time.Duration `help:"Timeout until panel returns" default:"0s" env:"GUM_PANEL_TIMEOUT"`
	Debug   bool          `help:"Enable debug output" default:"false" negatable:"" env:"GUM_PANEL_DEBUG"`

	Height   int    `help:"Height of each panel" default:"10" env:"GUM_PANEL_HEIGHT"`
	Border   string `help:"Border style (none, single, double, rounded)" default:"single" env:"GUM_PANEL_BORDER"`
	ShowHelp bool   `help:"Show help keybinds" default:"true" negatable:"" env:"GUM_PANEL_SHOW_HELP"`
	Gap      int    `help:"Space between panels" default:"1" env:"GUM_PANEL_GAP"`

	Limit          int    `help:"Maximum number of options per panel" default:"1" group:"Selection"`
	NoLimit        bool   `help:"Pick unlimited number of options" group:"Selection"`
	All            bool   `help:"Require selection in all panels" group:"Selection"`
	Selected       string `help:"Options to pre-select (* for all)" default:"" env:"GUM_PANEL_SELECTED"`
	SelectedPrefix string `help:"Prefix for selected items" default:"✓ " env:"GUM_PANEL_SELECTED_PREFIX"`
	Cursor         string `help:"Cursor prefix" default:"> " env:"GUM_PANEL_CURSOR"`

	// Choose-specific options
	UnselectedPrefix string `help:"Prefix for unselected items" default:"• " env:"GUM_PANEL_UNSELECTED_PREFIX"`
	CursorPrefix     string `help:"Prefix for cursor item" default:"• " env:"GUM_PANEL_CURSOR_PREFIX"`

	// Filter-specific options
	Fuzzy       bool   `help:"Enable fuzzy matching" default:"true" env:"GUM_PANEL_FUZZY" negatable:""`
	FuzzySort   bool   `help:"Sort fuzzy results by their scores" default:"true" env:"GUM_PANEL_FUZZY_SORT" negatable:""`
	Strict      bool   `help:"Only return if anything matched" default:"true" env:"GUM_PANEL_STRICT" negatable:""`
	Placeholder string `help:"Filter placeholder" default:"Filter..." env:"GUM_PANEL_PLACEHOLDER"`
	Prompt      string `help:"Filter prompt" default:"> " env:"GUM_PANEL_PROMPT"`
	Value       string `help:"Initial filter value" default:"" env:"GUM_PANEL_VALUE"`

	// Delimiters
	InputDelimiter  string `help:"Input delimiter" default:"\n" env:"GUM_PANEL_INPUT_DELIMITER"`
	OutputDelimiter string `help:"Output delimiter" default:"\n" env:"GUM_PANEL_OUTPUT_DELIMITER"`
	StripANSI       bool   `help:"Strip ANSI sequences" default:"true" negatable:"" env:"GUM_PANEL_STRIP_ANSI"`

	// Common styles
	CursorStyle         style.Styles `embed:"" prefix:"cursor." set:"defaultForeground=212" envprefix:"GUM_PANEL_CURSOR_"`
	HeaderStyle         style.Styles `embed:"" prefix:"header." set:"defaultForeground=99" envprefix:"GUM_PANEL_HEADER_"`
	ActiveBorderStyle   style.Styles `embed:"" prefix:"active-border." set:"defaultForeground=212" envprefix:"GUM_PANEL_ACTIVE_BORDER_"`
	InactiveBorderStyle style.Styles `embed:"" prefix:"inactive-border." set:"defaultForeground=240" envprefix:"GUM_PANEL_INACTIVE_BORDER_"`
	ItemStyle           style.Styles `embed:"" prefix:"item." envprefix:"GUM_PANEL_ITEM_"`
	SelectedItemStyle   style.Styles `embed:"" prefix:"selected." set:"defaultForeground=212" envprefix:"GUM_PANEL_SELECTED_"`
	MatchStyle          style.Styles `embed:"" prefix:"match." set:"defaultForeground=212" envprefix:"GUM_PANEL_MATCH_"`
	IndicatorStyle      style.Styles `embed:"" prefix:"indicator." set:"defaultForeground=212" envprefix:"GUM_PANEL_INDICATOR_"`

	// Choose-specific styles
	SelectedPrefixStyle   style.Styles `embed:"" prefix:"selected-indicator." set:"defaultForeground=212" envprefix:"GUM_PANEL_SELECTED_PREFIX_"`
	UnselectedPrefixStyle style.Styles `embed:"" prefix:"unselected-prefix." set:"defaultForeground=240" envprefix:"GUM_PANEL_UNSELECTED_PREFIX_"`

	// Filter-specific styles
	TextStyle         style.Styles `embed:"" prefix:"text." envprefix:"GUM_PANEL_TEXT_"`
	CursorTextStyle   style.Styles `embed:"" prefix:"cursor-text." envprefix:"GUM_PANEL_CURSOR_TEXT_"`
	PromptStyle       style.Styles `embed:"" prefix:"prompt." set:"defaultForeground=240" envprefix:"GUM_PANEL_PROMPT_"`
	PlaceholderStyle  style.Styles `embed:"" prefix:"placeholder." set:"defaultForeground=240" envprefix:"GUM_PANEL_PLACEHOLDER_"`
}
