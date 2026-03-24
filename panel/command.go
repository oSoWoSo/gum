package panel

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/gum/internal/timeout"
	"github.com/charmbracelet/gum/internal/tty"
)

// Run provides a shell script interface for running multiple panels
// side by side, allowing users to navigate between them.
func (o Options) Run() error {
	if len(o.Panel) == 0 {
		return os.ErrInvalid
	}

	panels, err := parsePanelsFromArgs(o.Panel, o.InputDelimiter, o.StripANSI)
	if err != nil {
		return err
	}

	height := o.Height
	if height <= 0 {
		height = 10
	}

	m := orchestrator{
		panels:      panels,
		activeIdx:   0,
		height:      height,
		borderStyle: o.BorderStyle(),
		showHelp:    o.ShowHelp,
		gap:         o.Gap,
		vertical:    o.Vertical,
		stacked:     o.Stacked,
		delimiter:   o.Delimiter,
		debug:       o.Debug,
		all:         o.All,

		// Border styles
		activeBorderStyle:   o.ActiveBorderStyle.ToLipgloss(),
		inactiveBorderStyle: o.InactiveBorderStyle.ToLipgloss(),

		// Common styles
		matchStyle:        o.MatchStyle.ToLipgloss(),
		cursorStyle:       o.CursorStyle.ToLipgloss(),
		headerStyle:       o.HeaderStyle.ToLipgloss(),
		itemStyle:         o.ItemStyle.ToLipgloss(),
		selectedItemStyle: o.SelectedItemStyle.ToLipgloss(),
		indicatorStyle:    o.IndicatorStyle.ToLipgloss(),

		// Choose-specific styles
		selectedPrefixStyle:   o.SelectedPrefixStyle.ToLipgloss(),
		unselectedPrefixStyle: o.UnselectedPrefixStyle.ToLipgloss(),

		// Filter-specific styles
		textStyle:        o.TextStyle.ToLipgloss(),
		cursorTextStyle:  o.CursorTextStyle.ToLipgloss(),
		promptStyle:      o.PromptStyle.ToLipgloss(),
		placeholderStyle: o.PlaceholderStyle.ToLipgloss(),

		// Options
		limit:            o.Limit,
		noLimit:          o.NoLimit,
		selectedPrefix:   o.SelectedPrefix,
		unselectedPrefix: o.UnselectedPrefix,
		cursor:           o.Cursor,
		cursorPrefix:     o.CursorPrefix,
		fuzzy:            o.Fuzzy,
		fuzzySort:        o.FuzzySort,
		strict:           o.Strict,
		placeholder:      o.Placeholder,
		prompt:           o.Prompt,
		value:            o.Value,

		keymap: defaultPanelKeymap(),
		help:   help.New(),
	}

	if err := m.initModels(o); err != nil {
		return err
	}

	ctx, cancel := timeout.Context(o.Timeout)
	defer cancel()

	tm, err := tea.NewProgram(
		m,
		tea.WithOutput(os.Stderr),
		tea.WithContext(ctx),
	).Run()
	if err != nil {
		return err
	}

	result := tm.(orchestrator)

	if !result.submitted {
		return os.ErrInvalid
	}

	output := result.getResults(o.OutputDelimiter)

	var finalOutput string
	if o.Stacked {
		finalOutput = strings.Join(output, "\n"+o.Delimiter+"\n")
	} else {
		finalOutput = strings.Join(output, o.OutputDelimiter)
	}

	tty.Println(finalOutput)
	return nil
}
