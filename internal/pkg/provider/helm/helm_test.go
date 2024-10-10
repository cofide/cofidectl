package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelmSPIREProvider(t *testing.T) {
	spireValues := map[string]interface{}{}
	spireCRDsValues := map[string]interface{}{}

	p := NewHelmSPIREProvider(spireValues, spireCRDsValues)
	assert.Equal(t, p.SPIREVersion, "0.21.0")
	assert.Equal(t, p.SPIRECRDsVersion, "0.4.0")
	assert.NotNil(t, p.spireClient)
	assert.NotNil(t, p.spireCRDsClient)
}
