package main

import "testing"

func TestHasAutoStartArg(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "none", args: []string{"--foo"}, want: false},
		{name: "exact", args: []string{"--autostart"}, want: true},
		{name: "case insensitive", args: []string{"--AUTOSTART"}, want: true},
		{name: "trimmed", args: []string{"  --autostart  "}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAutoStartArg(tt.args); got != tt.want {
				t.Fatalf("hasAutoStartArg(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
