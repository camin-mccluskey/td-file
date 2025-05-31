package tui

import (
	"strings"
	"testing"

	"td-file/parser"
	"td-file/sync"

	tea "github.com/charmbracelet/bubbletea"
)

type dummySync struct {
	Path     string
	ReloadCh chan struct{}
	SaveCh   chan []parser.Todo
}

func TestModel_View_Empty(t *testing.T) {
	m := Model{
		todos:     nil,
		roots:     nil,
		flat:      nil,
		sync:      nil, // not needed for View
		collapsed: make(map[int]bool),
	}
	out := m.View()
	if !strings.Contains(out, "No todos found") {
		t.Errorf("expected 'No todos found' in view, got: %s", out)
	}
}

func TestModel_View_NonEmpty(t *testing.T) {
	todo := parser.Todo{ID: 1, Text: "Test todo", State: parser.Incomplete}
	m := Model{
		todos:     []parser.Todo{todo},
		collapsed: make(map[int]bool),
		sync:      nil, // not needed for View
	}
	m.refreshTree()
	out := m.View()
	if !strings.Contains(out, "Test todo") {
		t.Errorf("expected todo text in view, got: %s", out)
	}
}

func TestModel_Update_Quit(t *testing.T) {
	m := Model{
		collapsed: make(map[int]bool),
		sync:      nil,
	}
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected a quit command, got nil")
	}
	if cmd() != tea.Quit() {
		t.Errorf("expected tea.Quit command, got something else")
	}
}

func TestModel_Update_AddAndEdit(t *testing.T) {
	fs := &sync.FileSynchronizer{
		Path:     "dummy.md",
		ReloadCh: make(chan struct{}, 1),
		SaveCh:   make(chan []parser.Todo, 10),
	}
	m := Model{
		collapsed: make(map[int]bool),
		sync:      fs,
	}
	// Add a todo (simulate 'a' key)
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m2, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model type after add")
	}
	if len(m2.todos) != 1 {
		t.Fatalf("expected 1 todo after add, got %d", len(m2.todos))
	}
	// Enter edit mode (simulate 'e' key)
	model, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m3, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model type after edit")
	}
	if !m3.editing {
		t.Fatal("expected editing mode after 'e' key")
	}
	// Clear the buffer (simulate backspaces)
	for range m3.editBuffer {
		model, _ = m3.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m3, _ = model.(Model)
	}
	// Type 'H'
	model, _ = m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	m4, _ := model.(Model)
	// Type 'i'
	model, _ = m4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m5, _ := model.(Model)
	// Save (simulate enter)
	model, _ = m5.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m6, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model type after enter")
	}
	if m6.todos[0].Text != "Hi" {
		t.Errorf("expected todo text to be updated to 'Hi', got: %q", m6.todos[0].Text)
	}
}

func TestModel_HighlightFeature(t *testing.T) {
	fs := &sync.FileSynchronizer{
		Path:     "dummy.md",
		ReloadCh: make(chan struct{}, 1),
		SaveCh:   make(chan []parser.Todo, 10),
	}
	m := Model{
		collapsed: make(map[int]bool),
		sync:      fs,
	}
	// Add a todo
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m2, _ := model.(Model)
	// Toggle highlight (simulate '*')
	model, _ = m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'*'}})
	m3, _ := model.(Model)
	if !m3.todos[0].Highlighted {
		t.Errorf("expected todo to be highlighted after '*' key")
	}
	// Toggle highlight off
	model, _ = m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'*'}})
	m4, _ := model.(Model)
	if m4.todos[0].Highlighted {
		t.Errorf("expected todo to not be highlighted after second '*' key")
	}
	// Mark as completed, highlight should be removed
	model, _ = m4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m5, _ := model.(Model)
	if m5.todos[0].Highlighted {
		t.Errorf("highlight should be removed when todo is completed")
	}
	// Try to highlight a completed todo
	model, _ = m5.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'*'}})
	m6, _ := model.(Model)
	if m6.todos[0].Highlighted {
		t.Errorf("should not be able to highlight a completed todo")
	}
	// Render highlighted todo
	m2.todos[0].Highlighted = true
	m2.refreshTree()
	out := m2.View()
	if !strings.Contains(out, "*") {
		t.Errorf("expected asterisk in rendered highlighted todo")
	}
	if !strings.Contains(out, "New todo") {
		t.Errorf("expected todo text in view")
	}
	// (Color check is not trivial in test, but we check for asterisk and text)
}

