package panel

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/gum/internal/stdin"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type orchestrator struct {
	panels      []Panel
	activeIdx   int
	height      int
	borderStyle lipgloss.Style
	showHelp    bool
	gap         int
	vertical    bool
	stacked     bool
	delimiter   string
	debug       bool
	all         bool

	// Border styles
	activeBorderStyle   lipgloss.Style
	inactiveBorderStyle lipgloss.Style

	// Common styles
	matchStyle        lipgloss.Style
	cursorStyle       lipgloss.Style
	headerStyle       lipgloss.Style
	itemStyle         lipgloss.Style
	selectedItemStyle lipgloss.Style
	indicatorStyle    lipgloss.Style

	// Choose-specific styles
	selectedPrefixStyle   lipgloss.Style
	unselectedPrefixStyle lipgloss.Style

	// Filter-specific styles
	textStyle        lipgloss.Style
	cursorTextStyle  lipgloss.Style
	promptStyle      lipgloss.Style
	placeholderStyle lipgloss.Style

	// Options
	limit            int
	noLimit          bool
	selectedPrefix   string
	unselectedPrefix string
	cursor           string
	cursorPrefix     string
	fuzzy            bool
	fuzzySort        bool
	strict           bool
	placeholder      string
	prompt           string
	value            string

	chooseModels []chooseModel
	filterModels []filterModel
	quitting,
	submitted bool
	errorMessage string
	help         help.Model
	keymap       panelKeymap
}

type chooseModel struct {
	index             int
	currentOrder      int
	height            int
	padding           []int
	cursor            string
	header            string
	selectedPrefix    string
	unselectedPrefix  string
	cursorPrefix      string
	items             []chooseItem
	limit             int
	numSelected       int
	paginator         paginator.Model
	showHelp          bool
	keymap            chooseKeymap
	cursorStyle       lipgloss.Style
	headerStyle       lipgloss.Style
	itemStyle         lipgloss.Style
	selectedItemStyle lipgloss.Style
	quitting          bool
	submitted         bool
}

type chooseItem struct {
	Text     string
	Selected bool
	Order    int
}

type chooseKeymap struct {
	Down,
	Up,
	Right,
	Left,
	Home,
	End,
	ToggleAll,
	Toggle,
	Abort,
	Quit,
	Submit key.Binding
}

type filterModel struct {
	textinput             textinput.Model
	viewport              *viewport.Model
	choices               map[string]string
	filteringChoices      []string
	matches               []fuzzy.Match
	cursor                int
	header                string
	selected              map[string]struct{}
	limit                 int
	numSelected           int
	indicator             string
	selectedPrefix        string
	unselectedPrefix      string
	height                int
	padding               []int
	quitting              bool
	headerStyle           lipgloss.Style
	matchStyle            lipgloss.Style
	textStyle             lipgloss.Style
	cursorTextStyle       lipgloss.Style
	indicatorStyle        lipgloss.Style
	selectedPrefixStyle   lipgloss.Style
	unselectedPrefixStyle lipgloss.Style
	reverse               bool
	fuzzy                 bool
	sort                  bool
	strict                bool
	value                 string
	showHelp              bool
	keymap                filterKeymap
	submitted             bool
}

type filterKeymap struct {
	FocusInSearch,
	FocusOutSearch,
	Down,
	Up,
	NDown,
	NUp,
	Home,
	End,
	ToggleAndNext,
	ToggleAndPrevious,
	ToggleAll,
	Toggle,
	Abort,
	Quit,
	Submit key.Binding
}

type panelKeymap struct {
	NextPanel  key.Binding
	PrevPanel  key.Binding
	LeftPanel  key.Binding
	RightPanel key.Binding
	Submit     key.Binding
	Quit       key.Binding
}

// FullHelp implements help.KeyMap.
func (k panelKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPanel, k.PrevPanel, k.LeftPanel, k.RightPanel},
		{k.Submit, k.Quit},
	}
}

// ShortHelp implements help.KeyMap.
func (k panelKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.NextPanel,
		k.PrevPanel,
		k.Submit,
		k.Quit,
	}
}

func defaultPanelKeymap() panelKeymap {
	return panelKeymap{
		NextPanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		PrevPanel: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
		LeftPanel: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "prev panel"),
		),
		RightPanel: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "next panel"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "next/submit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc", "ctrl+c"),
			key.WithHelp("esc", "quit"),
		),
	}
}

