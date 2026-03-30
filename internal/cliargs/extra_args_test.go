package cliargs

import (
	"reflect"
	"testing"
)

func TestNormalizeExtraArgs(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		want   []string
		hasErr bool
	}{
		{
			name:  "plain repeated args",
			input: []string{"--dart-define=A=1", "--flavor", "dev"},
			want:  []string{"--dart-define=A=1", "--flavor", "dev"},
		},
		{
			name:  "single json array string",
			input: []string{`["--dart-define=ENABLE_ADMIN_MANAGEMENT=true"]`},
			want:  []string{"--dart-define=ENABLE_ADMIN_MANAGEMENT=true"},
		},
		{
			name:  "mixed plain and json array",
			input: []string{`["--dart-define=A=1","--dart-define=B=2"]`, "--flavor=dev"},
			want:  []string{"--dart-define=A=1", "--dart-define=B=2", "--flavor=dev"},
		},
		{
			name:   "invalid json array",
			input:  []string{`["--dart-define=A=1"`},
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeExtraArgs(tt.input)
			if tt.hasErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("NormalizeExtraArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
