package inspector

import (
	"encoding/json"
	"testing"
)

func TestNewInspector(t *testing.T) {
	ins := NewInspector("ws://localhost:1234/ws")
	if ins.debugURL != "ws://localhost:1234/ws" {
		t.Errorf("expected debugURL ws://localhost:1234/ws, got %s", ins.debugURL)
	}
}

func TestInspectNoDebugURL(t *testing.T) {
	ins := NewInspector("")
	_, err := ins.Inspect()
	if err == nil {
		t.Error("expected error when debug URL is empty")
	}
}

func TestBuildSummary(t *testing.T) {
	tree := &WidgetNode{
		Widget: "MaterialApp",
		Children: []*WidgetNode{
			{
				Widget: "Scaffold",
				Children: []*WidgetNode{
					{Widget: "AppBar"},
					{
						Widget: "ListView",
						Children: []*WidgetNode{
							{Widget: "ListTile"},
							{Widget: "ListTile"},
							{Widget: "ListTile"},
						},
					},
				},
			},
		},
	}

	summary := buildSummary(tree)

	if summary.TotalWidgets != 7 {
		t.Errorf("expected 7 total widgets, got %d", summary.TotalWidgets)
	}

	if summary.MaxDepth != 3 {
		t.Errorf("expected max depth 3, got %d", summary.MaxDepth)
	}

	if len(summary.TopWidgets) == 0 {
		t.Error("expected top widgets to not be empty")
	}
}

func TestBuildSummaryNil(t *testing.T) {
	summary := buildSummary(nil)
	if summary.TotalWidgets != 0 {
		t.Errorf("expected 0 total widgets for nil tree, got %d", summary.TotalWidgets)
	}
}

func TestPruneTree(t *testing.T) {
	tree := &WidgetNode{
		Widget: "MaterialApp",
		Children: []*WidgetNode{
			{
				Widget: "Scaffold",
				Children: []*WidgetNode{
					{Widget: "AppBar"},
					{
						Widget: "ListView",
						Children: []*WidgetNode{
							{Widget: "ListTile"},
							{Widget: "ListTile"},
						},
					},
				},
			},
		},
	}

	pruned := PruneTree(tree, 2)

	if pruned.Widget != "MaterialApp" {
		t.Errorf("expected root widget MaterialApp, got %s", pruned.Widget)
	}

	// At depth 2, Scaffold's children should be present but ListView's children should be pruned.
	scaffold := pruned.Children[0]
	if scaffold.Widget != "Scaffold" {
		t.Errorf("expected Scaffold, got %s", scaffold.Widget)
	}

	// AppBar at depth 2 should exist but with no children.
	appBar := scaffold.Children[0]
	if appBar.Widget != "AppBar" {
		t.Errorf("expected AppBar, got %s", appBar.Widget)
	}

	// ListView at depth 2 should have description about omitted children.
	listView := scaffold.Children[1]
	if listView.Widget != "ListView" {
		t.Errorf("expected ListView, got %s", listView.Widget)
	}
	if listView.Description == "" {
		t.Error("expected pruned ListView to have description about omitted children")
	}
	if len(listView.Children) != 0 {
		t.Error("expected pruned ListView to have no children")
	}
}

func TestPruneTreeNil(t *testing.T) {
	result := PruneTree(nil, 5)
	if result != nil {
		t.Error("expected nil result for nil tree")
	}
}

func TestCountDescendants(t *testing.T) {
	tree := &WidgetNode{
		Widget: "Root",
		Children: []*WidgetNode{
			{
				Widget: "A",
				Children: []*WidgetNode{
					{Widget: "B"},
					{Widget: "C"},
				},
			},
			{Widget: "D"},
		},
	}

	count := countDescendants(tree)
	// Root has 2 direct children (A, D), A has 2 children (B, C) = 4 total descendants.
	if count != 4 {
		t.Errorf("expected 4 descendants, got %d", count)
	}
}

func TestParseWidgetTree(t *testing.T) {
	ins := NewInspector("")

	raw := json.RawMessage(`{
		"widgetRuntimeType": "MaterialApp",
		"description": "MaterialApp",
		"children": [
			{
				"widgetRuntimeType": "Scaffold",
				"description": "Scaffold",
				"properties": [
					{
						"name": "backgroundColor",
						"description": "Color(0xfffafafa)"
					}
				],
				"children": []
			}
		],
		"properties": []
	}`)

	node, err := ins.parseWidgetTree(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if node.Widget != "MaterialApp" {
		t.Errorf("expected widget MaterialApp, got %s", node.Widget)
	}

	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(node.Children))
	}

	scaffold := node.Children[0]
	if scaffold.Widget != "Scaffold" {
		t.Errorf("expected child widget Scaffold, got %s", scaffold.Widget)
	}

	if scaffold.Properties["backgroundColor"] != "Color(0xfffafafa)" {
		t.Errorf("expected backgroundColor property, got %v", scaffold.Properties)
	}
}

func TestWidgetNodeJSON(t *testing.T) {
	node := &WidgetNode{
		Widget: "Text",
		Properties: map[string]string{
			"data": "Hello World",
		},
	}

	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed WidgetNode
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Widget != "Text" {
		t.Errorf("expected widget Text, got %s", parsed.Widget)
	}

	if parsed.Properties["data"] != "Hello World" {
		t.Errorf("expected property data 'Hello World', got %s", parsed.Properties["data"])
	}
}
