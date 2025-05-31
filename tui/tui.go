package tui

import (
	"fmt"
	"strings"

	"td-file/parser"
	"td-file/sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type reloadMsg struct{}

type TreeNodeView struct {
	Todo  *parser.Todo
	Depth int
}

type Model struct {
	todos      []parser.Todo
	roots      []*parser.Todo
	flat       []TreeNodeView
	cursor     int
	editing    bool
	editBuffer string
	sync       *sync.FileSynchronizer
	warnings   []string
	errMsg     string
	help       bool
	collapsed  map[int]bool
	nextID     int
}

// Modular lipgloss styles for todo states

func (m *Model) refreshTree() {
	m.roots = buildTreeWithCollapse(m.todos, m.collapsed)
	m.flat = flattenTree(m.roots, 0)
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) Init() tea.Cmd {
	mm := m
	mm.refreshTree()
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reloadMsg:
		blocks, warnings, err := parser.ExtractTdBlocksWithWarnings(m.sync.Path)
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		todos, warn2 := parser.ParseTodosWithWarnings(blocks)
		m.todos = todos
		m.warnings = append(warnings, warn2...)
		m.errMsg = ""
		m.refreshTree()
		return m, nil
	case tea.KeyMsg:
		if m.errMsg != "" {
			return m, nil
		}
		if m.help {
			if msg.String() == "?" || msg.String() == "esc" {
				m.help = false
			}
			return m, nil
		}
		if msg.String() == "?" {
			m.help = true
			return m, nil
		}
		if m.editing {
			switch msg.Type {
			case tea.KeyEnter:
				if len(m.flat) > 0 {
					n := m.flat[m.cursor].Todo
					n.Text = m.editBuffer
					for i := range m.todos {
						if m.todos[i].ID == n.ID {
							m.todos[i].Text = m.editBuffer
							break
						}
					}
					m.sync.SaveCh <- m.flattenForSync()
				}
				m.editing = false
				m.editBuffer = ""
				m.refreshTree()
				return m, nil
			case tea.KeyEsc:
				m.editing = false
				m.editBuffer = ""
				return m, nil
			case tea.KeyBackspace, tea.KeyCtrlH:
				if len(m.editBuffer) > 0 {
					m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
				}
				return m, nil
			case tea.KeyRunes:
				m.editBuffer += msg.String()
				return m, nil
			case tea.KeySpace:
				m.editBuffer += " "
				return m, nil
			default:
				return m, nil
			}
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyDown:
			if m.cursor < len(m.flat)-1 {
				m.cursor++
			}
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyEnter:
			if len(m.flat) > 0 {
				n := m.flat[m.cursor].Todo
				if len(n.Children) > 0 {
					m.collapsed[n.ID] = !n.Collapsed
					m.refreshTree()
				}
			}
		case tea.KeyRunes:
			r := msg.Runes[0]
			switch r {
			case 'q':
				return m, tea.Quit
			case 'j':
				if m.cursor < len(m.flat)-1 {
					m.cursor++
				}
			case 'k':
				if m.cursor > 0 {
					m.cursor--
				}
			case 'x':
				if len(m.flat) > 0 {
					n := m.flat[m.cursor].Todo
					if n.State == parser.Completed {
						parser.SetState(n, parser.Incomplete)
					} else {
						parser.SetState(n, parser.Completed)
					}
					m.sync.SaveCh <- m.flattenForSync()
					m.todos = m.flattenForSync()
					m.refreshTree()
				}
			case '-':
				if len(m.flat) > 0 {
					n := m.flat[m.cursor].Todo
					if n.State == parser.Cancelled {
						parser.SetState(n, parser.Incomplete)
					} else {
						parser.SetState(n, parser.Cancelled)
					}
					m.sync.SaveCh <- m.flattenForSync()
					m.todos = m.flattenForSync()
					m.refreshTree()
				}
			case '>':
				if len(m.flat) > 0 {
					n := m.flat[m.cursor].Todo
					if n.State == parser.Pushed {
						parser.SetState(n, parser.Incomplete)
					} else {
						parser.SetState(n, parser.Pushed)
					}
					m.sync.SaveCh <- m.flattenForSync()
					m.todos = m.flattenForSync()
					m.refreshTree()
				}
			case ' ':
				if len(m.flat) > 0 {
					n := m.flat[m.cursor].Todo
					parser.SetState(n, parser.Incomplete)
					m.sync.SaveCh <- m.flattenForSync()
					m.todos = m.flattenForSync()
					m.refreshTree()
				}
			case 'e':
				if len(m.flat) > 0 {
					m.editing = true
					m.editBuffer = m.flat[m.cursor].Todo.Text
				}
			case 'a':
				if len(m.flat) > 0 {
					flat := m.flattenForSync()
					curIdx := m.cursor
					curIndent := flat[curIdx].IndentLevel
					lastDescendantIdx := curIdx
					for i := curIdx + 1; i < len(flat); i++ {
						if flat[i].IndentLevel <= curIndent {
							break
						}
						lastDescendantIdx = i
					}
					newTodo := parser.Todo{
						ID:          m.nextID,
						Text:        "New todo",
						State:       parser.Incomplete,
						IndentLevel: curIndent,
					}
					m.nextID++
					// Insert after last descendant
					flat = append(flat[:lastDescendantIdx+1], append([]parser.Todo{newTodo}, flat[lastDescendantIdx+1:]...)...)
					m.todos = flat
					m.refreshTree()
					m.sync.SaveCh <- m.todos
					m.cursor = lastDescendantIdx + 1
				} else {
					m.todos = []parser.Todo{{ID: m.nextID, Text: "New todo", State: parser.Incomplete}}
					m.nextID++
					m.refreshTree()
					m.sync.SaveCh <- m.todos
					m.cursor = 0
				}
			case 'A':
				if len(m.flat) > 0 {
					cur := m.flat[m.cursor]
					parent := cur.Todo
					newChild := &parser.Todo{
						ID:          m.nextID,
						Text:        "New child todo",
						State:       parser.Incomplete,
						IndentLevel: parent.IndentLevel + 2,
						Parent:      parent,
					}
					m.nextID++
					parser.AddChild(parent, newChild)
					m.todos = m.flattenForSync()
					m.refreshTree()
					for i, node := range m.flat {
						if node.Todo == parent.Children[len(parent.Children)-1] {
							m.cursor = i
							break
						}
					}
					m.sync.SaveCh <- m.todos
				}
			case 'd':
				if len(m.flat) > 0 {
					cur := m.flat[m.cursor]
					delete(m.collapsed, cur.Todo.ID)
					if cur.Depth == 0 {
						idx := m.findRootIdx(cur.Todo)
						if idx >= 0 {
							m.roots = append(m.roots[:idx], m.roots[idx+1:]...)
						}
					} else {
						parent := m.findParent(cur.Todo)
						if parent != nil {
							idx := m.findChildIdx(parent, cur.Todo)
							if idx >= 0 {
								parser.DeleteNode(parent, idx)
							}
						}
					}
					m.todos = m.flattenForSync()
					m.refreshTree()
					m.sync.SaveCh <- m.todos
					if m.cursor >= len(m.flat) && m.cursor > 0 {
						m.cursor--
					}
				}
			case '*':
				if len(m.flat) > 0 {
					n := m.flat[m.cursor].Todo
					if n.State == parser.Incomplete {
						parser.SetHighlight(n, !n.Highlighted)
						m.sync.SaveCh <- m.flattenForSync()
						m.todos = m.flattenForSync()
						m.refreshTree()
					}
				}
			case '?':
				m.help = true
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.help {
		return helpScreen()
	}
	var b strings.Builder
	if m.errMsg != "" {
		fmt.Fprintf(&b, "Error: %s\n\n", m.errMsg)
	}
	if len(m.warnings) > 0 {
		for _, w := range m.warnings {
			fmt.Fprintf(&b, "Warning: %s\n", w)
		}
		b.WriteString("\n")
	}

	// Get terminal width
	width := lipgloss.Width(b.String())
	if width == 0 {
		width = 80 // fallback width
	}

	// Create styles with dynamic width
	incompleteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Width(width)
	completedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Faint(true).Width(width)
	pushedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Faint(true).Width(width)
	cancelledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Strikethrough(true).Width(width)
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")).Width(width)
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true).Width(width)

	border := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("─", 40))
	b.WriteString(border + "\n")
	if len(m.flat) == 0 {
		b.WriteString("No todos found.\n")
	} else {
		for i, node := range m.flat {
			indent := strings.Repeat("  ", node.Depth)
			icon := "  "
			if len(node.Todo.Children) > 0 {
				if node.Todo.Collapsed {
					icon = "▸ "
				} else {
					icon = "▾ "
				}
			}

			var (
				stateIcon string
				style     lipgloss.Style
			)
			switch node.Todo.State {
			case parser.Completed:
				stateIcon = "✔"
				style = completedStyle
			case parser.Pushed:
				stateIcon = "➤"
				style = pushedStyle
			case parser.Cancelled:
				stateIcon = "✗"
				style = cancelledStyle
			default:
				stateIcon = "○"
				if node.Todo.Highlighted {
					style = highlightStyle
				} else {
					style = incompleteStyle
				}
			}

			text := node.Todo.Text
			if node.Todo.Highlighted {
				text = text + " *"
			}
			if m.editing && i == m.cursor {
				text = m.editBuffer + "|"
			}
			line := fmt.Sprintf("%s%s%s %s", indent, icon, stateIcon, text)
			line = style.Render(line)
			if i == m.cursor {
				line = cursorStyle.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}
	if m.editing {
		b.WriteString("\nEditing: type to edit, enter to save, esc to cancel\n")
	} else {
		b.WriteString("\nPress '?' for help\n")
	}
	return b.String()
}

func helpScreen() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")).Render("Todo TUI - Help")
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("─", 40))
	rows := []string{
		"j / k / ↑ / ↓   Move cursor up/down",
		"enter           Collapse/expand tree node",
		"x / - / > / ␣   Complete, cancel, push, uncomplete",
		"*               Toggle highlight (incomplete only)",
		"e               Edit todo text",
		"a               Add sibling todo",
		"A               Add child todo",
		"d               Delete todo",
		"q / ctrl+c      Quit",
		"? / esc         Toggle help screen",
	}
	rowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	var body strings.Builder
	for _, row := range rows {
		body.WriteString(rowStyle.Render(row) + "\n")
	}
	return header + "\n" + sep + "\n" + body.String()
}

// --- Tree and flatten helpers ---
// flattenTree, flattenForSync, findParent, findChildIdx, findRootIdx, buildTreeWithCollapse

func buildTreeWithCollapse(flat []parser.Todo, collapsed map[int]bool) []*parser.Todo {
	treeNodes := make([]*parser.Todo, len(flat))
	for i := range flat {
		treeNodes[i] = &parser.Todo{
			ID:          flat[i].ID,
			Text:        flat[i].Text,
			State:       flat[i].State,
			IndentLevel: flat[i].IndentLevel,
			LineNumber:  flat[i].LineNumber,
			Collapsed:   collapsed[flat[i].ID],
			Highlighted: flat[i].Highlighted,
		}
	}
	var roots []*parser.Todo
	var stack []*parser.Todo
	for _, t := range treeNodes {
		for len(stack) > 0 && t.IndentLevel <= stack[len(stack)-1].IndentLevel {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, t)
		} else {
			parent := stack[len(stack)-1]
			t.Parent = parent
			parent.Children = append(parent.Children, t)
		}
		stack = append(stack, t)
	}
	return roots
}