// BorderStyle returns the lipgloss styling for panel borders.
func (o Options) BorderStyle() lipgloss.Style {
	borderStyle := lipgloss.NewStyle()

	switch o.Border {
	case "none":
		borderStyle = borderStyle.BorderStyle(lipgloss.Border{})
	case "single":
		borderStyle = borderStyle.BorderStyle(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
		})
	case "double":
		borderStyle = borderStyle.BorderStyle(lipgloss.Border{
			Top:         "═",
			Bottom:      "═",
			Left:        "║",
			Right:       "║",
			TopLeft:     "╔",
			TopRight:    "╗",
			BottomLeft:  "╚",
			BottomRight: "╝",
		})
	case "rounded":
		borderStyle = borderStyle.BorderStyle(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
		})
	}

	return borderStyle
}

func parsePanelsFromArgs(args []string, inputDelimiter string, stripANSI bool) ([]Panel, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no panels specified")
	}

	var stdinData string
	if !stdin.IsEmpty() {
		data, _ := stdin.Read(stdin.StripANSI(stripANSI))
		if data != "" {
			stdinData = data
		}
	}

	panels := make([]Panel, 0)
	var currentPanel *Panel

	for _, arg := range args {
		argLower := strings.ToLower(arg)

		if argLower == "choose" || argLower == "filter" {
			if currentPanel != nil && len(currentPanel.Items) > 0 {
				panels = append(panels, *currentPanel)
			}

			panelType := PanelType(argLower)
			currentPanel = &Panel{
				Type:  panelType,
				Items: []string{},
			}
		} else if currentPanel != nil {
			currentPanel.Items = append(currentPanel.Items, arg)
		} else if stdinData != "" {
			items := strings.Split(stdinData, inputDelimiter)
			for i := range items {
				if items[i] != "" {
					panels = append(panels, Panel{
						Type:  PanelChoose,
						Items: []string{items[i]},
					})
				}
			}
		}
	}

	if currentPanel != nil && len(currentPanel.Items) > 0 {
		panels = append(panels, *currentPanel)
	}

	if len(panels) == 0 {
		return nil, fmt.Errorf("no panels specified")
	}

	return panels, nil
}

