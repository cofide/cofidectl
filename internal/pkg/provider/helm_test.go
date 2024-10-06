package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	p := NewHelmSPIREProvider()
	assert.Equal(t, p.SPIREVersion, "0.21.0")
	assert.Equal(t, p.SPIRECRDsVersion, "0.4.0")
}
