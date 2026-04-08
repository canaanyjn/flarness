package process

import (
	"testing"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateIdle, "idle"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateReloading, "reloading"},
		{StateStopped, "stopped"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNewManager(t *testing.T) {
	var events []string
	cb := func(source, line string) {
		events = append(events, source+":"+line)
	}

	m := New("/test/project", "chrome", nil, cb)

	if m.project != "/test/project" {
		t.Errorf("project: got %q, want /test/project", m.project)
	}
	if m.device != "chrome" {
		t.Errorf("device: got %q, want chrome", m.device)
	}
	if m.GetState() != StateIdle {
		t.Errorf("initial state: got %v, want idle", m.GetState())
	}
}

func TestStartConfiguresNewProcessGroup(t *testing.T) {
	m := New("/test/project", "chrome", nil, nil)
	args := []string{"run", "--machine", "-d", "chrome"}
	m.cmd = nil

	cmd := buildFlutterCommand(m.project, m.device, nil, nil)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Fatal("expected flutter command to start in its own process group")
	}
	if len(cmd.Args) != len(args)+1 {
		t.Fatalf("unexpected arg count: got %d", len(cmd.Args))
	}
}

func TestBuildFlutterCommandUsesConfiguredWrapper(t *testing.T) {
	cmd := buildFlutterCommand("/test/project", "macos", []string{"./scripts/dev.sh", "--api-base-url", "http://localhost:3000"}, []string{"--flavor", "dev"})

	want := []string{
		"./scripts/dev.sh",
		"--api-base-url", "http://localhost:3000",
		"run", "--machine", "-d", "macos",
		"--flavor", "dev",
	}
	if len(cmd.Args) != len(want) {
		t.Fatalf("arg count: got %d, want %d (%#v)", len(cmd.Args), len(want), cmd.Args)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Fatalf("arg[%d]: got %q, want %q (%#v)", i, cmd.Args[i], want[i], cmd.Args)
		}
	}
}

func TestDescendantPIDsParsesProcessTree(t *testing.T) {
	tree := map[int][]int{
		10: {11, 12},
		11: {13},
		12: {14, 15},
	}
	got := collectDescendantPIDs(tree, 10)
	want := []int{11, 13, 12, 14, 15}
	if len(got) != len(want) {
		t.Fatalf("descendant count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("descendants[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestManagerStateTransitions(t *testing.T) {
	m := New("/test", "chrome", nil, nil)

	if m.GetState() != StateIdle {
		t.Fatalf("expected idle")
	}

	m.SetState(StateStarting)
	if m.GetState() != StateStarting {
		t.Fatalf("expected starting")
	}

	m.SetState(StateRunning)
	if m.GetState() != StateRunning {
		t.Fatalf("expected running")
	}

	m.SetState(StateStopped)
	if m.GetState() != StateStopped {
		t.Fatalf("expected stopped")
	}
}

func TestReloadResultNotify(t *testing.T) {
	m := New("/test", "chrome", nil, nil)
	m.SetState(StateRunning)

	// Simulate a reload: create the result channel.
	m.reloadResult = make(chan ReloadResult, 1)

	expected := ReloadResult{
		Success:    true,
		DurationMs: 320,
	}

	// Notify the result.
	m.NotifyReloadResult(expected)

	// Read the result.
	result, err := m.WaitReloadResult(1e9) // 1 second
	if err != nil {
		t.Fatalf("WaitReloadResult error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.DurationMs != 320 {
		t.Errorf("duration: got %d, want 320", result.DurationMs)
	}
}

func TestReloadResultTimeout(t *testing.T) {
	m := New("/test", "chrome", nil, nil)
	m.SetState(StateRunning)
	m.reloadResult = make(chan ReloadResult, 1)

	// Don't notify — should timeout.
	_, err := m.WaitReloadResult(10e6) // 10ms
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestSendReloadNotRunning(t *testing.T) {
	m := New("/test", "chrome", nil, nil)
	// State is idle, reload should fail.
	err := m.SendReload()
	if err == nil {
		t.Error("expected error when reloading idle process")
	}
}

func TestSendRestartNotRunning(t *testing.T) {
	m := New("/test", "chrome", nil, nil)
	err := m.SendRestart()
	if err == nil {
		t.Error("expected error when restarting idle process")
	}
}
