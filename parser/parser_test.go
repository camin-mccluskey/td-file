package parser_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"td-file/parser"
)

func TestBuildTree(t *testing.T) {
	tests := []struct {
		name string
		flat []parser.Todo
		want []string
	}{
		{"flat list", []parser.Todo{{Text: "A", IndentLevel: 0}, {Text: "B", IndentLevel: 0}}, []string{"0:A", "0:B"}},
		{"simple nest", []parser.Todo{{Text: "A", IndentLevel: 0}, {Text: "B", IndentLevel: 2}}, []string{"0:A", "2:B"}},
		{"multi-level nest", []parser.Todo{{Text: "A", IndentLevel: 0}, {Text: "B", IndentLevel: 2}, {Text: "C", IndentLevel: 4}}, []string{"0:A", "2:B", "4:C"}},
		{"siblings at different levels", []parser.Todo{{Text: "A", IndentLevel: 0}, {Text: "B", IndentLevel: 2}, {Text: "C", IndentLevel: 2}, {Text: "D", IndentLevel: 0}}, []string{"0:A", "2:B", "2:C", "0:D"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots := parser.BuildTree(tt.flat)
			var got []string
			var walk func(nodes []*parser.Todo)
			walk = func(nodes []*parser.Todo) {
				for _, n := range nodes {
					got = append(got, formatNode(n))
					if len(n.Children) > 0 {
						walk(n.Children)
					}
				}
			}
			walk(roots)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tree = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlattenTree(t *testing.T) {
	flat := []parser.Todo{
		{Text: "A", IndentLevel: 0},
		{Text: "B", IndentLevel: 2},
		{Text: "C", IndentLevel: 2},
		{Text: "D", IndentLevel: 0},
	}
	roots := parser.BuildTree(flat)
	flatView := flattenTreeTest(roots, 0)
	want := []string{"0:A", "1:B", "1:C", "0:D"}
	var got []string
	for _, n := range flatView {
		got = append(got, formatFlatNode(n))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("flattened = %v, want %v", got, want)
	}
	roots[0].Collapsed = true
	flatView = flattenTreeTest(roots, 0)
	want = []string{"0:A", "0:D"}
	got = nil
	for _, n := range flatView {
		got = append(got, formatFlatNode(n))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collapsed = %v, want %v", got, want)
	}
}

type treeNodeViewTest struct {
	Todo  *parser.Todo
	Depth int
}

func flattenTreeTest(nodes []*parser.Todo, depth int) []treeNodeViewTest {
	var out []treeNodeViewTest
	for i := range nodes {
		n := nodes[i]
		out = append(out, treeNodeViewTest{Todo: n, Depth: depth})
		if !n.Collapsed && len(n.Children) > 0 {
			children := n.Children
			childrenFlat := flattenTreeTest(children, depth+1)
			out = append(out, childrenFlat...)
		}
	}
	return out
}

func formatNode(n *parser.Todo) string {
	return formatIndent(n.IndentLevel) + n.Text
}

func formatFlatNode(n treeNodeViewTest) string {
	return formatIndent(n.Depth) + n.Todo.Text
}

func formatIndent(i int) string {
	return fmt.Sprintf("%d:", i)
}

func TestTreeMutations(t *testing.T) {
	t.Run("add sibling", func(t *testing.T) {
		flat := []parser.Todo{{Text: "A", IndentLevel: 0}}
		roots := parser.BuildTree(flat)
		newTodo := parser.Todo{Text: "B", IndentLevel: 0}
		roots = parser.AddSibling(roots, roots[0], newTodo)
		var got []string
		var walk func(nodes []*parser.Todo)
		walk = func(nodes []*parser.Todo) {
			for _, n := range nodes {
				got = append(got, formatNode(n))
				if len(n.Children) > 0 {
					walk(n.Children)
				}
			}
		}
		walk(roots)
		want := []string{"0:A", "0:B"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("add sibling = %v, want %v", got, want)
		}
	})

	t.Run("add child", func(t *testing.T) {
		flat := []parser.Todo{{Text: "A", IndentLevel: 0}}
		roots := parser.BuildTree(flat)
		newTodo := parser.Todo{Text: "B", IndentLevel: 2}
		parser.AddChild(roots[0], &newTodo)
		var got []string
		var walk func(nodes []*parser.Todo)
		walk = func(nodes []*parser.Todo) {
			for _, n := range nodes {
				got = append(got, formatNode(n))
				if len(n.Children) > 0 {
					walk(n.Children)
				}
			}
		}
		walk(roots)
		want := []string{"0:A", "2:B"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("add child = %v, want %v", got, want)
		}
	})

	t.Run("delete node", func(t *testing.T) {
		flat := []parser.Todo{{Text: "A", IndentLevel: 0}, {Text: "B", IndentLevel: 2}, {Text: "C", IndentLevel: 0}}
		roots := parser.BuildTree(flat)
		parser.DeleteNode(roots[0], 0)
		var got []string
		var walk func(nodes []*parser.Todo)
		walk = func(nodes []*parser.Todo) {
			for _, n := range nodes {
				got = append(got, formatNode(n))
				if len(n.Children) > 0 {
					walk(n.Children)
				}
			}
		}
		walk(roots)
		want := []string{"0:A", "0:C"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("delete node = %v, want %v", got, want)
		}
	})

	t.Run("edit node", func(t *testing.T) {
		flat := []parser.Todo{{Text: "A", IndentLevel: 0}}
		roots := parser.BuildTree(flat)
		roots[0].Text = "Z"
		if roots[0].Text != "Z" {
			t.Errorf("edit node = %v, want Z", roots[0].Text)
		}
	})

	t.Run("state change", func(t *testing.T) {
		flat := []parser.Todo{{Text: "A", IndentLevel: 0, State: parser.Incomplete}}
		roots := parser.BuildTree(flat)
		parser.SetState(roots[0], parser.Completed)
		if roots[0].State != parser.Completed {
			t.Errorf("state change = %v, want %v", roots[0].State, parser.Completed)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("malformed lines are ignored", func(t *testing.T) {
		blocks := [][]string{{
			"- [ ] Good todo",
			"- [x] Also good",
			"not a todo line",
			"- [z] Invalid state",
			"   - [ ] Nested good",
		}}
		todos := parser.ParseTodos(blocks)
		if len(todos) != 3 {
			t.Errorf("expected 3 valid todos, got %d", len(todos))
		}
	})

	t.Run("deeply nested todos", func(t *testing.T) {
		var block []string
		for i := 0; i < 12; i++ {
			block = append(block, strings.Repeat("  ", i)+"- [ ] Level "+fmt.Sprint(i))
		}
		blocks := [][]string{block}
		todos := parser.ParseTodos(blocks)
		roots := parser.BuildTree(todos)
		cur := roots[0]
		for i := 1; i < 12; i++ {
			if len(cur.Children) != 1 {
				t.Fatalf("expected 1 child at level %d", i)
			}
			cur = cur.Children[0]
		}
		if cur.Text != "Level 11" {
			t.Errorf("deepest node text = %q, want 'Level 11'", cur.Text)
		}
	})

	t.Run("rapid add/delete", func(t *testing.T) {
		flat := []parser.Todo{{Text: "A", IndentLevel: 0}}
		roots := parser.BuildTree(flat)
		for i := 0; i < 20; i++ {
			parser.AddChild(roots[0], &parser.Todo{Text: fmt.Sprintf("C%d", i), IndentLevel: 2})
		}
		if len(roots[0].Children) != 20 {
			t.Errorf("expected 20 children, got %d", len(roots[0].Children))
		}
		for i := 19; i >= 0; i-- {
			parser.DeleteNode(roots[0], i)
		}
		if len(roots[0].Children) != 0 {
			t.Errorf("expected 0 children after deletes, got %d", len(roots[0].Children))
		}
	})
}

func TestTodoRegex(t *testing.T) {
	testCases := []string{
		"- [ ] Test 1",
		"- [x] Test 2",
		"- [-] Test 3",
		"- [>] Test 4",
	}
	for _, tc := range testCases {
		if !parser.TodoRe().MatchString(tc) {
			t.Errorf("Regex failed to match: %q", tc)
		}
	}
}

func TestHighlightFeature(t *testing.T) {
	t.Run("toggle highlight on incomplete", func(t *testing.T) {
		todo := parser.Todo{ID: 1, Text: "Test", State: parser.Incomplete}
		parser.SetHighlight(&todo, true)
		if !todo.Highlighted {
			t.Errorf("expected todo to be highlighted")
		}
		parser.SetHighlight(&todo, false)
		if todo.Highlighted {
			t.Errorf("expected todo to not be highlighted")
		}
	})

	t.Run("cannot highlight completed todo", func(t *testing.T) {
		todo := parser.Todo{ID: 2, Text: "Done", State: parser.Completed}
		parser.SetHighlight(&todo, true)
		if todo.Highlighted {
			t.Errorf("should not be able to highlight completed todo")
		}
	})

	t.Run("highlight removed on state change", func(t *testing.T) {
		todo := parser.Todo{ID: 3, Text: "Test", State: parser.Incomplete, Highlighted: true}
		parser.SetState(&todo, parser.Completed)
		if todo.Highlighted {
			t.Errorf("highlight should be removed when state is not incomplete")
		}
	})

	t.Run("parse and write highlight state", func(t *testing.T) {
		blocks := [][]string{{"- [ ] Highlighted todo *", "- [ ] Normal todo"}}
		todos := parser.ParseTodos(blocks)
		if !todos[0].Highlighted {
			t.Errorf("expected first todo to be highlighted")
		}
		if todos[1].Highlighted {
			t.Errorf("expected second todo to not be highlighted")
		}
		// Simulate writing and re-parsing
		// (This will require WriteTodosToFile to persist highlight state)
	})
}
