package plan

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Plan is the encapsulation of a cofidectl configuration to be applied to a given cluster or context
type Plan struct {
	// planFile is the location of the filesystem to serialise the contents of this Plan to
	planFile string

	// TrustZone is the TrustZone object defined and orchestrated by this Plan
	TrustZone TrustZone
}

func InitOrLoadPlan() *Plan {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Problem writing planfile: %v", err)
		return nil
	}

	var p Plan
	planFile := cwd + "/cofide.plan"
	if _, err := os.Stat(planFile); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("No Cofide planfile detected. Writing %s", planFile)
		p.planFile = planFile
		p.Write()
		return &p
	}

	data, err := os.ReadFile(planFile)
	if err != nil {
		fmt.Printf("Problem reading existing planfile: %v", err)
	}
	err = yaml.Unmarshal(data, &p)
	if err != nil {
		fmt.Printf("Problem unmarshalling existing planfile: %v", err)
	}
	p.planFile = planFile

	return &p
}

func (p *Plan) Write() {
	data, err := yaml.Marshal(p)
	if err != nil {
		fmt.Printf("Problem marshalling Plan: %v", err)
	}
	os.WriteFile(p.planFile, data, 0644)
}
