package panel

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var charPool = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" +
	"áčďéěíňóřšťúůýžÁČĎÉĚÍŇÓŘŠŤÚŮÝŽ" +
	"!#@&%$^*()[]{}<>?/\\|~`-_=+\"':;,."

func generateRandomMessage() string {
	return generateRandomMessageWithDelimiter(" ")
}

func generateRandomMessageWithDelimiter(delimiter string) string {
	numWords := 1 + rand.Intn(6)
	var words []string
	for i := 0; i < numWords; i++ {
		wordLen := 1 + rand.Intn(15)
		var word strings.Builder
		for j := 0; j < wordLen; j++ {
			word.WriteByte(charPool[rand.Intn(len(charPool))])
		}
		words = append(words, word.String())
	}
	result := strings.Join(words, " ")

	if strings.Contains(result, delimiter) {
		result = `"` + result + `"`
	}

	return result
}

func generateRandomMessagesWithDelimiterForcedItems(count int, delimiter string) []string {
	var messages []string

	singleChar := "a"
	for i := 0; i < len(charPool); i++ {
		c := string(charPool[i])
		if c != delimiter && !strings.Contains(c, delimiter) {
			singleChar = c
			break
		}
	}
	messages = append(messages, singleChar)

	messages = append(messages, `"`+delimiter+`"`)

	remaining := count - 2
	for i := 0; i < remaining; i++ {
		messages = append(messages, generateRandomMessage())
	}

	return messages
}

func generateRandomMessages(count int) []string {
	var messages []string
	for i := 0; i < count; i++ {
		messages = append(messages, generateRandomMessage())
	}
	return messages
}

func generateRandomMessagesWithDelimiter(count int, delimiter string) []string {
	var messages []string
	for i := 0; i < count; i++ {
		messages = append(messages, generateRandomMessageWithDelimiter(delimiter))
	}
	return messages
}

func TestParsePanelsFromArgs(t *testing.T) {
	tests := map[string]struct {
		args      []string
		wantLen   int
		wantTypes []PanelType
		wantErr   bool
	}{
		"single choose panel": {
			args:      []string{"choose", "a", "b", "c"},
			wantLen:   1,
			wantTypes: []PanelType{PanelChoose},
		},
		"single filter panel": {
			args:      []string{"filter", "one", "two", "three"},
			wantLen:   1,
			wantTypes: []PanelType{PanelFilter},
		},
		"two panels with -- separator": {
			args:      []string{"choose", "a", "b", "c", "--", "filter", "x", "y", "z"},
			wantLen:   2,
			wantTypes: []PanelType{PanelChoose, PanelFilter},
		},
		"three panels": {
			args:      []string{"choose", "a", "b", "--", "filter", "c", "d", "--", "choose", "e", "f"},
			wantLen:   3,
			wantTypes: []PanelType{PanelChoose, PanelFilter, PanelChoose},
		},
		"choose with flags": {
			args:      []string{"choose", "--limit", "3", "--header", "Ovoce", "a", "b", "c"},
			wantLen:   1,
			wantTypes: []PanelType{PanelChoose},
		},
		"filter with flags and -- separator": {
			args:      []string{"choose", "a", "b", "--", "filter", "--no-fuzzy", "--placeholder", "Hledat", "x", "y"},
			wantLen:   2,
			wantTypes: []PanelType{PanelChoose, PanelFilter},
		},
		"case insensitive type": {
			args:      []string{"CHOOSE", "a", "b", "--", "FILTER", "c", "d"},
			wantLen:   2,
			wantTypes: []PanelType{PanelChoose, PanelFilter},
		},
		"empty panel block returns error": {
			args:    []string{"choose"},
			wantErr: true,
		},
		"empty args returns error": {
			args:    []string{},
			wantErr: true,
		},
		"unknown type returns error": {
			args:    []string{"unknown", "a", "b"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parsePanelsFromArgs(tt.args, "\n", true)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePanelsFromArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("parsePanelsFromArgs() returned %d panels, want %d", len(got), tt.wantLen)
				return
			}
			for i, p := range got {
				if p.Type != tt.wantTypes[i] {
					t.Errorf("Panel[%d].Type = %v, want %v", i, p.Type, tt.wantTypes[i])
				}
			}
		})
	}
}

func TestPanelTypeConstants(t *testing.T) {
	if PanelChoose != "choose" {
		t.Errorf("PanelChoose = %v, want 'choose'", PanelChoose)
	}
	if PanelFilter != "filter" {
		t.Errorf("PanelFilter = %v, want 'filter'", PanelFilter)
	}
}

func TestInitModelsConsistency(t *testing.T) {
	makeChoosePanel := func(items ...string) Panel {
		opts, _ := parseChooseBlock(items)
		return Panel{Type: PanelChoose, ChooseOpts: opts}
	}
	makeFilterPanel := func(items ...string) Panel {
		opts, _ := parseFilterBlock(items)
		return Panel{Type: PanelFilter, FilterOpts: opts}
	}

	tests := map[string]struct {
		panels    []Panel
		wantItems []int // expected item count per panel
		wantErr   bool
	}{
		"single choose panel": {
			panels:    []Panel{makeChoosePanel("a", "b", "c")},
			wantItems: []int{3},
		},
		"single filter panel": {
			panels:    []Panel{makeFilterPanel("x", "y", "z")},
			wantItems: []int{3},
		},
		"multiple panels": {
			panels:    []Panel{makeChoosePanel("a", "b"), makeFilterPanel("x", "y")},
			wantItems: []int{2, 2},
		},
		"three panels mixed": {
			panels:    []Panel{makeChoosePanel("a"), makeFilterPanel("b", "c"), makeChoosePanel("d", "e", "f")},
			wantItems: []int{1, 2, 3},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m := &orchestrator{panels: tt.panels}
			err := m.initModels(Options{Height: 10})
			if (err != nil) != tt.wantErr {
				t.Errorf("initModels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				totalModels := len(m.chooseModels) + len(m.filterModels)
				if len(m.panels) != totalModels {
					t.Errorf("panel count mismatch: panels=%d models=%d", len(m.panels), totalModels)
				}
				for i, panel := range m.panels {
					switch panel.Type {
					case PanelChoose:
						got := len(m.chooseModels[panel.ModelIdx].items)
						if got != tt.wantItems[i] {
							t.Errorf("panel %d (choose): expected %d items, got %d", i, tt.wantItems[i], got)
						}
					case PanelFilter:
						got := len(m.filterModels[panel.ModelIdx].filteringChoices)
						if got != tt.wantItems[i] {
							t.Errorf("panel %d (filter): expected %d items, got %d", i, tt.wantItems[i], got)
						}
					}
				}
			}
		})
	}
}

func TestOptionsBorderStyle(t *testing.T) {
	tests := map[string]struct {
		border  string
		wantTop string
	}{
		"single border": {
			border:  "single",
			wantTop: "─",
		},
		"double border": {
			border:  "double",
			wantTop: "═",
		},
		"rounded border": {
			border:  "rounded",
			wantTop: "─",
		},
		"none border": {
			border:  "none",
			wantTop: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			o := Options{Border: tt.border}
			style := o.BorderStyle()
			_ = style
		})
	}
}

func TestDefaultPanelKeymap(t *testing.T) {
	km := defaultPanelKeymap()

	if km.NextPanel.Help().Key == "" {
		t.Error("NextPanel keybinding should have help text")
	}
	if km.PrevPanel.Help().Key == "" {
		t.Error("PrevPanel keybinding should have help text")
	}
	if km.Submit.Help().Key == "" {
		t.Error("Submit keybinding should have help text")
	}
	if km.Quit.Help().Key == "" {
		t.Error("Quit keybinding should have help text")
	}
}