func (m *orchestrator) initModels(o Options) error {
	m.chooseModels = make([]chooseModel, 0)
	m.filterModels = make([]filterModel, 0)

	chooseIdx := 0
	filterIdx := 0

	for i, panel := range m.panels {
		switch panel.Type {
		case PanelChoose:
			items := make([]chooseItem, len(panel.Items))
			for i, item := range panel.Items {
				items[i] = chooseItem{Text: item, Selected: false, Order: 0}
			}

			pager := paginator.New()
			pager.SetTotalPages((len(items) + o.Height - 1) / o.Height)
			pager.PerPage = o.Height
			pager.Type = paginator.Dots

			km := defaultChooseKeymap()
			if o.NoLimit || o.Limit > 1 {
				km.Toggle.SetEnabled(true)
			}
			if o.NoLimit {
				km.ToggleAll.SetEnabled(true)
			}

			cm := chooseModel{
				index:             0,
				currentOrder:      0,
				height:            o.Height,
				padding:           []int{0, 0, 0, 0},
				cursor:            o.Cursor,
				header:            "",
				selectedPrefix:    o.SelectedPrefix,
				unselectedPrefix:  o.UnselectedPrefix,
				cursorPrefix:      o.CursorPrefix,
				items:             items,
				limit:             o.Limit,
				numSelected:       0,
				paginator:         pager,
				showHelp:          false,
				keymap:            km,
				cursorStyle:       o.CursorStyle.ToLipgloss(),
				headerStyle:       o.HeaderStyle.ToLipgloss(),
				itemStyle:         o.ItemStyle.ToLipgloss(),
				selectedItemStyle: o.SelectedItemStyle.ToLipgloss(),
				quitting:          false,
				submitted:         false,
			}
			m.chooseModels = append(m.chooseModels, cm)
			m.panels[i].ModelIdx = chooseIdx
			chooseIdx++

		case PanelFilter:
			ti := textinput.New()
			ti.Focus()
			ti.Prompt = o.Prompt
			ti.PromptStyle = o.PromptStyle.ToLipgloss()
			ti.Placeholder = o.Placeholder
			ti.PlaceholderStyle = o.PlaceholderStyle.ToLipgloss()

			v := viewport.New(0, o.Height)

			choices := map[string]string{}
			filteringChoices := []string{}
			for _, opt := range panel.Items {
				choices[opt] = opt
				filteringChoices = append(filteringChoices, opt)
			}

			matches := matchAll(filteringChoices)

			fkm := defaultFilterKeymap()
			if o.NoLimit || o.Limit > 1 {
				fkm.Toggle.SetEnabled(true)
				fkm.ToggleAndPrevious.SetEnabled(true)
				fkm.ToggleAndNext.SetEnabled(true)
				fkm.ToggleAll.SetEnabled(true)
			}

			fm := filterModel{
				textinput:             ti,
				viewport:              &v,
				choices:               choices,
				filteringChoices:      filteringChoices,
				matches:               matches,
				cursor:                0,
				header:                "",
				selected:              make(map[string]struct{}),
				limit:                 o.Limit,
				numSelected:           0,
				indicator:             "•",
				selectedPrefix:        o.SelectedPrefix,
				unselectedPrefix:      o.UnselectedPrefix,
				height:                o.Height,
				padding:               []int{0, 0, 0, 0},
				quitting:              false,
				headerStyle:           o.HeaderStyle.ToLipgloss(),
				matchStyle:            o.MatchStyle.ToLipgloss(),
				textStyle:             o.TextStyle.ToLipgloss(),
				cursorTextStyle:       o.CursorTextStyle.ToLipgloss(),
				indicatorStyle:        o.IndicatorStyle.ToLipgloss(),
				selectedPrefixStyle:   o.SelectedPrefixStyle.ToLipgloss(),
				unselectedPrefixStyle: o.UnselectedPrefixStyle.ToLipgloss(),
				reverse:               false,
				fuzzy:                 o.Fuzzy,
				sort:                  o.FuzzySort,
				strict:                o.Strict,
				value:                 o.Value,
				showHelp:              false,
				keymap:                fkm,
				submitted:             false,
			}
			if o.Value != "" {
				fm.textinput.SetValue(o.Value)
			}
			m.filterModels = append(m.filterModels, fm)
			m.panels[i].ModelIdx = filterIdx
			filterIdx++
		}
	}

	return nil
}

func defaultChooseKeymap() chooseKeymap {
	return chooseKeymap{
		Down: key.NewBinding(
			key.WithKeys("down", "j", "ctrl+j", "ctrl+n"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k", "ctrl+k", "ctrl+p"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l", "ctrl+f"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h", "ctrl+b"),
		),
		Home: key.NewBinding(
			key.WithKeys("g", "home"),
		),
		End: key.NewBinding(
			key.WithKeys("G", "end"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("a", "A", "ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
			key.WithDisabled(),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "tab", "x", "ctrl+@"),
			key.WithHelp("x", "toggle"),
			key.WithDisabled(),
		),
		Abort: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "abort"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter", "ctrl+q"),
			key.WithHelp("enter", "submit"),
		),
	}
}

func defaultFilterKeymap() filterKeymap {
	return filterKeymap{
		Down: key.NewBinding(
			key.WithKeys("down", "ctrl+j", "ctrl+n"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "ctrl+k", "ctrl+p"),
		),
		NDown: key.NewBinding(
			key.WithKeys("j"),
		),
		NUp: key.NewBinding(
			key.WithKeys("k"),
		),
		Home: key.NewBinding(
			key.WithKeys("g", "home"),
		),
		End: key.NewBinding(
			key.WithKeys("G", "end"),
		),
		ToggleAndNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle"),
			key.WithDisabled(),
		),
		ToggleAndPrevious: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "toggle"),
			key.WithDisabled(),
		),
		Toggle: key.NewBinding(
			key.WithKeys("ctrl+@"),
			key.WithHelp("ctrl+@", "toggle"),
			key.WithDisabled(),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
			key.WithDisabled(),
		),
		FocusInSearch: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		FocusOutSearch: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "blur search"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
		),
		Abort: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "abort"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter", "ctrl+q"),
			key.WithHelp("enter", "submit"),
		),
	}
}

