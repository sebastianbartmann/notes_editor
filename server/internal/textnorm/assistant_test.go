package textnorm

import "testing"

func TestTrimLeadingBlankLines(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "no prefix", in: "Hello", want: "Hello"},
		{name: "single newline", in: "\nHello", want: "Hello"},
		{name: "multiple blank lines", in: "\n \n\t\nHello", want: "Hello"},
		{name: "crlf blank lines", in: " \r\n\r\nHello", want: "Hello"},
		{name: "preserve first-line indentation", in: "  Hello", want: "  Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimLeadingBlankLines(tt.in)
			if got != tt.want {
				t.Fatalf("TrimLeadingBlankLines() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLeadingBlankLineTrimmerPush(t *testing.T) {
	trimmer := LeadingBlankLineTrimmer{}
	deltas := []string{"\n", " \n", "Hello", " world"}
	want := []string{"", "", "Hello", " world"}

	for i := range deltas {
		got := trimmer.Push(deltas[i])
		if got != want[i] {
			t.Fatalf("delta %d => %q, want %q", i, got, want[i])
		}
	}
}

func TestLeadingBlankLineTrimmerPushPreservesIndentedStart(t *testing.T) {
	trimmer := LeadingBlankLineTrimmer{}
	if got := trimmer.Push("  Hello"); got != "  Hello" {
		t.Fatalf("expected indented text, got %q", got)
	}
	if got := trimmer.Push(" world"); got != " world" {
		t.Fatalf("expected subsequent text, got %q", got)
	}
}
