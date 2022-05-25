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
	{
		input:    21 * time.Minute,
		expected: 10 * time.Minute,
	},
}

func TestNextEmit(t *testing.T) {
	for _, et := range nextEmitTests {
		if result := waitFor(et.input); result != et.expected {
			t.Errorf("From %d\n\texpected: %s\n\treceieved: %s", et.input/time.Minute, et.expected, result)
		}
	}
}
