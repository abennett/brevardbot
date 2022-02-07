package main

import (
	"testing"
	"time"
)

var nextEmitTests = []struct {
	input    time.Duration
	expected time.Duration
}{
	{
		input:    10 * time.Minute,
		expected: 1 * time.Minute,
	},
	{
		input:    16 * time.Minute,
		expected: 5 * time.Minute,
	},
	{
		input:    12 * time.Minute,
		expected: 2 * time.Minute,
	},
	{
		input:    3 * time.Minute,
		expected: 1 * time.Minute,
	},
}

func TestNextEmit(t *testing.T) {
	for _, et := range nextEmitTests {
		if result := waitFor(et.input); result != et.expected {
			t.Errorf("expected: %s receieved: %s", et.expected, result)
		}
	}
}