func TestPanelStruct(t *testing.T) {
	opts, err := parseChooseBlock([]string{"item1", "item2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := Panel{Type: PanelChoose, ChooseOpts: opts}

	if p.Type != PanelChoose {
		t.Errorf("Panel.Type = %v, want %v", p.Type, PanelChoose)
	}
	if len(p.ChooseOpts.Options) != 2 {
		t.Errorf("Panel.ChooseOpts.Options length = %d, want 2", len(p.ChooseOpts.Options))
	}
}

func newMockChooseModel(items []string, height int) *chooseModel {
	chooseItems := make([]chooseItem, len(items))
	for i, item := range items {
		chooseItems[i] = chooseItem{Text: item, Selected: false, Order: 0}
	}

	pager := paginator.New()
	pager.SetTotalPages((len(items) + height - 1) / height)
	pager.PerPage = height
	pager.Type = paginator.Dots

	km := defaultChooseKeymap()
	km.Toggle.SetEnabled(true)
	km.ToggleAll.SetEnabled(true)

	return &chooseModel{
		index:             0,
		currentOrder:      0,
		height:            height,
		padding:           []int{0, 0, 0, 0},
		cursor:            "> ",
		header:            "",
		selectedPrefix:    "✓ ",
		unselectedPrefix:  "• ",
		cursorPrefix:      "• ",
		items:             chooseItems,
		limit:             10,
		numSelected:       0,
		paginator:         pager,
		showHelp:          false,
		keymap:            km,
		cursorStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		headerStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("99")),
		itemStyle:         lipgloss.NewStyle(),
		selectedItemStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		quitting:          false,
		submitted:         false,
	}
}

func newMockFilterModel(items []string, height int, fuzzy bool) *filterModel {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = "> "
	ti.Placeholder = "Filter..."

	choices := map[string]string{}
	filteringChoices := []string{}
	for _, opt := range items {
		choices[opt] = opt
		filteringChoices = append(filteringChoices, opt)
	}

	matches := matchAll(filteringChoices)

	fkm := defaultFilterKeymap()
	fkm.Toggle.SetEnabled(true)
	fkm.ToggleAndPrevious.SetEnabled(true)
	fkm.ToggleAndNext.SetEnabled(true)
	fkm.ToggleAll.SetEnabled(true)

	return &filterModel{
		textinput:             ti,
		choices:               choices,
		filteringChoices:      filteringChoices,
		matches:               matches,
		cursor:                0,
		header:                "",
		selected:              make(map[string]struct{}),
		limit:                 10,
		numSelected:           0,
		indicator:             "•",
		selectedPrefix:        " ◉ ",
		unselectedPrefix:      " ○ ",
		height:                height,
		padding:               []int{0, 0, 0, 0},
		quitting:              false,
		headerStyle:           lipgloss.NewStyle().Foreground(lipgloss.Color("99")),
		matchStyle:            lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		textStyle:             lipgloss.NewStyle(),
		cursorTextStyle:       lipgloss.NewStyle(),
		indicatorStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		selectedPrefixStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		unselectedPrefixStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		reverse:               false,
		fuzzy:                 fuzzy,
		sort:                  fuzzy,
		showHelp:              false,
		keymap:                fkm,
		submitted:             false,
	}
}

func TestRenderChooseView_AllItems(t *testing.T) {
	tests := []struct {
		name          string
		items         []string
		height        int
		expectedCount int
	}{
		{
			name:          "short names",
			items:         []string{"a", "b", "c"},
			height:        10,
			expectedCount: 3,
		},
		{
			name:          "many items",
			items:         []string{"item1", "item2", "item3", "item4", "item5"},
			height:        10,
			expectedCount: 5,
		},
		{
			name:          "exactly height",
			items:         []string{"a", "b", "c", "d", "e"},
			height:        5,
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := newMockChooseModel(tt.items, tt.height)
			o := orchestrator{
				chooseModels: []chooseModel{*cm},
				panels:       []Panel{{Type: PanelChoose, ModelIdx: 0}},
			}

			view := o.renderChooseView(cm)

			for _, item := range tt.items {
				if !strings.Contains(view, item) {
					t.Errorf("View should contain item %q, got:\n%s", item, view)
				}
			}

			if tt.height < len(tt.items) {
				if !strings.Contains(view, "•") {
					t.Error("View should contain pagination dots when items exceed height")
				}
			}
		})
	}
}

func TestRenderFilterView_AllItems(t *testing.T) {
	tests := []struct {
		name          string
		items         []string
		height        int
		expectedCount int
	}{
		{
			name:          "short names",
			items:         []string{"apple", "banana", "cherry"},
			height:        10,
			expectedCount: 3,
		},
		{
			name:          "many items",
			items:         []string{"one", "two", "three", "four", "five"},
			height:        10,
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := newMockFilterModel(tt.items, tt.height, true)
			o := orchestrator{
				filterModels: []filterModel{*fm},
				panels:       []Panel{{Type: PanelFilter, ModelIdx: 0}},
			}

			view := o.renderFilterView(fm)

			for _, item := range tt.items {
				if !strings.Contains(view, item) {
					t.Errorf("View should contain item %q, got:\n%s", item, view)
				}
			}

			if !strings.Contains(view, "Filter...") && !strings.Contains(view, "> ") {
				t.Error("View should contain filter input")
			}
		})
	}
}

func TestRenderMixedPanels(t *testing.T) {
	panels := []Panel{
		{Type: PanelChoose, ModelIdx: 0},
		{Type: PanelFilter, ModelIdx: 0},
	}

	chooseItems := []chooseItem{
		{Text: "a", Selected: false}, {Text: "b", Selected: false}, {Text: "c", Selected: false},
	}

	o := orchestrator{
		panels:       panels,
		chooseModels: []chooseModel{{items: chooseItems}},
		filterModels: []filterModel{},
		activeIdx:    0,
		borderStyle:  lipgloss.NewStyle().BorderStyle(lipgloss.Border{}),
		headerStyle:  lipgloss.NewStyle(),
	}

	view := o.View()

	if !strings.Contains(view, "CHOOSE") {
		t.Error("View should contain CHOOSE panel type")
	}
}

func TestRenderItemsWithSpaces(t *testing.T) {
	cm := newMockChooseModel([]string{"hello world", "foo bar", "baz qux"}, 10)
	o := orchestrator{
		chooseModels: []chooseModel{*cm},
		panels:       []Panel{{Type: PanelChoose, ModelIdx: 0}},
	}

	view := o.renderChooseView(cm)

	items := []string{"hello world", "foo bar", "baz qux"}
	for _, item := range items {
		if !strings.Contains(view, item) {
			t.Errorf("View should contain item with space %q, got:\n%s", item, view)
		}
	}
}

func TestRenderUnicodeItems(t *testing.T) {
	tests := []struct {
		name  string
		items []string
	}{
		{
			name:  "greek letters",
			items: []string{"α", "β", "γ"},
		},
		{
			name:  "japanese",
			items: []string{"日本語", "テスト", "ケース"},
		},
		{
			name:  "emoji",
			items: []string{"😀", "🎉", "🚀"},
		},
		{
			name:  "mixed unicode",
			items: []string{"café", "naïve", "résumé"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := newMockChooseModel(tt.items, 10)
			o := orchestrator{
				chooseModels: []chooseModel{*cm},
				panels:       []Panel{{Type: PanelChoose, ModelIdx: 0}},
			}

			view := o.renderChooseView(cm)

			for _, item := range tt.items {
				if !strings.Contains(view, item) {
					t.Errorf("View should contain unicode item %q, got:\n%s", item, view)
				}
			}
		})
	}
}

func TestRenderSpecialCharacterItems(t *testing.T) {
	tests := []struct {
		name  string
		items []string
	}{
		{
			name:  "special chars",
			items: []string{"test<tag>", "foo@bar", "baz#qux"},
		},
		{
			name:  "brackets and parens",
			items: []string{"(one)", "[two]", "{three}"},
		},
		{
			name:  "quotes",
			items: []string{`"quoted"`, "'single'", "`code`"},
		},
		{
			name:  "path-like",
			items: []string{"/usr/bin", "./script.sh", "C:\\Windows"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := newMockChooseModel(tt.items, 10)
			o := orchestrator{
				chooseModels: []chooseModel{*cm},
				panels:       []Panel{{Type: PanelChoose, ModelIdx: 0}},
			}

			view := o.renderChooseView(cm)

			for _, item := range tt.items {
				if !strings.Contains(view, item) {
					t.Errorf("View should contain special char item %q, got:\n%s", item, view)
				}
			}
		})
	}
}