func TestRegression_AddRootTodoPreservesAllTodos(t *testing.T) {
	fs := &sync.FileSynchronizer{
		Path:     "dummy.md",
		ReloadCh: make(chan struct{}, 1),
		SaveCh:   make(chan []parser.Todo, 10),
	}
	// Two root todos, one with a child
	flat := []parser.Todo{
		{ID: 1, Text: "A", IndentLevel: 0},
		{ID: 2, Text: "B", IndentLevel: 2},
		{ID: 3, Text: "C", IndentLevel: 0},
	}
	m := Model{
		todos:     flat,
		collapsed: make(map[int]bool),
		sync:      fs,
		nextID:    4,
	}
	m.refreshTree()
	// Select C, add a new root todo after C
	m.cursor = 2
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m2 := model.(Model)
	m2.refreshTree()
	if len(m2.roots) != 3 {
		t.Fatalf("expected 3 root todos, got %d", len(m2.roots))
	}
	if len(m2.roots[0].Children) != 1 || m2.roots[0].Children[0].Text != "B" {
		t.Fatalf("expected A to have child B, got %+v", m2.roots[0].Children)
	}
}

func TestRegression_AddSiblingDoesNotStealChildren(t *testing.T) {
	fs := &sync.FileSynchronizer{
		Path:     "dummy.md",
		ReloadCh: make(chan struct{}, 1),
		SaveCh:   make(chan []parser.Todo, 10),
	}
	// A with children B, C; D is another root
	flat := []parser.Todo{
		{ID: 1, Text: "A", IndentLevel: 0},
		{ID: 2, Text: "B", IndentLevel: 2},
		{ID: 3, Text: "C", IndentLevel: 2},
		{ID: 4, Text: "D", IndentLevel: 0},
	}
	m := Model{
		todos:     flat,
		collapsed: make(map[int]bool),
		sync:      fs,
		nextID:    5,
	}
	m.refreshTree()
	// Select A, add a sibling after A
	m.cursor = 0
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m2 := model.(Model)
	m2.refreshTree()
	if len(m2.roots) != 3 {
		t.Fatalf("expected 3 root todos, got %d", len(m2.roots))
	}
	if len(m2.roots[0].Children) != 2 {
		t.Fatalf("expected A to have 2 children, got %d", len(m2.roots[0].Children))
	}
	if m2.roots[1].Text != "New todo" {
		t.Fatalf("expected new sibling after A, got %+v", m2.roots[1])
	}
	if len(m2.roots[1].Children) != 0 {
		t.Fatalf("expected new sibling to have no children, got %+v", m2.roots[1].Children)
	}
}

func TestRegression_AddChildOnlyAffectsParent(t *testing.T) {
	fs := &sync.FileSynchronizer{
		Path:     "dummy.md",
		ReloadCh: make(chan struct{}, 1),
		SaveCh:   make(chan []parser.Todo, 10),
	}
	// A and B are roots
	flat := []parser.Todo{
		{ID: 1, Text: "A", IndentLevel: 0},
		{ID: 2, Text: "B", IndentLevel: 0},
	}
	m := Model{
		todos:     flat,
		collapsed: make(map[int]bool),
		sync:      fs,
		nextID:    3,
	}
	m.refreshTree()
	// Select A, add a child
	m.cursor = 0
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	m2 := model.(Model)
	m2.refreshTree()
	if len(m2.roots) != 2 {
		t.Fatalf("expected 2 root todos, got %d", len(m2.roots))
	}
	if len(m2.roots[0].Children) != 1 {
		t.Fatalf("expected A to have 1 child, got %d", len(m2.roots[0].Children))
	}
	if m2.roots[1].Text != "B" {
		t.Fatalf("expected B to remain as root, got %+v", m2.roots[1])
	}
}
