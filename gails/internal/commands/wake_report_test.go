package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestUnderWake(t *testing.T) {
	cases := []struct {
		name   string
		envVal string
		setEnv bool
		want   bool
	}{
		{name: "unset", setEnv: false, want: false},
		{name: "empty", envVal: "", setEnv: true, want: false},
		{name: "1", envVal: "1", setEnv: true, want: true},
		{name: "true", envVal: "true", setEnv: true, want: false}, // exact "1" only
		{name: "0", envVal: "0", setEnv: true, want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.setEnv {
				t.Setenv("WAKE_REPORT", tc.envVal)
			}
			if got := underWake(); got != tc.want {
				t.Errorf("underWake() = %v, want %v (env=%q set=%v)", got, tc.want, tc.envVal, tc.setEnv)
			}
		})
	}
}

func TestWakeLogger_Formatting(t *testing.T) {
	// We can construct a wakeLogger directly with any io.Writer (same package
	// access), avoiding a refactor of newWakeLogger().
	cases := []struct {
		name string
		emit func(l wakeLogger)
		// The wake report wire format is JSON lines like {"event":"info","msg":"..."}.
		// We assert the *kind* by checking the prefix of the JSON value.
		wantKind string
		wantMsg  string
	}{
		{
			name:     "Errorf",
			emit:     func(l wakeLogger) { l.Errorf("oops: %d", 42) },
			wantKind: `"k":"error"`,
			wantMsg:  "oops: 42",
		},
		{
			name:     "Warningf",
			emit:     func(l wakeLogger) { l.Warningf("careful: %s", "danger") },
			wantKind: `"k":"warn"`,
			wantMsg:  "careful: danger",
		},
		{
			name:     "Infof",
			emit:     func(l wakeLogger) { l.Infof("step %d/%d", 1, 3) },
			wantKind: `"k":"info"`,
			wantMsg:  "step 1/3",
		},
		{
			name:     "Statusf",
			emit:     func(l wakeLogger) { l.Statusf("working: %s", "build") },
			wantKind: `"k":"status"`,
			wantMsg:  "working: build",
		},
		{
			name:     "Debugf is dropped",
			emit:     func(l wakeLogger) { l.Debugf("noise: %d", 99) },
			wantKind: "",
			wantMsg:  "",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := wakeLogger{w: &buf}
			tc.emit(l)
			got := buf.String()
			if tc.wantKind == "" {
				if got != "" {
					t.Errorf("expected no output, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.wantKind) {
				t.Errorf("output missing kind %q\n  got: %s", tc.wantKind, got)
			}
			if !strings.Contains(got, tc.wantMsg) {
				t.Errorf("output missing msg %q\n  got: %s", tc.wantMsg, got)
			}
		})
	}
}