func TestRenderVerticalLayout(t *testing.T) {
	cm := newMockChooseModel([]string{"a", "b", "c"}, 10)
	fm := newMockFilterModel([]string{"x", "y", "z"}, 10, true)

	o := orchestrator{
		panels:            []Panel{{Type: PanelChoose}, {Type: PanelFilter}},
		chooseModels:      []chooseModel{*cm},
		filterModels:      []filterModel{*fm},
		activeIdx:         0,
		vertical:          true,
		borderStyle:       lipgloss.NewStyle().BorderStyle(lipgloss.Border{}),
		headerStyle:       lipgloss.NewStyle(),
		activeBorderStyle: lipgloss.NewStyle(),
	}

	view := o.View()

	if !strings.Contains(view, "CHOOSE") || !strings.Contains(view, "FILTER") {
		t.Error("View should contain both panel types")
	}
}

func TestRenderHorizontalLayout(t *testing.T) {
	cm := newMockChooseModel([]string{"a", "b", "c"}, 10)
	fm := newMockFilterModel([]string{"x", "y", "z"}, 10, true)

	o := orchestrator{
		panels:            []Panel{{Type: PanelChoose}, {Type: PanelFilter}},
		chooseModels:      []chooseModel{*cm},
		filterModels:      []filterModel{*fm},
		activeIdx:         0,
		vertical:          false,
		gap:               1,
		borderStyle:       lipgloss.NewStyle().BorderStyle(lipgloss.Border{}),
		headerStyle:       lipgloss.NewStyle(),
		activeBorderStyle: lipgloss.NewStyle(),
	}

	view := o.View()

	if !strings.Contains(view, "CHOOSE") || !strings.Contains(view, "FILTER") {
		t.Error("View should contain both panel types")
	}
}

func TestRenderPagination(t *testing.T) {
	items := make([]string, 20)
	for i := range items {
		items[i] = string(rune('a' + i))
	}

	cm := newMockChooseModel(items, 5)

	if cm.paginator.TotalPages != 4 {
		t.Errorf("Expected 4 pages for 20 items with height 5, got %d", cm.paginator.TotalPages)
	}

	cm.paginator.Page = 1
	o := orchestrator{
		chooseModels: []chooseModel{*cm},
		panels:       []Panel{{Type: PanelChoose, ModelIdx: 0}},
	}

	view := o.renderChooseView(cm)

	if !strings.Contains(view, "•") {
		t.Error("View should contain pagination indicator")
	}
}

// TestRenderActiveInactiveBorder tests the active/inactive border styling
func TestRenderActiveInactiveBorder(t *testing.T) {
	cm := newMockChooseModel([]string{"a", "b", "c"}, 10)
	fm := newMockFilterModel([]string{"x", "y", "z"}, 10, true)

	o := orchestrator{
		panels:              []Panel{{Type: PanelChoose}, {Type: PanelFilter}},
		chooseModels:        []chooseModel{*cm},
		filterModels:        []filterModel{*fm},
		activeIdx:           0,
		borderStyle:         lipgloss.NewStyle(),
		activeBorderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		inactiveBorderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		headerStyle:         lipgloss.NewStyle(),
	}

	view := o.View()

	if !strings.Contains(view, "●") {
		t.Error("View should contain active panel indicator")
	}
}

// TestChooseCustomPrefixes tests that custom prefix values are used in choose panels
func TestChooseCustomPrefixes(t *testing.T) {
	cm := newMockChooseModel([]string{"a", "b", "c"}, 10)
	cm.cursor = ">> "
	cm.selectedPrefix = "[X] "
	cm.unselectedPrefix = "[ ] "
	cm.cursorPrefix = ">> "
	cm.selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green"))
	cm.cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("blue"))
	cm.itemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
	// Select first item to test selectedPrefix
	cm.items[0].Selected = true
	cm.numSelected = 1

	o := orchestrator{
		panels:       []Panel{{Type: PanelChoose, ModelIdx: 0}},
		chooseModels: []chooseModel{*cm},
		showHelp:     false,
		borderStyle:  lipgloss.NewStyle(),
		headerStyle:  lipgloss.NewStyle(),
	}

	view := o.View()

	if !strings.Contains(view, ">> ") {
		t.Error("View should contain custom cursor prefix '>> '")
	}
	if !strings.Contains(view, "[X] ") {
		t.Error("View should contain custom selected prefix '[X] '")
	}
	if !strings.Contains(view, "[ ] ") {
		t.Error("View should contain custom unselected prefix '[ ] '")
	}
}

// TestFilterCustomStyles tests that custom styles are applied to filter panels
func TestFilterCustomStyles(t *testing.T) {
	fm := newMockFilterModel([]string{"apple", "banana"}, 10, true)
	fm.indicator = "→"
	fm.selectedPrefix = "[*] "
	fm.unselectedPrefix = "[ ] "
	fm.indicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("cyan"))
	fm.textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("white"))
	fm.cursorTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow")).Bold(true)
	fm.selectedPrefixStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("green"))
	fm.unselectedPrefixStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("gray"))
	fm.matchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("red")).Bold(true)
	// Select first item to test selectedPrefix
	fm.selected["apple"] = struct{}{}
	fm.numSelected = 1

	o := orchestrator{
		panels:       []Panel{{Type: PanelFilter, ModelIdx: 0}},
		filterModels: []filterModel{*fm},
		showHelp:     false,
		borderStyle:  lipgloss.NewStyle(),
		headerStyle:  lipgloss.NewStyle(),
	}

	view := o.View()

	if !strings.Contains(view, "→") {
		t.Error("View should contain custom indicator '→'")
	}
	if !strings.Contains(view, "[*] ") {
		t.Error("View should contain custom selected prefix '[*] '")
	}
	if !strings.Contains(view, "[ ] ") {
		t.Error("View should contain custom unselected prefix '[ ] '")
	}
}

// TestHelpView tests that help is rendered properly
func TestHelpView(t *testing.T) {
	cm := newMockChooseModel([]string{"a", "b", "c"}, 10)

	o := orchestrator{
		panels:       []Panel{{Type: PanelChoose, ModelIdx: 0}},
		chooseModels: []chooseModel{*cm},
		showHelp:     true,
		borderStyle:  lipgloss.NewStyle(),
		headerStyle:  lipgloss.NewStyle(),
		keymap:       defaultPanelKeymap(),
		help:         help.New(),
	}

	view := o.View()

	// Help should be present - check for some common help keys
	if !strings.Contains(view, "tab") && !strings.Contains(view, "enter") {
		t.Error("View should contain help text with key bindings")
	}
}

// TestPanelKeymapHelpInterface tests that panelKeymap implements help.KeyMap
func TestPanelKeymapHelpInterface(t *testing.T) {
	km := defaultPanelKeymap()

	// Test FullHelp
	fullHelp := km.FullHelp()
	if len(fullHelp) == 0 {
		t.Error("FullHelp should return non-empty help lines")
	}

	// Test ShortHelp
	shortHelp := km.ShortHelp()
	if len(shortHelp) == 0 {
		t.Error("ShortHelp should return non-empty help bindings")
	}
}

// TestTextInputStyles tests that text input styles are properly set
func TestTextInputStyles(t *testing.T) {
	fm := newMockFilterModel([]string{"test"}, 10, true)
	fm.textinput.Prompt = ">>> "
	fm.textinput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("magenta"))
	fm.textinput.Placeholder = "Search... "
	fm.textinput.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("darkgray"))

	o := orchestrator{
		panels:       []Panel{{Type: PanelFilter, ModelIdx: 0}},
		filterModels: []filterModel{*fm},
		showHelp:     false,
		borderStyle:  lipgloss.NewStyle(),
		headerStyle:  lipgloss.NewStyle(),
	}

	view := o.View()

	// The prompt should be rendered in the view
	if !strings.Contains(view, ">>> ") {
		t.Error("View should contain custom prompt '>>> '")
	}
}

