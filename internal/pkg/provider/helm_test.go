package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	p := NewHelmSPIREProvider()
	assert.Equal(t, p.chart, "spire")
	assert.Equal(t, p.version, "0.21.0")
}
