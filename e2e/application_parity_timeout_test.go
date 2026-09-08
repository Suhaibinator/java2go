package e2e

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestApplicationParityTimeout(t *testing.T) {
	for _, test := range []struct {
		name, value string
		want        time.Duration
	}{
		{"default", "", 60 * time.Second},
		{"seconds", "180s", 180 * time.Second},
		{"minutes", "3m", 3 * time.Minute},
		{"fractional", "1.5s", 1500 * time.Millisecond},
		{"milliseconds", "250ms", 250 * time.Millisecond},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseApplicationParityTimeout(test.value)
			if err != nil || got != test.want {
				t.Fatalf("parseApplicationParityTimeout(%q) = %v, %v; want %v, nil", test.value, got, err, test.want)
			}
		})
	}
}

func TestApplicationParityTimeoutRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "0s", "-1s", "180", "invalid", "1d", " 180s ", "999999999999999999999h"} {
		t.Run(value, func(t *testing.T) {
			_, err := parseApplicationParityTimeout(value)
			if err == nil {
				t.Fatalf("parseApplicationParityTimeout(%q) accepted an invalid timeout", value)
			}
			for _, detail := range []string{"JAVA2GO_PARITY_TIMEOUT", strconv.Quote(value), "positive Go duration"} {
				if !strings.Contains(err.Error(), detail) {
					t.Errorf("timeout error %q must include %q", err, detail)
				}
			}
		})
	}
}