// TestMultipleStylesInOrchestrator tests that all style options work together
func TestMultipleStylesInOrchestrator(t *testing.T) {
	items := []string{"one", "two", "three"}

	cm := newMockChooseModel(items, 10)
	cm.selectedPrefix = "✓ "
	cm.unselectedPrefix = "○ "
	cm.cursorPrefix = "→ "
	cm.cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	cm.selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	cm.itemStyle = lipgloss.NewStyle()

	fm := newMockFilterModel(items, 10, true)
	fm.indicator = "•"
	fm.selectedPrefix = "◉ "
	fm.unselectedPrefix = "○ "
	fm.textStyle = lipgloss.NewStyle()
	fm.cursorTextStyle = lipgloss.NewStyle()
	fm.indicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels:        []chooseModel{*cm},
		filterModels:        []filterModel{*fm},
		showHelp:            false,
		borderStyle:         lipgloss.NewStyle(),
		activeBorderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		inactiveBorderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		headerStyle:         lipgloss.NewStyle(),
		activeIdx:           0,
	}

	view := o.View()

	// Both panels should be present
	if !strings.Contains(view, "CHOOSE") {
		t.Error("View should contain CHOOSE panel header")
	}
	if !strings.Contains(view, "FILTER") {
		t.Error("View should contain FILTER panel header")
	}

	// Items should be in both panels
	for _, item := range items {
		count := strings.Count(view, item)
		if count < 2 {
			t.Errorf("Item %q should appear at least twice (in both panels), found %d times", item, count)
		}
	}
}

func TestFilterTypingText(t *testing.T) {
	fm := newMockFilterModel([]string{"apple", "banana", "cherry", "date"}, 10, true)

	fm.textinput.SetValue("ban")
	fm.matches = fuzzy.FindNoSort("ban", fm.filteringChoices)

	if len(fm.matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(fm.matches))
	}
	if fm.matches[0].Str != "banana" {
		t.Errorf("expected 'banana', got %s", fm.matches[0].Str)
	}
}

func TestFilterFuzzyMatch(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		items    []string
		expected []string
	}{
		{
			name:     "partial match",
			query:    "app",
			items:    []string{"apple", "application", "banana"},
			expected: []string{"apple", "application"},
		},
		{
			name:     "single char",
			query:    "b",
			items:    []string{"apple", "banana", "cherry"},
			expected: []string{"banana"},
		},
		{
			name:     "no match",
			query:    "xyz",
			items:    []string{"apple", "banana", "cherry"},
			expected: nil,
		},
		{
			name:     "case insensitive",
			query:    "APP",
			items:    []string{"Apple", "APPLE", "banana"},
			expected: []string{"Apple", "APPLE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := newMockFilterModel(tt.items, 10, true)
			fm.textinput.SetValue(tt.query)
			fm.matches = fuzzy.FindNoSort(tt.query, fm.filteringChoices)

			if tt.expected == nil {
				if len(fm.matches) != 0 {
					t.Errorf("expected no matches, got %d", len(fm.matches))
				}
				return
			}

			if len(fm.matches) != len(tt.expected) {
				t.Errorf("expected %d matches, got %d", len(tt.expected), len(fm.matches))
				return
			}

			for i, expected := range tt.expected {
				if fm.matches[i].Str != expected {
					t.Errorf("match[%d] = %q, want %q", i, fm.matches[i].Str, expected)
				}
			}
		})
	}
}

func TestFilterExactMatch(t *testing.T) {
	fm := newMockFilterModel([]string{"apple", "banana", "apple pie"}, 10, false)

	fm.textinput.SetValue("apple")
	fm.matches = exactMatches("apple", fm.filteringChoices)

	if len(fm.matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(fm.matches))
	}

	matchStrs := make([]string, len(fm.matches))
	for i, m := range fm.matches {
		matchStrs[i] = m.Str
	}
	if !strings.Contains(strings.Join(matchStrs, ","), "apple") {
		t.Error("should match both 'apple' and 'apple pie'")
	}
}

func TestFilterNoMatch(t *testing.T) {
	fm := newMockFilterModel([]string{"apple", "banana", "cherry"}, 10, true)

	fm.textinput.SetValue("xyz")
	fm.matches = fuzzy.FindNoSort("xyz", fm.filteringChoices)

	if len(fm.matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(fm.matches))
	}
}

func TestFilterClearText(t *testing.T) {
	fm := newMockFilterModel([]string{"apple", "banana", "cherry"}, 10, true)

	fm.textinput.SetValue("ban")
	fm.matches = fuzzy.FindNoSort("ban", fm.filteringChoices)
	if len(fm.matches) != 1 {
		t.Errorf("expected 1 match after typing, got %d", len(fm.matches))
	}

	fm.textinput.SetValue("")
	fm.matches = matchAll(fm.filteringChoices)
	if len(fm.matches) != 3 {
		t.Errorf("expected 3 matches after clearing, got %d", len(fm.matches))
	}
}

func TestFilterMultipleCharacters(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		items    []string
		wantLen  int
		firstStr string
	}{
		{
			name:     "three chars",
			query:    "ana",
			items:    []string{"banana", "canada", "china"},
			wantLen:  2,
			firstStr: "banana",
		},
		{
			name:     "full word",
			query:    "banana",
			items:    []string{"banana", "banana split", "bananaland"},
			wantLen:  3,
			firstStr: "banana",
		},
		{
			name:     "space in query",
			query:    "banana split",
			items:    []string{"banana", "banana split", "split"},
			wantLen:  1,
			firstStr: "banana split",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := newMockFilterModel(tt.items, 10, true)
			fm.textinput.SetValue(tt.query)
			fm.matches = fuzzy.FindNoSort(tt.query, fm.filteringChoices)

			if len(fm.matches) != tt.wantLen {
				t.Errorf("expected %d matches, got %d", tt.wantLen, len(fm.matches))
			}
			if tt.wantLen > 0 && fm.matches[0].Str != tt.firstStr {
				t.Errorf("first match = %q, want %q", fm.matches[0].Str, tt.firstStr)
			}
		})
	}
}

func TestFilterFocusAwareness(t *testing.T) {
	fm := newMockFilterModel([]string{"apple", "banana", "cherry"}, 10, true)

	if !fm.textinput.Focused() {
		t.Error("Filter input should be focused by default")
	}

	fm.textinput.Blur()
	if fm.textinput.Focused() {
		t.Error("Filter input should not be focused after blur")
	}

	fm.textinput.Focus()
	if !fm.textinput.Focused() {
		t.Error("Filter input should be focused after focus")
	}
}

func TestChooseCursorNavigation(t *testing.T) {
	cm := newMockChooseModel([]string{"a", "b", "c"}, 10)
	cm.index = 1

	msg := tea.KeyMsg{Type: tea.KeySpace}
	m := &orchestrator{}
	m.updateChooseModel(cm, msg)

	if !cm.items[1].Selected {
		t.Error("Item at cursor should be selected after space")
	}
	if cm.numSelected != 1 {
		t.Errorf("numSelected = %d, want 1", cm.numSelected)
	}

	m.updateChooseModel(cm, msg)
	if cm.items[1].Selected {
		t.Error("Item should be deselected after second space")
	}
}

func TestChoosePagination(t *testing.T) {
	items := make([]string, 15)
	for i := range items {
		items[i] = string(rune('a' + i))
	}
	cm := newMockChooseModel(items, 5)

	initialPage := cm.paginator.Page
	msg := tea.KeyMsg{Type: tea.KeyEnd}
	m := &orchestrator{}
	m.updateChooseModel(cm, msg)

	if cm.paginator.Page == initialPage {
		t.Error("Page should change after End key")
	}
}

func TestGetResults(t *testing.T) {
	cm := chooseModel{
		items: []chooseItem{
			{Text: "a", Selected: true},
			{Text: "b", Selected: false},
			{Text: "c", Selected: true},
		},
	}

	fm := filterModel{
		selected: map[string]struct{}{
			"x": {},
			"y": {},
		},
	}

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels: []chooseModel{cm},
		filterModels: []filterModel{fm},
	}

	results := o.getResults("|")

	// getResults returns one string per panel
	if len(results) != 2 {
		t.Errorf("expected 2 results (one per panel), got %d: %v", len(results), results)
		return
	}
	// Choose panel: a and c selected, joined by |
	if results[0] != "a|c" {
		t.Errorf("choose panel result = %q, want 'a|c'", results[0])
	}
	// Filter panel: x and y selected (order may vary), joined by |
	parts := strings.Split(results[1], "|")
	if len(parts) != 2 {
		t.Errorf("filter panel result = %q, expected 2 items joined by |", results[1])
	}
}

func TestViewEmpty(t *testing.T) {
	o := orchestrator{
		quitting: true,
	}

	view := o.View()
	if view != "" {
		t.Errorf("View should return empty string when quitting, got %q", view)
	}

	o.quitting = false
	o.panels = []Panel{}

	view = o.View()
	if view != "" {
		t.Errorf("View should return empty string with no panels, got %q", view)
	}
}

