package analyzer

import (
	"testing"
)

func TestParseOutputErrors(t *testing.T) {
	output := `Analyzing flutter_harness...

   error • Undefined name 'foo' • lib/pages/home_page.dart:42:10 • undefined_identifier
   warning • Unused import • lib/widgets/todo_item.dart:15:8 • unused_import
   info • Prefer const constructors • lib/main.dart:5:3 • prefer_const_constructors

3 issues found.`

	result := ParseOutput(output)

	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].File != "lib/pages/home_page.dart" {
		t.Errorf("error file: got %q", result.Errors[0].File)
	}
	if result.Errors[0].Line != 42 {
		t.Errorf("error line: got %d, want 42", result.Errors[0].Line)
	}
	if result.Errors[0].Col != 10 {
		t.Errorf("error col: got %d, want 10", result.Errors[0].Col)
	}
	if result.Errors[0].Code != "undefined_identifier" {
		t.Errorf("error code: got %q", result.Errors[0].Code)
	}
	if result.Errors[0].Message != "Undefined name 'foo'" {
		t.Errorf("error message: got %q", result.Errors[0].Message)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Code != "unused_import" {
		t.Errorf("warning code: got %q", result.Warnings[0].Code)
	}

	if len(result.Infos) != 1 {
		t.Fatalf("expected 1 info, got %d", len(result.Infos))
	}
	if result.Infos[0].Code != "prefer_const_constructors" {
		t.Errorf("info code: got %q", result.Infos[0].Code)
	}
}

func TestParseOutputNoIssues(t *testing.T) {
	output := `Analyzing flutter_harness...
No issues found!`

	result := ParseOutput(output)

	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
	}
	if len(result.Infos) != 0 {
		t.Errorf("expected 0 infos, got %d", len(result.Infos))
	}
}

func TestParseOutputMultipleErrors(t *testing.T) {
	output := `   error • Missing return type • lib/a.dart:1:1 • missing_return
   error • Syntax error • lib/b.dart:10:5 • syntax_error
   error • Type mismatch • lib/c.dart:20:3 • type_mismatch`

	result := ParseOutput(output)

	if len(result.Errors) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(result.Errors))
	}

	files := []string{"lib/a.dart", "lib/b.dart", "lib/c.dart"}
	for i, f := range files {
		if result.Errors[i].File != f {
			t.Errorf("error[%d].file: got %q, want %q", i, result.Errors[i].File, f)
		}
	}
}

func TestParseOutputEmptyLines(t *testing.T) {
	output := `

   info • Unused variable • lib/main.dart:3:7 • unused_local_variable

`

	result := ParseOutput(output)

	if len(result.Infos) != 1 {
		t.Fatalf("expected 1 info, got %d", len(result.Infos))
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"42", 42},
		{"0", 0},
		{"999", 999},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseInt(tt.input)
		if got != tt.want {
			t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