func matchAll(options []string) []fuzzy.Match {
	matches := make([]fuzzy.Match, len(options))
	for i, option := range options {
		matches[i] = fuzzy.Match{Str: option}
	}
	return matches
}

func (m orchestrator) Init() tea.Cmd {
	var cmds []tea.Cmd

	for range m.chooseModels {
		cmds = append(cmds, nil)
	}
	for range m.filterModels {
		cmds = append(cmds, textinput.Blink)
	}

	return tea.Batch(cmds...)
}

func (m orchestrator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		km := m.keymap

		switch {
		case key.Matches(msg, km.NextPanel), key.Matches(msg, km.RightPanel):
			m.activeIdx = (m.activeIdx + 1) % len(m.panels)
			m.errorMessage = ""
			return m, nil

		case key.Matches(msg, km.PrevPanel), key.Matches(msg, km.LeftPanel):
			m.activeIdx = (m.activeIdx - 1 + len(m.panels)) % len(m.panels)
			m.errorMessage = ""
			return m, nil

		case key.Matches(msg, km.Submit):
			return m.handleSubmit()

		case key.Matches(msg, km.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Safety check: activeIdx in range
	if m.activeIdx < 0 || m.activeIdx >= len(m.panels) {
		return m, nil
	}

	panel := m.panels[m.activeIdx]

	switch panel.Type {
	case PanelChoose:
		if panel.ModelIdx < 0 || panel.ModelIdx >= len(m.chooseModels) {
			err := fmt.Errorf("internal error: choose model index %d out of range (have %d models)", panel.ModelIdx, len(m.chooseModels))
			m.quitting = true
			return m, func() tea.Msg { return err }
		}
		cm := &m.chooseModels[panel.ModelIdx]
		return m, m.updateChooseModel(cm, msg)
	case PanelFilter:
		if panel.ModelIdx < 0 || panel.ModelIdx >= len(m.filterModels) {
			err := fmt.Errorf("internal error: filter model index %d out of range (have %d models)", panel.ModelIdx, len(m.filterModels))
			m.quitting = true
			return m, func() tea.Msg { return err }
		}
		fm := &m.filterModels[panel.ModelIdx]
		return m, m.updateFilterModel(fm, msg)
	default:
		err := fmt.Errorf("internal error: unknown panel type: %s", panel.Type)
		m.quitting = true
		return m, func() tea.Msg { return err }
	}
}

func (m *orchestrator) updateChooseModel(cm *chooseModel, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		km := cm.keymap
		switch {
		case key.Matches(msg, km.Down):
			cm.index++
			if cm.index >= len(cm.items) {
				cm.index = 0
				cm.paginator.Page = 0
			}
			_, end := cm.paginator.GetSliceBounds(len(cm.items))
			if cm.index >= end {
				cm.paginator.NextPage()
			}
		case key.Matches(msg, km.Up):
			cm.index--
			if cm.index < 0 {
				cm.index = len(cm.items) - 1
				cm.paginator.Page = cm.paginator.TotalPages - 1
			}
			_, end := cm.paginator.GetSliceBounds(len(cm.items))
			if cm.index >= end {
				cm.paginator.NextPage()
			}
			start, _ := cm.paginator.GetSliceBounds(len(cm.items))
			if cm.index < start {
				cm.paginator.PrevPage()
			}
		case key.Matches(msg, km.Home):
			cm.index = 0
			cm.paginator.Page = 0
		case key.Matches(msg, km.End):
			cm.index = len(cm.items) - 1
			cm.paginator.Page = cm.paginator.TotalPages - 1
		case key.Matches(msg, km.Toggle):
			if cm.items[cm.index].Selected {
				cm.items[cm.index].Selected = false
				cm.numSelected--
			} else if cm.numSelected < cm.limit {
				cm.items[cm.index].Selected = true
				cm.items[cm.index].Order = cm.currentOrder
				cm.numSelected++
				cm.currentOrder++
			}
		case key.Matches(msg, km.ToggleAll):
			if cm.limit <= 1 {
				break
			}
			if cm.numSelected < len(cm.items) && cm.numSelected < cm.limit {
				for i := range cm.items {
					if cm.numSelected >= cm.limit {
						break
					}
					if !cm.items[i].Selected {
						cm.items[i].Selected = true
						cm.items[i].Order = cm.currentOrder
						cm.numSelected++
						cm.currentOrder++
					}
				}
			} else {
				for i := range cm.items {
					cm.items[i].Selected = false
					cm.items[i].Order = 0
				}
				cm.numSelected = 0
				cm.currentOrder = 0
			}
		case key.Matches(msg, km.Submit):
			cm.submitted = true
			cm.quitting = true
			if cm.limit <= 1 && cm.numSelected < 1 {
				cm.items[cm.index].Selected = true
			}
		case key.Matches(msg, km.Quit):
			cm.quitting = true
		}
	}

	var cmd tea.Cmd
	cm.paginator, cmd = cm.paginator.Update(msg)
	return cmd
}

func (m *orchestrator) updateFilterModel(fm *filterModel, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		km := fm.keymap
		switch {
		case key.Matches(msg, km.FocusInSearch):
			fm.textinput.Focus()
		case key.Matches(msg, km.FocusOutSearch):
			fm.textinput.Blur()
		case key.Matches(msg, km.Quit):
			fm.quitting = true
		case key.Matches(msg, km.Submit):
			fm.submitted = true
			fm.quitting = true
		case key.Matches(msg, km.Down, km.NDown):
			m.filterCursorDown(fm)
		case key.Matches(msg, km.Up, km.NUp):
			m.filterCursorUp(fm)
		case key.Matches(msg, km.Home):
			fm.cursor = 0
		case key.Matches(msg, km.End):
			fm.cursor = len(fm.choices) - 1
		case key.Matches(msg, km.ToggleAndNext):
			if fm.limit == 1 {
				break
			}
			m.toggleFilterSelection(fm)
			m.filterCursorDown(fm)
		case key.Matches(msg, km.ToggleAndPrevious):
			if fm.limit == 1 {
				break
			}
			m.toggleFilterSelection(fm)
			m.filterCursorUp(fm)
		case key.Matches(msg, km.Toggle):
			if fm.limit == 1 {
				break
			}
			m.toggleFilterSelection(fm)
		case key.Matches(msg, km.ToggleAll):
			if fm.limit <= 1 {
				break
			}
			if fm.numSelected < len(fm.matches) && fm.numSelected < fm.limit {
				for i := range fm.matches {
					if fm.numSelected >= fm.limit {
						break
					}
					if _, ok := fm.selected[fm.matches[i].Str]; !ok {
						fm.selected[fm.matches[i].Str] = struct{}{}
						fm.numSelected++
					}
				}
			} else {
				fm.selected = make(map[string]struct{})
				fm.numSelected = 0
			}
		default:
			var icmd tea.Cmd
			fm.textinput, icmd = fm.textinput.Update(msg)

			newValue := fm.textinput.Value()
			var choices []string
			if !fm.strict {
				choices = append(choices, newValue)
			}
			choices = append(choices, fm.filteringChoices...)
			if fm.fuzzy {
				if fm.sort {
					fm.matches = fuzzy.Find(newValue, choices)
				} else {
					fm.matches = fuzzy.FindNoSort(newValue, choices)
				}
			} else {
				fm.matches = exactMatches(newValue, choices)
			}
			if newValue == "" {
				fm.matches = matchAll(fm.filteringChoices)
			}
			fm.cursor = 0

			return icmd
		}
	}

	fm.keymap.FocusInSearch.SetEnabled(!fm.textinput.Focused())
	fm.keymap.FocusOutSearch.SetEnabled(fm.textinput.Focused())
	fm.keymap.NUp.SetEnabled(!fm.textinput.Focused())
	fm.keymap.NDown.SetEnabled(!fm.textinput.Focused())
	fm.keymap.Home.SetEnabled(!fm.textinput.Focused())
	fm.keymap.End.SetEnabled(!fm.textinput.Focused())

	if len(fm.matches) > 0 {
		fm.cursor = (fm.cursor + len(fm.matches)) % len(fm.matches)
	}

	return nil
}