func flattenTree(nodes []*parser.Todo, depth int) []TreeNodeView {
	var out []TreeNodeView
	for i := range nodes {
		n := nodes[i]
		out = append(out, TreeNodeView{Todo: n, Depth: depth})
		if !n.Collapsed && len(n.Children) > 0 {
			children := n.Children
			childrenFlat := flattenTree(children, depth+1)
			out = append(out, childrenFlat...)
		}
	}
	return out
}

// flattenForSync flattens the tree to a []parser.Todo for file writing
func (m *Model) flattenForSync() []parser.Todo {
	var out []parser.Todo
	var walk func(nodes []*parser.Todo, indent int)
	walk = func(nodes []*parser.Todo, indent int) {
		for _, n := range nodes {
			t := *n
			t.IndentLevel = indent
			t.Children = nil
			out = append(out, t)
			if len(n.Children) > 0 {
				children := n.Children
				walk(children, indent+2)
			}
		}
	}
	walk(m.roots, 0)
	return out
}

func (m *Model) findParent(child *parser.Todo) *parser.Todo {
	var parent *parser.Todo
	var walk func(nodes []*parser.Todo)
	walk = func(nodes []*parser.Todo) {
		for _, n := range nodes {
			children := n.Children
			for _, c := range children {
				if c == child {
					parent = n
					return
				}
			}
			walk(children)
		}
	}
	walk(m.roots)
	return parent
}

func (m *Model) findChildIdx(parent *parser.Todo, child *parser.Todo) int {
	for i := range parent.Children {
		if parent.Children[i] == child {
			return i
		}
	}
	return -1
}

func (m *Model) findRootIdx(node *parser.Todo) int {
	for i := range m.roots {
		if m.roots[i] == node {
			return i
		}
	}
	return -1
}

// StartTUI launches the Bubbletea program with the given model and synchronizer
func StartTUI(todos []parser.Todo, sync *sync.FileSynchronizer, maxID int) error {
	mdl := Model{todos: todos, sync: sync, collapsed: make(map[int]bool), nextID: maxID + 1}
	mdl.refreshTree()
	p := tea.NewProgram(mdl)
	go func() {
		for range sync.ReloadCh {
			p.Send(reloadMsg{})
		}
	}()
	_, err := p.Run()
	return err
}