func TestHandleSubmit_FilterWithFilteredMatches(t *testing.T) {
	items := []string{"gg", "hh", "jgd", "77 volu slo domu"}
	fm := newMockFilterModel(items, 10, true)
	fm.limit = 1

	fm.textinput.SetValue("77")
	fm.matches = fuzzy.FindNoSort("77", fm.filteringChoices)

	if len(fm.matches) != 1 {
		t.Fatalf("expected 1 match after filtering '77', got %d", len(fm.matches))
	}
	if fm.matches[0].Str != "77 volu slo domu" {
		t.Fatalf("expected filtered match '77 volu slo domu', got %q", fm.matches[0].Str)
	}

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels: []chooseModel{},
		filterModels: []filterModel{*fm},
		activeIdx:    1,
	}

	o, _ = o.handleSubmit()

	if len(fm.selected) != 1 {
		t.Errorf("expected 1 selected item, got %d", len(fm.selected))
	}

	expectedSelected := "77 volu slo domu"
	if _, ok := fm.selected[expectedSelected]; !ok {
		t.Errorf("expected %q to be selected, got selected=%v", expectedSelected, fm.selected)
	}
}

func TestGetResults_FilterWithFilteredMatches(t *testing.T) {
	items := []string{"gg", "hh", "jgd", "77 volu slo domu"}
	fm := newMockFilterModel(items, 10, true)

	fm.textinput.SetValue("77")
	fm.matches = fuzzy.FindNoSort("77", fm.filteringChoices)
	fm.selected["77 volu slo domu"] = struct{}{}
	fm.numSelected = 1

	cm := newMockChooseModel([]string{"a", "b", "c"}, 10)
	cm.items[0].Selected = true
	cm.numSelected = 1

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels: []chooseModel{*cm},
		filterModels: []filterModel{*fm},
	}

	results := o.getResults("\n")

	expected := []string{"a", "77 volu slo domu"}
	if len(results) != len(expected) {
		t.Errorf("expected %d results %v, got %d results %v", len(expected), expected, len(results), results)
	}

	found := false
	for _, r := range results {
		if r == "77 volu slo domu" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected '77 volu slo domu' in results, got %v", results)
	}
}

func TestHandleSubmit_FilterNoMatches(t *testing.T) {
	items := []string{"gg", "hh", "jgd", "77 volu slo domu"}
	fm := newMockFilterModel(items, 10, true)

	fm.textinput.SetValue("xyz")
	fm.matches = fuzzy.FindNoSort("xyz", fm.filteringChoices)

	if len(fm.matches) != 0 {
		t.Fatalf("expected 0 matches for 'xyz', got %d", len(fm.matches))
	}

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels: []chooseModel{},
		filterModels: []filterModel{*fm},
		activeIdx:    1,
	}

	o, _ = o.handleSubmit()

	if len(fm.selected) != 0 {
		t.Errorf("expected 0 selected items when no matches, got %d", len(fm.selected))
	}
}

func TestHandleSubmit_FilterWithRandomMessage(t *testing.T) {
	items := generateRandomMessages(5)
	fm := newMockFilterModel(items, 10, true)
	fm.limit = 1

	msg := generateRandomMessage()
	fm.textinput.SetValue(msg)
	fm.matches = fuzzy.FindNoSort(msg, fm.filteringChoices)

	if len(fm.matches) == 0 {
		t.Skipf("no matches found for random message %q, skipping test", msg)
	}

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels: []chooseModel{},
		filterModels: []filterModel{*fm},
		activeIdx:    1,
	}

	o, _ = o.handleSubmit()

	expectedMatch := fm.matches[0].Str
	if _, ok := fm.selected[expectedMatch]; !ok {
		t.Errorf("expected %q to be selected, got selected=%v", expectedMatch, fm.selected)
	}
}

func TestGetResults_WithRandomMessages(t *testing.T) {
	chooseItems := generateRandomMessages(3)
	filterItems := generateRandomMessages(4)

	cm := newMockChooseModel(chooseItems, 10)
	cm.items[0].Selected = true
	cm.numSelected = 1

	fm := newMockFilterModel(filterItems, 10, true)
	randMsg := generateRandomMessage()
	fm.textinput.SetValue(randMsg)
	fm.matches = fuzzy.FindNoSort(randMsg, fm.filteringChoices)

	if len(fm.matches) > 0 {
		fm.selected[fm.matches[0].Str] = struct{}{}
		fm.numSelected = 1
	} else {
		t.Skipf("no matches for random filter query %q, skipping", randMsg)
	}

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels: []chooseModel{*cm},
		filterModels: []filterModel{*fm},
	}

	results := o.getResults("\n")

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d: %v", len(results), results)
	}

	found := false
	for _, r := range results {
		if r == fm.matches[0].Str {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected filtered item %q in results, got %v", fm.matches[0].Str, results)
	}
}

func TestView_ChooseAndFilterItemsDisplayed(t *testing.T) {
	chooseItems := generateRandomMessages(3)
	filterItems := generateRandomMessages(4)

	cm := newMockChooseModel(chooseItems, 10)
	fm := newMockFilterModel(filterItems, 10, true)

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
		},
		chooseModels: []chooseModel{*cm},
		filterModels: []filterModel{*fm},
		showHelp:     false,
	}

	view := o.View()

	for _, item := range chooseItems {
		if !strings.Contains(view, item) {
			t.Errorf("View should contain choose item %q, got:\n%s", item, view)
		}
	}

	for _, item := range filterItems {
		if !strings.Contains(view, item) {
			t.Errorf("View should contain filter item %q, got:\n%s", item, view)
		}
	}
}

func TestHandleSubmit_RandomMessagesMultiplePanels(t *testing.T) {
	panel1Items := generateRandomMessages(3)
	panel2Items := generateRandomMessages(3)
	panel3Items := generateRandomMessages(3)

	cm1 := newMockChooseModel(panel1Items, 10)
	fm := newMockFilterModel(panel2Items, 10, true)
	cm2 := newMockChooseModel(panel3Items, 10)

	o := orchestrator{
		panels: []Panel{
			{Type: PanelChoose, ModelIdx: 0},
			{Type: PanelFilter, ModelIdx: 0},
			{Type: PanelChoose, ModelIdx: 1},
		},
		chooseModels: []chooseModel{*cm1, *cm2},
		filterModels: []filterModel{*fm},
		activeIdx:    0,
	}

	o, _ = o.handleSubmit()

	if o.activeIdx != 1 {
		t.Errorf("expected activeIdx=1 after submit on first panel, got %d", o.activeIdx)
	}

	o, _ = o.handleSubmit()

	if o.activeIdx != 2 {
		t.Errorf("expected activeIdx=2 after submit on second panel, got %d", o.activeIdx)
	}

	_, cmd := o.handleSubmit()

	if cmd == nil {
		t.Error("expected tea.Quit command on last panel submit")
	}

	if !o.submitted {
		t.Error("expected submitted=true after last panel submit")
	}
}

func TestView_ChooseAndFilterWithDifferentDelimiters(t *testing.T) {
	delimiters := []string{"---", ":::", "|||", "___", ";;;"}

	for i := 0; i < 3; i++ {
		delimiter := delimiters[rand.Intn(len(delimiters))]
		t.Run("delimiter="+delimiter, func(t *testing.T) {
			items := generateRandomMessagesWithDelimiter(5, delimiter)

			for _, item := range items {
				if strings.Contains(item, delimiter) && !strings.HasPrefix(item, `"`) {
					t.Errorf("item %q should not contain delimiter %q without quotes", item, delimiter)
				}
			}

			cm := newMockChooseModel(items, 10)
			fm := newMockFilterModel(items, 10, true)

			o := orchestrator{
				panels: []Panel{
					{Type: PanelChoose, ModelIdx: 0},
					{Type: PanelFilter, ModelIdx: 0},
				},
				chooseModels: []chooseModel{*cm},
				filterModels: []filterModel{*fm},
				showHelp:     false,
			}

			view := o.View()

			for _, item := range items {
				if !strings.Contains(view, item) {
					t.Errorf("View should contain item %q, got:\n%s", item, view)
				}
			}
		})
	}
}

