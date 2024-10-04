package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	p := NewHelmSPIREProvider("test-chart", "0.0.0")
	assert.Equal(t, p.chart, "test-chart")
	assert.Equal(t, p.version, "0.0.0")
}
