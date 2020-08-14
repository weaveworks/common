package kubectl

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"'foo'", "foo"},
		{"  'foo'  ", "foo"},
	}

	for _, tc := range tests {
		if output := trimOutput(tc.input); output != tc.expected {
			t.Errorf("Expected %s, got: %s", tc.expected, output)
		}
	}
}

func ExampleLocalClient() {
	local := LocalClient{}
	local.Execute("apply", "-f", "service.yaml")
}

func TestOutputMatrix(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "echo stdout; sleep 0.001; echo stderr >&2; sleep 0.001; echo stdout")

	stdout, stderr, err := outputMatrix(cmd)
	assert.Equal(t, "stdout\nstdout\n", stdout)
	assert.Equal(t, "stderr\n", stderr)
	assert.NoError(t, err)
}