func (m *orchestrator) filterCursorDown(fm *filterModel) {
	if len(fm.matches) == 0 {
		return
	}
	fm.cursor = (fm.cursor + 1) % len(fm.matches)
}

func (m *orchestrator) filterCursorUp(fm *filterModel) {
	if len(fm.matches) == 0 {
		return
	}
	fm.cursor = (fm.cursor - 1 + len(fm.matches)) % len(fm.matches)
}

func (m *orchestrator) toggleFilterSelection(fm *filterModel) {
	if len(fm.matches) == 0 {
		return
	}
	currentMatch := fm.matches[fm.cursor].Str
	if _, ok := fm.selected[currentMatch]; ok {
		delete(fm.selected, currentMatch)
		fm.numSelected--
	} else if fm.numSelected < fm.limit {
		fm.selected[currentMatch] = struct{}{}
		fm.numSelected++
	}
}

func exactMatches(search string, choices []string) []fuzzy.Match {
	matches := fuzzy.Matches{}
	search = strings.ToLower(search)
	for i, choice := range choices {
		matchedString := strings.ToLower(choice)
		index := strings.Index(matchedString, search)
		if index >= 0 {
			matchedIndexes := []int{}
			for s := range search {
				matchedIndexes = append(matchedIndexes, index+s)
			}
			matches = append(matches, fuzzy.Match{
				Str:            choice,
				Index:          i,
				MatchedIndexes: matchedIndexes,
			})
		}
	}
	return matches
}