func TestView_ChooseAndFilterWithForcedDelimiter(t *testing.T) {
	delimiters := []string{"---", ":::", "|||", "___", ";;;"}

	for i := 0; i < 3; i++ {
		delimiter := delimiters[rand.Intn(len(delimiters))]
		t.Run("delimiter="+delimiter, func(t *testing.T) {
			items := generateRandomMessagesWithDelimiterForcedItems(5, delimiter)

			if len(items) == 0 {
				t.Fatal("no items generated")
			}

			hasSingleChar := false
			hasQuotedDelimiter := false
			for _, item := range items {
				if len(item) == 1 {
					hasSingleChar = true
				}
				if item == `"`+delimiter+`"` {
					hasQuotedDelimiter = true
				}
			}

			if !hasSingleChar {
				t.Error("expected at least one single-character item")
			}
			if !hasQuotedDelimiter {
				t.Error("expected at least one item with quoted delimiter")
			}

			cm := newMockChooseModel(items, 10)
			fm := newMockFilterModel(items, 10, true)

			o := orchestrator{
				panels: []Panel{
					{Type: PanelChoose, ModelIdx: 0},
					{Type: PanelFilter, ModelIdx: 0},
				},
				chooseModels: []chooseModel{*cm},
				filterModels: []filterModel{*fm},
				showHelp:     false,
			}

			view := o.View()

			for _, item := range items {
				if !strings.Contains(view, item) {
					t.Errorf("View should contain item %q, got:\n%s", item, view)
				}
			}
		})
	}
}

