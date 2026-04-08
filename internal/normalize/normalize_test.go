package normalize_test

import (
	"testing"
	"time"

	"github.com/DavidSerranoG/go-estado-cuenta-mx/internal/normalize"
)

func TestParseMoneyToCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		want      int64
		wantError bool
	}{
		{name: "whole and fraction", value: "1,234.56", want: 123456},
		{name: "negative amount", value: "-25.00", want: -2500},
		{name: "invalid OCR separators", value: "1.234.56", wantError: true},
		{name: "invalid fraction length", value: "20.5", wantError: true},
		{name: "empty", value: " ", wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalize.ParseMoneyToCents(tt.value)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error for %q", tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("unexpected cents for %q: got %d want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseDateDDMMYYYY(t *testing.T) {
	t.Parallel()

	got, err := normalize.ParseDateDDMMYYYY("05/04/2026")
	if err != nil {
		t.Fatalf("parse dd/mm/yyyy: %v", err)
	}

	want := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("unexpected date %v want %v", got, want)
	}
}

func TestParseDateDDMonYYYYSpanish(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		wantMonth time.Month
		wantError bool
	}{
		{name: "short month", value: "15-Sep-2025", wantMonth: time.September},
		{name: "month with dot", value: "15-Sept.-2025", wantMonth: time.September},
		{name: "lowercase", value: "16-dic-2025", wantMonth: time.December},
		{name: "english january", value: "05-Jan-2026", wantMonth: time.January},
		{name: "english april", value: "05-Apr-2026", wantMonth: time.April},
		{name: "english august", value: "05-Aug-2026", wantMonth: time.August},
		{name: "english december", value: "05-Dec-2026", wantMonth: time.December},
		{name: "invalid month", value: "16-foo-2025", wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalize.ParseDateDDMonYYYYSpanish(tt.value)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error for %q", tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.value, err)
			}
			if got.Month() != tt.wantMonth {
				t.Fatalf("unexpected month for %q: got %v want %v", tt.value, got.Month(), tt.wantMonth)
			}
		})
	}
}

func TestHelpersNormalizeTextAndDigits(t *testing.T) {
	t.Parallel()

	if got := normalize.DigitsOnly("Cuenta 12-34 56"); got != "123456" {
		t.Fatalf("unexpected digits %q", got)
	}
	if got := normalize.CollapseWhitespace("  hola\t mundo \n banco  "); got != "hola mundo banco" {
		t.Fatalf("unexpected collapsed text %q", got)
	}
}

func TestParseOCRMoneyToCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{name: "comma as decimal", value: "822,45", want: 82245},
		{name: "spaces in amount", value: "1 221 00", want: 122100},
		{name: "dots as thousand separators", value: "1.234.56", want: 123456},
		{name: "leading noise", value: "|)$214.00", want: 21400},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalize.ParseOCRMoneyToCents(tt.value)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("unexpected cents for %q: got %d want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeOCRAmountLine(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"+1$453.85": "+$453.85",
		"+|$822.45": "+$822.45",
		"-]$214.00": "-$214.00",
	}

	for input, want := range tests {
		input := input
		want := want

		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if got := normalize.NormalizeOCRAmountLine(input); got != want {
				t.Fatalf("unexpected normalized line: got %q want %q", got, want)
			}
		})
	}
}

func TestNormalizeExtractedText(t *testing.T) {
	t.Parallel()

	raw := "Pagina 1de6\nHSBC\x00 MEXICO\n+\n1 221 00\nOPENAI  \u00a0 CHATGPT\n"
	got := normalize.NormalizeExtractedText(raw)

	if got != "HSBC MEXICO\n+$1221.00\nOPENAI CHATGPT" {
		t.Fatalf("unexpected normalized text %q", got)
	}
}