func (m orchestrator) View() string {
	if m.quitting {
		return ""
	}

	if len(m.panels) == 0 {
		return ""
	}

	// Safety check: activeIdx in range
	if m.activeIdx < 0 || m.activeIdx >= len(m.panels) {
		m.activeIdx = 0
	}

	var panelViews []string

	for i, panel := range m.panels {
		panelBorder := m.borderStyle
		if i != m.activeIdx {
			panelBorder = panelBorder.BorderForeground(lipgloss.Color("240"))
		}

		var view string

		switch panel.Type {
		case PanelChoose:
			if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.chooseModels) {
				cm := m.chooseModels[panel.ModelIdx]
				view = m.renderChooseView(&cm)
			}
		case PanelFilter:
			if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.filterModels) {
				fm := m.filterModels[panel.ModelIdx]
				view = m.renderFilterView(&fm)
			}
		}

		panelHeader := strings.ToUpper(string(panel.Type))
		headerView := m.headerStyle.Render(" " + panelHeader + " ")
		if i == m.activeIdx {
			headerView = " ● " + headerView
		}

		panelView := panelBorder.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(headerView),
				view,
			),
		)
		panelViews = append(panelViews, panelView)
	}

	var joinedView string
	if m.vertical {
		joinedView = lipgloss.JoinVertical(lipgloss.Top, panelViews...)
	} else {
		if m.gap > 0 && len(panelViews) > 1 {
			gapViews := make([]string, 0, len(panelViews)*2-1)
			for i, pv := range panelViews {
				gapViews = append(gapViews, pv)
				if i < len(panelViews)-1 {
					gapViews = append(gapViews, strings.Repeat(" ", m.gap))
				}
			}
			joinedView = lipgloss.JoinHorizontal(lipgloss.Top, gapViews...)
		} else {
			joinedView = lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)
		}
	}

	if m.showHelp {
		var helpText string
		if m.errorMessage != "" {
			helpText = " " + m.errorMessage + " "
		} else {
			// Use help.Model for consistent rendering with filter and choose
			helpText = " " + m.help.View(m.keymap)
		}
		joinedView = lipgloss.JoinVertical(lipgloss.Left, joinedView, helpText)
	}

	return joinedView
}

func (m orchestrator) renderChooseView(cm *chooseModel) string {
	if cm.quitting {
		return ""
	}

	var s strings.Builder

	start, end := cm.paginator.GetSliceBounds(len(cm.items))
	for i, item := range cm.items[start:end] {
		actualIdx := start + i

		if actualIdx == cm.index%cm.height {
			s.WriteString(cm.cursorStyle.Render(cm.cursor))
		} else {
			s.WriteString(strings.Repeat(" ", lipgloss.Width(cm.cursor)))
		}

		if item.Selected {
			s.WriteString(cm.selectedItemStyle.Render(cm.selectedPrefix + item.Text))
		} else if actualIdx == cm.index%cm.height {
			s.WriteString(cm.cursorStyle.Render(cm.cursorPrefix + item.Text))
		} else {
			s.WriteString(cm.itemStyle.Render(cm.unselectedPrefix + item.Text))
		}
		if i != cm.height {
			s.WriteRune('\n')
		}
	}

	if cm.paginator.TotalPages > 1 {
		s.WriteString(strings.Repeat("\n", cm.height-cm.paginator.ItemsOnPage(len(cm.items))+1))
		s.WriteString("  " + cm.paginator.View())
	}

	return s.String()
}

