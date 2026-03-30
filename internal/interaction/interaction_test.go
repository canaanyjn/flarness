package interaction

import (
	"strings"
	"testing"
)

func TestRectCenter(t *testing.T) {
	r := Rect{Left: 100, Top: 200, Width: 50, Height: 60}
	cx, cy := r.Center()
	if cx != 125 {
		t.Errorf("expected cx=125, got %f", cx)
	}
	if cy != 230 {
		t.Errorf("expected cy=230, got %f", cy)
	}
}

func TestMatchesFinder(t *testing.T) {
	node := &SemanticsNode{
		ID:    1,
		Label: "Add Todo",
		Value: "some value",
		Hint:  "Double tap to add",
		Flags: []string{"isButton", "hasEnabledState"},
	}

	tests := []struct {
		name   string
		finder Finder
		want   bool
	}{
		{"text match label", Finder{By: FindByText, Value: "Add Todo"}, true},
		{"text match case insensitive", Finder{By: FindByText, Value: "add todo"}, true},
		{"text match partial", Finder{By: FindByText, Value: "Add"}, true},
		{"text no match", Finder{By: FindByText, Value: "Delete"}, false},
		{"text match value", Finder{By: FindByText, Value: "some value"}, true},
		{"tooltip match", Finder{By: FindByTooltip, Value: "Double tap"}, true},
		{"tooltip no match", Finder{By: FindByTooltip, Value: "Long press"}, false},
		{"type match flag", Finder{By: FindByType, Value: "isButton"}, true},
		{"type no match", Finder{By: FindByType, Value: "isTextField"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := finderScore(node, tt.finder) > 0
			if got != tt.want {
				t.Errorf("finderScore()>0 = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseQuotedValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"Hello World"`, "Hello World"},
		{` "Hello" `, "Hello"},
		{"no quotes", "no quotes"},
		{`""`, ""},
	}
	for _, tt := range tests {
		got := parseQuotedValue(tt.input)
		if got != tt.want {
			t.Errorf("parseQuotedValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseRect(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Rect
	}{
		{"LTWH format", "Rect.fromLTWH(10.0, 20.0, 100.0, 50.0)", Rect{Left: 10, Top: 20, Width: 100, Height: 50}},
		{"LTRB format", "Rect.fromLTRB(10.0, 20.0, 110.0, 70.0)", Rect{Left: 10, Top: 20, Width: 100, Height: 50}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &SemanticsNode{}
			parseRect(tt.input, node)
			if node.Rect != tt.want {
				t.Errorf("parseRect() got %+v, want %+v", node.Rect, tt.want)
			}
		})
	}
}

func TestNodeLabel(t *testing.T) {
	tests := []struct {
		name string
		node *SemanticsNode
		want string
	}{
		{"with label", &SemanticsNode{ID: 1, Label: "Submit"}, "Submit"},
		{"with value", &SemanticsNode{ID: 2, Value: "Hello"}, "Hello"},
		{"with hint", &SemanticsNode{ID: 3, Hint: "Enter name"}, "Enter name"},
		{"no label", &SemanticsNode{ID: 4}, "node#4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nodeLabel(tt.node)
			if got != tt.want {
				t.Errorf("nodeLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasAction(t *testing.T) {
	node := &SemanticsNode{
		Actions: []string{"tap", "longPress", "scrollUp"},
	}

	if !hasAction(node, "tap") {
		t.Error("expected hasAction(tap)=true")
	}
	if !hasAction(node, "Tap") {
		t.Error("expected hasAction(Tap)=true")
	}
	if hasAction(node, "scrollDown") {
		t.Error("expected hasAction(scrollDown)=false")
	}
}

func TestSemanticsActionIndex(t *testing.T) {
	if idx := semanticsActionIndex("tap"); idx != 0 {
		t.Errorf("tap index = %d, want 0", idx)
	}
	if idx := semanticsActionIndex("longPress"); idx != 1 {
		t.Errorf("longPress index = %d, want 1", idx)
	}
	if idx := semanticsActionIndex("nonexistent"); idx != -1 {
		t.Errorf("nonexistent index = %d, want -1", idx)
	}
}

func TestIsTransientInteractionError(t *testing.T) {
	if !isTransientInteractionError(errString("read failed: i/o timeout")) {
		t.Fatal("expected timeout error to be transient")
	}
	if isTransientInteractionError(errString("service extension returned error")) {
		t.Fatal("expected regular extension error to be non-transient")
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"it's", "it\\'s"},
		{"line\nbreak", "line\\nbreak"},
		{"back\\slash", "back\\\\slash"},
	}

	for _, tt := range tests {
		got := escapeString(tt.input)
		if got != tt.want {
			t.Errorf("escapeString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSemanticsUnavailableReason(t *testing.T) {
	msg := `Semantics not generated for _ReusableRenderView#ee571.
For performance reasons, the framework only generates semantics when asked to do so by the platform.
To generate semantics, try turning on an assistive technology (like VoiceOver or TalkBack) on your device.`

	reason := semanticsUnavailableReason(msg)
	if reason == "" {
		t.Fatalf("expected non-empty reason for unavailable semantics message")
	}
}

func TestNormalizeExtensionError(t *testing.T) {
	err := normalizeExtensionError(errString(`RPC error: Unknown method "ext.flarness.tapAt"`), "ext.flarness.tapAt")
	if err == nil || !strings.Contains(err.Error(), "flarness_plugin") {
		t.Fatalf("expected plugin guidance error, got %v", err)
	}
}

func TestDecodeSemanticsDumpPayload(t *testing.T) {
	raw := []byte(`{"result":"{\"status\":\"ok\",\"nodes\":[{\"id\":1,\"label\":\"Login\",\"rect\":{\"left\":0,\"top\":0,\"width\":10,\"height\":10}}]}"}`)
	payload, err := decodeSemanticsDumpPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payload.Nodes) != 1 || payload.Nodes[0].Label != "Login" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestParseSemanticsTreeDump(t *testing.T) {
	dump := `SemanticsNode#0
 ├SemanticsNode#1
 │ Rect.fromLTWH(0.0, 0.0, 393.0, 852.0)
 │ label: "My App"
 │ actions: tap
 │ flags: isButton, hasEnabledState
 ├SemanticsNode#2
 │ Rect.fromLTWH(50.0, 100.0, 200.0, 40.0)
 │ label: "Add Todo"
 │ actions: tap, longPress
 │ flags: isButton`

	nodes := parseSemanticsTreeDump(dump)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(nodes))
	}

	root := nodes[0]
	if root.ID != 0 {
		t.Errorf("root ID = %d, want 0", root.ID)
	}
	if len(root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(root.Children))
	}

	child1 := root.Children[0]
	if child1.ID != 1 {
		t.Errorf("child1 ID = %d, want 1", child1.ID)
	}
	if child1.Label != "My App" {
		t.Errorf("child1 label = %q, want %q", child1.Label, "My App")
	}
	if len(child1.Actions) != 1 || child1.Actions[0] != "tap" {
		t.Errorf("child1 actions = %v, want [tap]", child1.Actions)
	}
	if len(child1.Flags) != 2 {
		t.Errorf("child1 flags = %v, want [isButton, hasEnabledState]", child1.Flags)
	}
	if child1.Rect.Width != 393 {
		t.Errorf("child1 rect width = %f, want 393", child1.Rect.Width)
	}

	child2 := root.Children[1]
	if child2.ID != 2 {
		t.Errorf("child2 ID = %d, want 2", child2.ID)
	}
	if child2.Label != "Add Todo" {
		t.Errorf("child2 label = %q, want %q", child2.Label, "Add Todo")
	}
	if len(child2.Actions) != 2 {
		t.Errorf("child2 actions = %v, want [tap, longPress]", child2.Actions)
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
