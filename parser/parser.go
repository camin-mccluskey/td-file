package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type TodoState int

const (
	Incomplete TodoState = iota
	Completed
	Cancelled
	Pushed
)

type Todo struct {
	ID          int
	Text        string
	State       TodoState
	IndentLevel int
	LineNumber  int
	Children    []*Todo
	Parent      *Todo
	Collapsed   bool
	Highlighted bool
}

var todoRe = regexp.MustCompile(`^(\s*)- \[( |x|\-|>)\] (.*)$`)

// Extracts all complete :td blocks, tolerating odd numbers (ignores unmatched)
func ExtractTdBlocks(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var blocks [][]string
	scanner := bufio.NewScanner(f)
	var inBlock bool
	var block []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == ":td" {
			if inBlock {
				blocks = append(blocks, block)
				block = nil
				inBlock = false
			} else {
				inBlock = true
				block = nil
			}
		} else if inBlock {
			block = append(block, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return blocks, nil
}

// Defensive extractTdBlocks returns blocks and warnings
func ExtractTdBlocksWithWarnings(path string) ([][]string, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var blocks [][]string
	scanner := bufio.NewScanner(f)
	var inBlock bool
	var block []string
	var warnings []string
	blockCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == ":td" {
			if inBlock {
				blocks = append(blocks, block)
				block = nil
				inBlock = false
				blockCount++
			} else {
				inBlock = true
				block = nil
			}
		} else if inBlock {
			block = append(block, line)
		}
	}
	if inBlock {
		warnings = append(warnings, "Unmatched :td block at end of file ignored")
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return blocks, warnings, nil
}

func ParseTodos(blocks [][]string) []Todo {
	var todos []Todo
	lineNum := 0
	for _, block := range blocks {
		for _, line := range block {
			lineNum++
			m := todoRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			indent := len(m[1])
			var state TodoState
			switch m[2] {
			case " ":
				state = Incomplete
			case "x":
				state = Completed
			case "-":
				state = Cancelled
			case ">":
				state = Pushed
			}
			text := m[3]
			highlighted := false
			if strings.HasSuffix(strings.TrimSpace(text), "*") {
				highlighted = true
				text = strings.TrimSpace(text)
				text = strings.TrimSuffix(text, "*")
				text = strings.TrimSpace(text)
			}
			todos = append(todos, Todo{
				ID:          lineNum,
				Text:        text,
				State:       state,
				IndentLevel: indent,
				LineNumber:  lineNum,
				Highlighted: highlighted,
			})
		}
	}
	return todos
}

// Defensive parseTodos returns todos and warnings
func ParseTodosWithWarnings(blocks [][]string) ([]Todo, []string) {
	var todos []Todo
	var warnings []string
	lineNum := 0
	for blockIdx, block := range blocks {
		for _, line := range block {
			lineNum++
			m := todoRe.FindStringSubmatch(line)
			if m == nil {
				if strings.TrimSpace(line) != "" {
					warnings = append(warnings, fmt.Sprintf("Malformed todo in block %d, line %d: '%s'", blockIdx+1, lineNum, line))
				}
				continue
			}
			indent := len(m[1])
			var state TodoState
			switch m[2] {
			case " ":
				state = Incomplete
			case "x":
				state = Completed
			case "-":
				state = Cancelled
			case ">":
				state = Pushed
			}
			text := m[3]
			highlighted := false
			if strings.HasSuffix(strings.TrimSpace(text), "*") {
				highlighted = true
				text = strings.TrimSpace(text)
				text = strings.TrimSuffix(text, "*")
				text = strings.TrimSpace(text)
			}
			todos = append(todos, Todo{
				ID:          lineNum,
				Text:        text,
				State:       state,
				IndentLevel: indent,
				LineNumber:  lineNum,
				Highlighted: highlighted,
			})
		}
	}
	return todos, warnings
}

func WriteTodosToFile(path string, todos []Todo) {
	input, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(input), "\n")
	var out []string
	inBlock := false
	blockIdx := 0
	curTodo := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == ":td" {
			if !inBlock {
				inBlock = true
				out = append(out, line)
				for curTodo < len(todos) && (blockIdx == 0 || todos[curTodo].LineNumber > 0) {
					indent := strings.Repeat(" ", todos[curTodo].IndentLevel)
					state := " "
					switch todos[curTodo].State {
					case Completed:
						state = "x"
					case Cancelled:
						state = "-"
					case Pushed:
						state = ">"
					}
					text := todos[curTodo].Text
					if todos[curTodo].Highlighted {
						text = strings.TrimSpace(text) + " *"
					}
					out = append(out, fmt.Sprintf("%s- [%s] %s", indent, state, text))
					curTodo++
				}
				blockIdx++
				for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != ":td" {
					i++
				}
			} else {
				inBlock = false
				out = append(out, line)
			}
		} else if !inBlock {
			out = append(out, line)
		}
	}
	os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

// Add mutation helpers for todos
func SetState(todo *Todo, state TodoState) {
	todo.State = state
	if state != Incomplete {
		todo.Highlighted = false
	}
}

func AddSibling(roots []*Todo, node *Todo, newTodo Todo) []*Todo {
	for i, n := range roots {
		if n == node {
			out := append([]*Todo{}, roots[:i+1]...)
			out = append(out, &newTodo)
			out = append(out, roots[i+1:]...)
			var flat []Todo
			for _, t := range out {
				flat = append(flat, *t)
			}
			return BuildTree(flat)
		}
	}
	return roots
}

func AddChild(parent *Todo, newTodo *Todo) {
	parent.Children = append(parent.Children, newTodo)
}

func DeleteNode(parent *Todo, childIdx int) {
	if childIdx < 0 || childIdx >= len(parent.Children) {
		return
	}
	parent.Children = append(parent.Children[:childIdx], parent.Children[childIdx+1:]...)
}

// BuildTree converts a flat slice of todos (with IndentLevel) into a tree of todos.
func BuildTree(flat []Todo) []*Todo {
	treeNodes := make([]*Todo, len(flat))
	for i := range flat {
		treeNodes[i] = &Todo{
			ID:          flat[i].ID,
			Text:        flat[i].Text,
			State:       flat[i].State,
			IndentLevel: flat[i].IndentLevel,
			LineNumber:  flat[i].LineNumber,
			Highlighted: flat[i].Highlighted,
		}
	}
	var roots []*Todo
	var stack []*Todo
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

// Exported accessor for todoRe
func TodoRe() *regexp.Regexp {
	return todoRe
}

// SetHighlight sets the highlight state, only if todo is incomplete
func SetHighlight(todo *Todo, highlight bool) {
	if todo.State == Incomplete {
		todo.Highlighted = highlight
	}
}