func (m orchestrator) renderFilterView(fm *filterModel) string {
	if fm.quitting {
		return ""
	}

	var s strings.Builder
	var lineTextStyle lipgloss.Style

	inputView := fm.textinput.View()
	s.WriteString(inputView)
	s.WriteString("\n")

	for i := range fm.matches {
		if i == fm.cursor {
			s.WriteString(fm.indicatorStyle.Render(fm.indicator))
			lineTextStyle = fm.cursorTextStyle
		} else {
			s.WriteString(strings.Repeat(" ", lipgloss.Width(fm.indicator)))
			lineTextStyle = fm.textStyle
		}

		if _, ok := fm.selected[fm.matches[i].Str]; ok {
			s.WriteString(fm.selectedPrefixStyle.Render(fm.selectedPrefix))
		} else if fm.limit > 1 {
			s.WriteString(fm.unselectedPrefixStyle.Render(fm.unselectedPrefix))
		} else {
			s.WriteString(" ")
		}

		styledOption := fm.choices[fm.matches[i].Str]
		if len(fm.matches[i].MatchedIndexes) == 0 {
			s.WriteString(lineTextStyle.Render(styledOption))
			s.WriteRune('\n')
			continue
		}

		s.WriteString(lineTextStyle.Render(styledOption))
		s.WriteRune('\n')
	}

	return s.String()
}

func (m orchestrator) getResults(outputDelimiter string) []string {
	var results []string

	for _, panel := range m.panels {
		switch panel.Type {
		case PanelChoose:
			if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.chooseModels) {
				cm := m.chooseModels[panel.ModelIdx]
				selectedCount := 0
				for _, item := range cm.items {
					if item.Selected {
						results = append(results, item.Text)
						selectedCount++
					}
				}
				// If no selection in this panel, add whitespace to maintain output field count
				if selectedCount == 0 {
					results = append(results, "")
				}
			}
		case PanelFilter:
			if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.filterModels) {
				fm := m.filterModels[panel.ModelIdx]
				selectedCount := 0
				for k := range fm.selected {
					results = append(results, k)
					selectedCount++
				}
				// If no selection in this panel, add whitespace to maintain output field count
				if selectedCount == 0 {
					results = append(results, "")
				}
			}
		}
	}

	return results
}

func (m *orchestrator) handleSubmit() (orchestrator, tea.Cmd) {
	panel := m.panels[m.activeIdx]

	switch panel.Type {
	case PanelChoose:
		if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.chooseModels) {
			cm := &m.chooseModels[panel.ModelIdx]
			if cm.limit <= 1 && cm.numSelected < 1 {
				cm.items[cm.index].Selected = true
				cm.numSelected = 1
			}
		}
	case PanelFilter:
		if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.filterModels) {
			fm := &m.filterModels[panel.ModelIdx]
			if len(fm.selected) == 0 && fm.limit <= 1 && len(fm.matches) > 0 {
				if fm.strict && len(fm.matches) == 0 {
					m.errorMessage = "No matches found"
					return *m, nil
				}
				fm.selected[fm.matches[fm.cursor].Str] = struct{}{}
			}
		}
	}

	if m.activeIdx < len(m.panels)-1 {
		m.activeIdx++
		m.errorMessage = ""
		return *m, nil
	}

	if m.all {
		firstIncomplete := m.findFirstIncompletePanel()
		if firstIncomplete >= 0 {
			m.activeIdx = firstIncomplete
			m.errorMessage = "You must make a choice in all panels!"
			return *m, nil
		}
	}

	m.submitted = true
	m.quitting = true
	return *m, tea.Quit
}

func (m *orchestrator) findFirstIncompletePanel() int {
	for i, panel := range m.panels {
		switch panel.Type {
		case PanelChoose:
			if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.chooseModels) {
				cm := m.chooseModels[panel.ModelIdx]
				if cm.numSelected == 0 {
					return i
				}
			}
		case PanelFilter:
			if panel.ModelIdx >= 0 && panel.ModelIdx < len(m.filterModels) {
				fm := m.filterModels[panel.ModelIdx]
				if len(fm.selected) == 0 {
					return i
				}
			}
		}
	}
	return -1
}
