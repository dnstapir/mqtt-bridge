package logging

import (
	"testing"
)

func TestLoggingDebugNoPanic(t *testing.T) {
	var tests = []struct {
		name    string
		indata1 bool
		indata2 bool
	}{
		{"true-true", true, true},
		{"true-false", true, false},
		{"false-true", false, true},
		{"true-true", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Create(tt.indata1, tt.indata2)
			l.Debug("nothing")
		})
	}
}

func TestLoggingInfoNoPanic(t *testing.T) {
	var tests = []struct {
		name    string
		indata1 bool
		indata2 bool
	}{
		{"true-true", true, true},
		{"true-false", true, false},
		{"false-true", false, true},
		{"true-true", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Create(tt.indata1, tt.indata2)
			l.Info("nothing")
		})
	}
}

func TestLoggingWarningNoPanic(t *testing.T) {
	var tests = []struct {
		name    string
		indata1 bool
		indata2 bool
	}{
		{"true-true", true, true},
		{"true-false", true, false},
		{"false-true", false, true},
		{"true-true", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Create(tt.indata1, tt.indata2)
			l.Warning("nothing")
		})
	}
}

func TestLoggingErrorNoPanic(t *testing.T) {
	var tests = []struct {
		name    string
		indata1 bool
		indata2 bool
	}{
		{"true-true", true, true},
		{"true-false", true, false},
		{"false-true", false, true},
		{"true-true", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Create(tt.indata1, tt.indata2)
			l.Error("nothing")
		})
	}
}

func TestLoggingFormat(t *testing.T) {
	var tests = []struct {
		name           string
		indata_fmtstr  string
		indata_varargs []any
		expected       string
	}{
		{"NO_VARARGS", "hello", []any{}, "hello"},
		{"NIL_VARARGS", "hello", nil, "hello"},
		{"ONE_VARARGS", "hello %s", []any{"world"}, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := format(tt.indata_fmtstr, tt.indata_varargs)
			if got != tt.expected {
				t.Fatalf("got %s, expected %s", got, tt.expected)
			}
		})
	}
}