func TestParseChooseBlock(t *testing.T) {
	t.Run("basic items", func(t *testing.T) {
		opts, err := parseChooseBlock([]string{"apple", "banana", "cherry"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts.Options) != 3 {
			t.Errorf("expected 3 items, got %d: %v", len(opts.Options), opts.Options)
		}
	})

	t.Run("with limit flag", func(t *testing.T) {
		opts, err := parseChooseBlock([]string{"--limit", "3", "a", "b", "c"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Limit != 3 {
			t.Errorf("expected Limit=3, got %d", opts.Limit)
		}
		if len(opts.Options) != 3 {
			t.Errorf("expected 3 items, got %d", len(opts.Options))
		}
	})

	t.Run("with header flag", func(t *testing.T) {
		opts, err := parseChooseBlock([]string{"--header", "Ovoce", "apple"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Header != "Ovoce" {
			t.Errorf("expected Header='Ovoce', got %q", opts.Header)
		}
	})

	t.Run("with no-limit flag", func(t *testing.T) {
		opts, err := parseChooseBlock([]string{"--no-limit", "a", "b"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !opts.NoLimit {
			t.Error("expected NoLimit=true")
		}
	})

	t.Run("invalid flag returns error", func(t *testing.T) {
		_, err := parseChooseBlock([]string{"--nonexistent-flag", "a"})
		if err == nil {
			t.Error("expected error for invalid flag, got nil")
		}
	})
}

func TestParseFilterBlock(t *testing.T) {
	t.Run("basic items", func(t *testing.T) {
		opts, err := parseFilterBlock([]string{"mango", "papaya"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opts.Options) != 2 {
			t.Errorf("expected 2 items, got %d", len(opts.Options))
		}
	})

	t.Run("with no-fuzzy flag", func(t *testing.T) {
		opts, err := parseFilterBlock([]string{"--no-fuzzy", "mango"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Fuzzy {
			t.Error("expected Fuzzy=false")
		}
	})

	t.Run("with placeholder flag", func(t *testing.T) {
		opts, err := parseFilterBlock([]string{"--placeholder", "Hledat...", "mango"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Placeholder != "Hledat..." {
			t.Errorf("expected Placeholder='Hledat...', got %q", opts.Placeholder)
		}
	})

	t.Run("with no-strict flag", func(t *testing.T) {
		opts, err := parseFilterBlock([]string{"--no-strict", "mango"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Strict {
			t.Error("expected Strict=false")
		}
	})
}

// newOptionsParser creates a kong parser for Options (for testing env var binding).
func newOptionsParser(o *Options) (*kong.Kong, error) {
	return kong.New(o, kong.Exit(func(int) {}), kong.Vars{
		"version":                 "test",
		"versionNumber":           "test",
		"defaultHeight":           "0",
		"defaultWidth":            "0",
		"defaultAlign":            "left",
		"defaultBorder":           "none",
		"defaultBorderForeground": "",
		"defaultBorderBackground": "",
		"defaultBackground":       "",
		"defaultForeground":       "",
		"defaultMargin":           "0 0",
		"defaultPadding":          "0 0",
		"defaultUnderline":        "false",
		"defaultBold":             "false",
		"defaultFaint":            "false",
		"defaultItalic":           "false",
		"defaultStrikethrough":    "false",
	})
}

// TestAllGlobalEnvVarsRespected verifies every GUM_PANEL_* env var is read by
// the Options struct. If a test fails the env var is declared but never bound.
func TestAllGlobalEnvVarsRespected(t *testing.T) {
	cases := []struct {
		env      string
		describe string
		check    func(o Options) bool
	}{
		{"GUM_PANEL_GAP=3", "Gap", func(o Options) bool { return o.Gap == 3 }},
		{"GUM_PANEL_HEIGHT=15", "Height", func(o Options) bool { return o.Height == 15 }},
		{"GUM_PANEL_BORDER=double", "Border", func(o Options) bool { return o.Border == "double" }},
		{"GUM_PANEL_VERTICAL=true", "Vertical", func(o Options) bool { return o.Vertical }},
		{"GUM_PANEL_STACKED=false", "Stacked", func(o Options) bool { return !o.Stacked }},
		{"GUM_PANEL_DELIMITER=|||", "Delimiter", func(o Options) bool { return o.Delimiter == "|||" }},
		{"GUM_PANEL_OUTPUT_DELIMITER=,", "OutputDelimiter", func(o Options) bool { return o.OutputDelimiter == "," }},
		{"GUM_PANEL_SINGLE=true", "Single", func(o Options) bool { return o.Single }},
		{"GUM_PANEL_ALL=true", "All", func(o Options) bool { return o.All }},
		{"GUM_PANEL_ACTIVE=1", "Active", func(o Options) bool { return o.Active == 1 }},
		{"GUM_PANEL_SHOW_HELP=false", "ShowHelp", func(o Options) bool { return !o.ShowHelp }},
		{"GUM_PANEL_ACTIVE_BORDER_FOREGROUND=196", "ActiveBorderStyle.Foreground", func(o Options) bool { return o.ActiveBorderStyle.Foreground == "196" }},
		{"GUM_PANEL_ACTIVE_BORDER_BACKGROUND=1", "ActiveBorderStyle.Background", func(o Options) bool { return o.ActiveBorderStyle.Background == "1" }},
		{"GUM_PANEL_INACTIVE_BORDER_FOREGROUND=33", "InactiveBorderStyle.Foreground", func(o Options) bool { return o.InactiveBorderStyle.Foreground == "33" }},
		{"GUM_PANEL_INACTIVE_BORDER_BACKGROUND=0", "InactiveBorderStyle.Background", func(o Options) bool { return o.InactiveBorderStyle.Background == "0" }},
		{"GUM_PANEL_ACTIVE_INDICATOR_FOREGROUND=82", "ActiveIndicatorStyle.Foreground", func(o Options) bool { return o.ActiveIndicatorStyle.Foreground == "82" }},
		{"GUM_PANEL_HEADER_FOREGROUND=99", "HeaderStyle.Foreground", func(o Options) bool { return o.HeaderStyle.Foreground == "99" }},
		{"GUM_PANEL_CURSOR_FOREGROUND=212", "CursorStyle.Foreground", func(o Options) bool { return o.CursorStyle.Foreground == "212" }},
		{"GUM_PANEL_ITEM_FOREGROUND=240", "ItemStyle.Foreground", func(o Options) bool { return o.ItemStyle.Foreground == "240" }},
		{"GUM_PANEL_SELECTED_FOREGROUND=46", "SelectedItemStyle.Foreground", func(o Options) bool { return o.SelectedItemStyle.Foreground == "46" }},
		{"GUM_PANEL_MATCH_FOREGROUND=11", "MatchStyle.Foreground", func(o Options) bool { return o.MatchStyle.Foreground == "11" }},
		{"GUM_PANEL_INDICATOR_FOREGROUND=10", "IndicatorStyle.Foreground", func(o Options) bool { return o.IndicatorStyle.Foreground == "10" }},
		{"GUM_PANEL_SELECTED_PREFIX_FOREGROUND=9", "SelectedPrefixStyle.Foreground", func(o Options) bool { return o.SelectedPrefixStyle.Foreground == "9" }},
		{"GUM_PANEL_UNSELECTED_PREFIX_FOREGROUND=8", "UnselectedPrefixStyle.Foreground", func(o Options) bool { return o.UnselectedPrefixStyle.Foreground == "8" }},
		{"GUM_PANEL_PROMPT_FOREGROUND=7", "PromptStyle.Foreground", func(o Options) bool { return o.PromptStyle.Foreground == "7" }},
		{"GUM_PANEL_PLACEHOLDER_FOREGROUND=6", "PlaceholderStyle.Foreground", func(o Options) bool { return o.PlaceholderStyle.Foreground == "6" }},
		{"GUM_PANEL_TEXT_FOREGROUND=5", "TextStyle.Foreground", func(o Options) bool { return o.TextStyle.Foreground == "5" }},
		{"GUM_PANEL_CURSOR_TEXT_FOREGROUND=4", "CursorTextStyle.Foreground", func(o Options) bool { return o.CursorTextStyle.Foreground == "4" }},
	}

	for _, tc := range cases {
		t.Run(tc.describe, func(t *testing.T) {
			parts := strings.SplitN(tc.env, "=", 2)
			t.Setenv(parts[0], parts[1])

			var o Options
			parser, err := newOptionsParser(&o)
			if err != nil {
				t.Fatalf("create parser: %v", err)
			}
			if _, err := parser.Parse([]string{"dummy"}); err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !tc.check(o) {
				t.Errorf("env var %s not reflected in Options", tc.env)
			}
		})
	}
}

// TestBorderStylesAppliedInView verifies that activeBorderStyle and
// inactiveBorderStyle foreground colors are actually used in View rendering,
// not hardcoded values.
func TestBorderStylesAppliedInView(t *testing.T) {
	// Force ANSI color rendering in test environment (no real terminal).
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	newOrch := func(activeColor, inactiveColor string) orchestrator {
		cm0 := newMockChooseModel([]string{"a"}, 5)
		cm1 := newMockChooseModel([]string{"b"}, 5)
		return orchestrator{
			panels: []Panel{
				{Type: PanelChoose, ModelIdx: 0},
				{Type: PanelChoose, ModelIdx: 1},
			},
			chooseModels:         []chooseModel{*cm0, *cm1},
			activeIdx:            0,
			borderStyle:          lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()),
			activeBorderStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color(activeColor)),
			inactiveBorderStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color(inactiveColor)),
			activeIndicatorStyle: lipgloss.NewStyle(),
			headerStyle:          lipgloss.NewStyle(),
		}
	}

	t.Run("active border uses activeBorderStyle foreground", func(t *testing.T) {
		o := newOrch("196", "240")
		view := o.View()
		if !strings.Contains(view, "\x1b[38;5;196m") {
			t.Errorf("active border color 196 not found in rendered view")
		}
	})

	t.Run("inactive border uses inactiveBorderStyle foreground", func(t *testing.T) {
		o := newOrch("212", "33")
		view := o.View()
		if !strings.Contains(view, "\x1b[38;5;33m") {
			t.Errorf("inactive border color 33 not found in rendered view")
		}
	})

	t.Run("hardcoded 240 not used when inactiveBorderStyle set to different color", func(t *testing.T) {
		o := newOrch("212", "196")
		view := o.View()
		// color 240 should NOT appear — it was the old hardcoded value
		if strings.Contains(view, "\x1b[38;5;240m") {
			t.Errorf("hardcoded color 240 found in view — inactiveBorderStyle is not being used")
		}
		if !strings.Contains(view, "\x1b[38;5;196m") {
			t.Errorf("inactive border color 196 not found in rendered view")
		}
	})
}

// TestEnvVarsRespectedBySubParsers verifies that GUM_FILTER_* and GUM_CHOOSE_*
// environment variables are picked up by the per-panel sub-parsers.
func TestEnvVarsRespectedBySubParsers(t *testing.T) {
	t.Run("GUM_FILTER_PLACEHOLDER respected", func(t *testing.T) {
		t.Setenv("GUM_FILTER_PLACEHOLDER", "Hledat ve filtru...")
		opts, err := parseFilterBlock([]string{"mango", "papaya"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Placeholder != "Hledat ve filtru..." {
			t.Errorf("expected Placeholder='Hledat ve filtru...', got %q", opts.Placeholder)
		}
	})

	t.Run("GUM_FILTER_HEADER respected", func(t *testing.T) {
		t.Setenv("GUM_FILTER_HEADER", "Moje ovoce")
		opts, err := parseFilterBlock([]string{"mango"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Header != "Moje ovoce" {
			t.Errorf("expected Header='Moje ovoce', got %q", opts.Header)
		}
	})

	t.Run("GUM_FILTER_* overridden by explicit per-panel flag", func(t *testing.T) {
		t.Setenv("GUM_FILTER_PLACEHOLDER", "from env")
		opts, err := parseFilterBlock([]string{"--placeholder", "from flag", "mango"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Placeholder != "from flag" {
			t.Errorf("explicit flag should override env var, got %q", opts.Placeholder)
		}
	})

	t.Run("GUM_CHOOSE_HEADER respected", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_HEADER", "Vyber ovoce")
		opts, err := parseChooseBlock([]string{"apple", "banana"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Header != "Vyber ovoce" {
			t.Errorf("expected Header='Vyber ovoce', got %q", opts.Header)
		}
	})

	t.Run("GUM_CHOOSE_* overridden by explicit per-panel flag", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_HEADER", "from env")
		opts, err := parseChooseBlock([]string{"--header", "from flag", "apple"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Header != "from flag" {
			t.Errorf("explicit flag should override env var, got %q", opts.Header)
		}
	})
}

// makeOrchestrator builds a minimal orchestrator with the given panels and calls
// initModels. Returns the initialised orchestrator or fails the test.
func makeOrchestrator(t *testing.T, panels []Panel, globalOpts Options) *orchestrator {
	t.Helper()
	if globalOpts.Height == 0 {
		globalOpts.Height = 10
	}
	m := &orchestrator{
		panels:               panels,
		activeBorderStyle:    globalOpts.ActiveBorderStyle.ToLipgloss(),
		inactiveBorderStyle:  globalOpts.InactiveBorderStyle.ToLipgloss(),
		activeIndicatorStyle: globalOpts.ActiveIndicatorStyle.ToLipgloss(),
		headerStyle:          globalOpts.HeaderStyle.ToLipgloss(),
		cursorStyle:          globalOpts.CursorStyle.ToLipgloss(),
		itemStyle:            globalOpts.ItemStyle.ToLipgloss(),
		selectedItemStyle:    globalOpts.SelectedItemStyle.ToLipgloss(),
		indicatorStyle:       globalOpts.IndicatorStyle.ToLipgloss(),
		selectedPrefixStyle:  globalOpts.SelectedPrefixStyle.ToLipgloss(),
		unselectedPrefixStyle: globalOpts.UnselectedPrefixStyle.ToLipgloss(),
		matchStyle:           globalOpts.MatchStyle.ToLipgloss(),
		textStyle:            globalOpts.TextStyle.ToLipgloss(),
		cursorTextStyle:      globalOpts.CursorTextStyle.ToLipgloss(),
		promptStyle:          globalOpts.PromptStyle.ToLipgloss(),
		placeholderStyle:     globalOpts.PlaceholderStyle.ToLipgloss(),
		borderStyle:          globalOpts.BorderStyle(),
	}
	if err := m.initModels(globalOpts); err != nil {
		t.Fatalf("initModels: %v", err)
	}
	return m
}

// TestPerPanelEnvVarsPropagateThroughToModels verifies that GUM_FILTER_* and
// GUM_CHOOSE_* env vars travel the full path:
//   env var → parseFilterBlock/parseChooseBlock → initModels → model field
//
// These are distinct from GUM_PANEL_* (global) — they set per-panel behaviour
// such as placeholder text, prompt, header, cursor prefix, fuzzy mode, etc.
func TestPerPanelEnvVarsPropagateThroughToModels(t *testing.T) {
	defaultOpts := Options{Height: 10}

	t.Run("GUM_FILTER_PLACEHOLDER sets filterModel textinput placeholder", func(t *testing.T) {
		t.Setenv("GUM_FILTER_PLACEHOLDER", "Hledat...")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if got := m.filterModels[0].textinput.Placeholder; got != "Hledat..." {
			t.Errorf("textinput.Placeholder = %q, want %q", got, "Hledat...")
		}
	})

	t.Run("GUM_FILTER_PROMPT sets filterModel textinput prompt", func(t *testing.T) {
		t.Setenv("GUM_FILTER_PROMPT", ">> ")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if got := m.filterModels[0].textinput.Prompt; got != ">> " {
			t.Errorf("textinput.Prompt = %q, want %q", got, ">> ")
		}
	})

	t.Run("GUM_FILTER_HEADER sets filterModel header", func(t *testing.T) {
		t.Setenv("GUM_FILTER_HEADER", "Vyber si")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if got := m.filterModels[0].header; got != "Vyber si" {
			t.Errorf("filterModel.header = %q, want %q", got, "Vyber si")
		}
	})

	t.Run("GUM_FILTER_INDICATOR sets filterModel indicator", func(t *testing.T) {
		t.Setenv("GUM_FILTER_INDICATOR", "→")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if got := m.filterModels[0].indicator; got != "→" {
			t.Errorf("filterModel.indicator = %q, want %q", got, "→")
		}
	})

	t.Run("GUM_FILTER_FUZZY=false disables fuzzy", func(t *testing.T) {
		t.Setenv("GUM_FILTER_FUZZY", "false")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if m.filterModels[0].fuzzy {
			t.Error("filterModel.fuzzy should be false when GUM_FILTER_FUZZY=false")
		}
	})

	t.Run("GUM_FILTER_SELECTED_PREFIX sets selectedPrefix", func(t *testing.T) {
		t.Setenv("GUM_FILTER_SELECTED_PREFIX", "[x] ")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if got := m.filterModels[0].selectedPrefix; got != "[x] " {
			t.Errorf("filterModel.selectedPrefix = %q, want %q", got, "[x] ")
		}
	})

	t.Run("GUM_FILTER_UNSELECTED_PREFIX sets unselectedPrefix", func(t *testing.T) {
		t.Setenv("GUM_FILTER_UNSELECTED_PREFIX", "[ ] ")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if got := m.filterModels[0].unselectedPrefix; got != "[ ] " {
			t.Errorf("filterModel.unselectedPrefix = %q, want %q", got, "[ ] ")
		}
	})

	t.Run("GUM_CHOOSE_HEADER sets chooseModel header", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_HEADER", "Vyber ovoce")
		copts, _ := parseChooseBlock([]string{"apple", "banana"})
		m := makeOrchestrator(t, []Panel{{Type: PanelChoose, ChooseOpts: copts}}, defaultOpts)
		if got := m.chooseModels[0].header; got != "Vyber ovoce" {
			t.Errorf("chooseModel.header = %q, want %q", got, "Vyber ovoce")
		}
	})

	t.Run("GUM_CHOOSE_CURSOR sets chooseModel cursor", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_CURSOR", "→ ")
		copts, _ := parseChooseBlock([]string{"apple", "banana"})
		m := makeOrchestrator(t, []Panel{{Type: PanelChoose, ChooseOpts: copts}}, defaultOpts)
		if got := m.chooseModels[0].cursor; got != "→ " {
			t.Errorf("chooseModel.cursor = %q, want %q", got, "→ ")
		}
	})

	t.Run("GUM_CHOOSE_SELECTED_PREFIX sets chooseModel selectedPrefix", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_SELECTED_PREFIX", "✓ ")
		copts, _ := parseChooseBlock([]string{"apple", "banana"})
		m := makeOrchestrator(t, []Panel{{Type: PanelChoose, ChooseOpts: copts}}, defaultOpts)
		if got := m.chooseModels[0].selectedPrefix; got != "✓ " {
			t.Errorf("chooseModel.selectedPrefix = %q, want %q", got, "✓ ")
		}
	})

	t.Run("GUM_CHOOSE_UNSELECTED_PREFIX sets chooseModel unselectedPrefix", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_UNSELECTED_PREFIX", "○ ")
		copts, _ := parseChooseBlock([]string{"apple", "banana"})
		m := makeOrchestrator(t, []Panel{{Type: PanelChoose, ChooseOpts: copts}}, defaultOpts)
		if got := m.chooseModels[0].unselectedPrefix; got != "○ " {
			t.Errorf("chooseModel.unselectedPrefix = %q, want %q", got, "○ ")
		}
	})

	t.Run("per-panel env vars independent per panel", func(t *testing.T) {
		// Both GUM_FILTER_* and GUM_CHOOSE_* set simultaneously
		t.Setenv("GUM_FILTER_PLACEHOLDER", "Filter here")
		t.Setenv("GUM_CHOOSE_HEADER", "Pick one")
		fopts, _ := parseFilterBlock([]string{"x", "y"})
		copts, _ := parseChooseBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{
			{Type: PanelFilter, FilterOpts: fopts},
			{Type: PanelChoose, ChooseOpts: copts},
		}, defaultOpts)
		if got := m.filterModels[0].textinput.Placeholder; got != "Filter here" {
			t.Errorf("filter placeholder = %q, want %q", got, "Filter here")
		}
		if got := m.chooseModels[0].header; got != "Pick one" {
			t.Errorf("choose header = %q, want %q", got, "Pick one")
		}
	})

	t.Run("explicit per-panel flag overrides env var in model", func(t *testing.T) {
		t.Setenv("GUM_FILTER_PLACEHOLDER", "from env")
		fopts, _ := parseFilterBlock([]string{"--placeholder", "from flag", "a"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		if got := m.filterModels[0].textinput.Placeholder; got != "from flag" {
			t.Errorf("explicit flag should override env var, got %q", got)
		}
	})
}

// TestPerPanelEnvVarsReflectedInView verifies that select per-panel env vars
// are visible in the rendered output (not just stored in model fields).
func TestPerPanelEnvVarsReflectedInView(t *testing.T) {
	defaultOpts := Options{Height: 10, Border: "single"}

	t.Run("GUM_FILTER_HEADER visible in filter view", func(t *testing.T) {
		t.Setenv("GUM_FILTER_HEADER", "MujFilter")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		m.borderStyle = lipgloss.NewStyle()
		view := m.View()
		if !strings.Contains(view, "MujFilter") {
			t.Errorf("filter header 'MujFilter' not in view:\n%s", view)
		}
	})

	t.Run("GUM_CHOOSE_HEADER visible in choose view", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_HEADER", "MujChoose")
		copts, _ := parseChooseBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelChoose, ChooseOpts: copts}}, defaultOpts)
		m.borderStyle = lipgloss.NewStyle()
		view := m.View()
		if !strings.Contains(view, "MujChoose") {
			t.Errorf("choose header 'MujChoose' not in view:\n%s", view)
		}
	})

	t.Run("GUM_FILTER_PROMPT visible in filter view", func(t *testing.T) {
		t.Setenv("GUM_FILTER_PROMPT", "??> ")
		fopts, _ := parseFilterBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelFilter, FilterOpts: fopts}}, defaultOpts)
		m.borderStyle = lipgloss.NewStyle()
		view := m.View()
		if !strings.Contains(view, "??> ") {
			t.Errorf("filter prompt '??> ' not in view:\n%s", view)
		}
	})

	t.Run("GUM_CHOOSE_CURSOR visible in choose view", func(t *testing.T) {
		t.Setenv("GUM_CHOOSE_CURSOR", "==> ")
		copts, _ := parseChooseBlock([]string{"a", "b"})
		m := makeOrchestrator(t, []Panel{{Type: PanelChoose, ChooseOpts: copts}}, defaultOpts)
		m.borderStyle = lipgloss.NewStyle()
		view := m.View()
		if !strings.Contains(view, "==> ") {
			t.Errorf("choose cursor '==> ' not in view:\n%s", view)
		}
	})
}
