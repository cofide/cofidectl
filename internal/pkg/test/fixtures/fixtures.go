// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"fmt"

	ap_binding_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/ap_binding/v1alpha1"
	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/attestation_policy/v1alpha1"
	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/federation/v1alpha1"
	pluginspb "github.com/cofide/cofide-api-sdk/gen/go/proto/plugins/v1alpha1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_provider/v1alpha1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/proto"
	"github.com/cofide/cofidectl/internal/pkg/test/utils"
	spiretypes "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var trustZoneFixtures map[string]*trust_zone_proto.TrustZone = map[string]*trust_zone_proto.TrustZone{
	"tz1": {
		Name:        "tz1",
		TrustDomain: "td1",
		Bundle: &spiretypes.Bundle{
			JwtAuthorities: []*spiretypes.JWTKey{
				{
					PublicKey: utils.Base64Decode("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0mg3S/3z/NlFHhqvd49RibgQpgsWvVBs66pC27AsJIh9UFs5jW17QQJkaBRt/LtA4jhQIQErj3g1ZPyv2JCfLOA+rFHcGFdsnuf8xTgKQfmp4v/xpvUQVmA9rzoFLx5DTDxLe0tU0lgGhJxPJcoSGzAae/Tn/1jenWkIvyPX1W5TMFiIJkpPpqASOUCOnkdwwZ+XeLo+7XWGUAjNtHVsEIOjiIRFkeZCwKSXJvXy9T5OMjCtGsQFaF6+fg5wE0VJBXCDXMr/uPIbVmozGC75opOOPJXcV8daVbEpCKm2BFDcm0MNchNijGGCR0JhYEhb04YSAhN8tmyjxeHHJiblmwIDAQAB"),
					KeyId:     "sHYIGH99d7NhlAVufX9a9e0D9HMPGCQw",
					ExpiresAt: 1738987145,
					Tainted:   false,
				},
			},
			RefreshHint:    2,
			SequenceNumber: 3,
			TrustDomain:    "td1",
			X509Authorities: []*spiretypes.X509Certificate{
				{
					Asn1:    utils.Base64Decode("MIIDrjCCApagAwIBAgIRAL6Ru792Wi5AhHhh387STRIwDQYJKoZIhvcNAQELBQAwZDELMAkGA1UEBhMCVUsxDzANBgNVBAoTBkNvZmlkZTESMBAGA1UEAxMJY29maWRlLmlvMTAwLgYDVQQFEycyNTMzMTAwMTAyMjM0MjQ3NDE4NDYzOTczNzY0MDQzMTM0OTI3NTQwHhcNMjUwMjA3MTU1ODU1WhcNMjUwMjA4MDM1OTA1WjBkMQswCQYDVQQGEwJVSzEPMA0GA1UEChMGQ29maWRlMRIwEAYDVQQDEwljb2ZpZGUuaW8xMDAuBgNVBAUTJzI1MzMxMDAxMDIyMzQyNDc0MTg0NjM5NzM3NjQwNDMxMzQ5Mjc1NDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM0IjG8AFER3+u7njyJqVyHWnGNqEWkOWGXmUmEAx87fpJr4U5X8piXZwPHPVIfcrH1jINpBAOuCBihrAbhwAX0HmtkPt3LFWMUp47zHS7+sSy2TReuEHTLtqxgEG7iwBG2sby0YTotZnb3q1XjnuydOzYBuLXCghNiIkS+NRe2koOv5QeUZJN7IoDuG6bGg6R4CwmHFhLeA2ZMY9QO/X7PhI9PcL6yDurOxgt43qjjGPrkUVVb4v4ju5iz8COaFp1oGchAq+3Tkd0Pl9Vclv8vllDBDMxMjkXjKO1P0ueomldaBJQ5nP/OpmVjhEZ5S9EOKTcfJ7qqS33TAJnBnp00CAwEAAaNbMFkwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGCz3aiUExK4+2cTKGFcJpxBcAexMBcGA1UdEQQQMA6GDHNwaWZmZTovL3RkMjANBgkqhkiG9w0BAQsFAAOCAQEAfhzGZqw3UC+uJGsOLFQ0v7EWS35UB8PvgWABDd+2cRABnSSsNciaszN0Fz9t1qJcP20eldna5b0eZNJLOH89BEqWGTiXD37B3qAqKsT/pAU0eglMtDCNW+KipDpAoo9dFlbF+cSk9dJlH0gNYsMwO1vMFdrRK/4O79sRkxKn2JMf082EXsFpDzPORDsZ1FidOkWT3kTKbH469zFz8a0El7Tq58/2aELkF9qUnP3ZfN6H9CGiES7OV7kNuzuTadVIiFQpeYxd+U/ro6jKeyUdY83FZ6Qfx/bRTRqXStrbutDcdetWWQvRGRCHRoa0uMNmz8fkqLDRkc+emcJGyGSLAQ=="),
					Tainted: true,
				},
			},
		},
		BundleEndpointUrl: StringPtr("127.0.0.1"),
		Federations: []*federation_proto.Federation{
			{
				From: "tz1",
				To:   "tz2",
			},
		},
		AttestationPolicies: []*ap_binding_proto.APBinding{
			{
				TrustZone:     "tz1",
				Policy:        "ap1",
				FederatesWith: []string{"tz2"},
			},
		},
		JwtIssuer:             StringPtr("https://tz1.example.com"),
		BundleEndpointProfile: trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_SPIFFE.Enum(),
	},
	"tz2": {
		Name:              "tz2",
		TrustDomain:       "td2",
		BundleEndpointUrl: StringPtr("127.0.0.2"),
		Federations: []*federation_proto.Federation{
			{
				From: "tz2",
				To:   "tz1",
			},
		},
		AttestationPolicies: []*ap_binding_proto.APBinding{
			{
				TrustZone:     "tz2",
				Policy:        "ap2",
				FederatesWith: []string{"tz1"},
			},
		},
		JwtIssuer:             StringPtr("https://tz2.example.com"),
		BundleEndpointProfile: trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_WEB.Enum(),
	},
	// tz3 has no federations or bound attestation policies.
	"tz3": {
		Name:                  "tz3",
		TrustDomain:           "td3",
		BundleEndpointUrl:     StringPtr("127.0.0.3"),
		Federations:           []*federation_proto.Federation{},
		AttestationPolicies:   []*ap_binding_proto.APBinding{},
		BundleEndpointProfile: trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_SPIFFE.Enum(),
	},
	// tz4 has no federations or bound attestation policies and uses the istio profile.
	"tz4": {
		Name:                "tz4",
		TrustDomain:         "td4",
		BundleEndpointUrl:   StringPtr("127.0.0.4"),
		Federations:         []*federation_proto.Federation{},
		AttestationPolicies: []*ap_binding_proto.APBinding{},
	},
	// tz5 has no federations or bound attestation policies and has an external SPIRE server.
	"tz5": {
		Name:                "tz5",
		TrustDomain:         "td5",
		BundleEndpointUrl:   StringPtr("127.0.0.5"),
		Federations:         []*federation_proto.Federation{},
		AttestationPolicies: []*ap_binding_proto.APBinding{},
	},
	"tz6": {
		Name:              "tz6",
		TrustDomain:       "td6",
		BundleEndpointUrl: StringPtr("127.0.0.5"),
		Federations:       []*federation_proto.Federation{},
		AttestationPolicies: []*ap_binding_proto.APBinding{
			{
				TrustZone:     "tz6",
				Policy:        "ap4",
				FederatesWith: []string{},
			},
		},
		JwtIssuer:             StringPtr("https://tz6.example.com"),
		BundleEndpointProfile: trust_zone_proto.BundleEndpointProfile_BUNDLE_ENDPOINT_PROFILE_HTTPS_WEB.Enum(),
	},
}

var clusterFixtures map[string]*clusterpb.Cluster = map[string]*clusterpb.Cluster{
	"local1": {
		Name:              StringPtr("local1"),
		TrustZone:         StringPtr("tz1"),
		KubernetesContext: StringPtr("kind-local1"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		Profile: StringPtr("kubernetes"),
		ExtraHelmValues: func() *structpb.Struct {
			ev := map[string]any{
				"global": map[string]any{
					"spire": map[string]any{
						// Modify multiple values in the same map.
						"caSubject": map[string]any{
							"organization": "acme-org",
							"commonName":   "cn.example.com",
						},
					},
				},
				"spire-server": map[string]any{
					// Modify an existing value.
					"logLevel": "INFO",
					// Customise a new value.
					"nameOverride": "custom-server-name",
				},
			}
			value, err := structpb.NewStruct(ev)
			if err != nil {
				panic(err)
			}
			return value
		}(),
		ExternalServer: BoolPtr(false),
	},
	"local2": {
		Name:              StringPtr("local2"),
		TrustZone:         StringPtr("tz2"),
		KubernetesContext: StringPtr("kind-local2"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		Profile:        StringPtr("kubernetes"),
		ExternalServer: BoolPtr(false),
	},
	"local3": {
		Name:              StringPtr("local3"),
		TrustZone:         StringPtr("tz3"),
		KubernetesContext: StringPtr("kind-local3"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		Profile: StringPtr("kubernetes"),
	},
	"local4": {
		Name:              StringPtr("local4"),
		TrustZone:         StringPtr("tz4"),
		KubernetesContext: StringPtr("kind-local4"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		Profile: StringPtr("istio"),
	},
	"local5": {
		Name:              StringPtr("local5"),
		TrustZone:         StringPtr("tz5"),
		KubernetesContext: StringPtr("kind-local5"),
		TrustProvider: &trust_provider_proto.TrustProvider{
			Kind: StringPtr("kubernetes"),
		},
		Profile:        StringPtr("kubernetes"),
		ExternalServer: BoolPtr(true),
	},
}

var attestationPolicyFixtures map[string]*attestation_policy_proto.AttestationPolicy = map[string]*attestation_policy_proto.AttestationPolicy{
	"ap1": {
		Name: "ap1",
		Policy: &attestation_policy_proto.AttestationPolicy_Kubernetes{
			Kubernetes: &attestation_policy_proto.APKubernetes{
				NamespaceSelector: &attestation_policy_proto.APLabelSelector{
					MatchLabels: map[string]string{"kubernetes.io/metadata.name": "ns1"},
				},
			},
		},
	},
	"ap2": {
		Name: "ap2",
		Policy: &attestation_policy_proto.AttestationPolicy_Kubernetes{
			Kubernetes: &attestation_policy_proto.APKubernetes{
				PodSelector: &attestation_policy_proto.APLabelSelector{
					MatchExpressions: []*attestation_policy_proto.APMatchExpression{
						{
							Key:      "foo",
							Operator: string(metav1.LabelSelectorOpIn),
							Values:   []string{"bar"},
						},
					},
				},
			},
		},
	},
	"ap3": {
		Name: "ap3",
		Policy: &attestation_policy_proto.AttestationPolicy_Kubernetes{
			Kubernetes: &attestation_policy_proto.APKubernetes{
				NamespaceSelector: &attestation_policy_proto.APLabelSelector{
					MatchLabels: map[string]string{"kubernetes.io/metadata.name": "ns3"},
				},
				PodSelector: &attestation_policy_proto.APLabelSelector{
					MatchLabels: map[string]string{"label1": "value1", "label2": "value2"},
					MatchExpressions: []*attestation_policy_proto.APMatchExpression{
						{
							Key:      "foo",
							Operator: string(metav1.LabelSelectorOpIn),
							Values:   []string{"bar", "baz"},
						},
						{
							Key:      "foo",
							Operator: string(metav1.LabelSelectorOpNotIn),
							Values:   []string{"qux", "quux"},
						},
						{
							Key:      "bar",
							Operator: string(metav1.LabelSelectorOpExists),
						},
						{
							Key:      "baz",
							Operator: string(metav1.LabelSelectorOpDoesNotExist),
						},
					},
				},
			},
		},
	},
	"ap4": {
		Name: "ap4",
		Policy: &attestation_policy_proto.AttestationPolicy_Static{
			Static: &attestation_policy_proto.APStatic{
				SpiffeId: StringPtr("spiffe://example.com/foo"),
				Selectors: []*spiretypes.Selector{
					{
						Type:  "k8s",
						Value: "ns:foo",
					},
				},
			},
		},
	},
}

var pluginConfigFixtures map[string]*structpb.Struct = map[string]*structpb.Struct{
	"plugin1": func() *structpb.Struct {
		s, err := structpb.NewStruct(map[string]any{
			"list-cfg": []any{
				456,
				"another-string",
			},
			"map-cfg": map[string]any{
				"key1": "yet-another",
				"key2": 789,
			},
		})
		if err != nil {
			panic(fmt.Sprintf("failed to create struct: %s", err))
		}
		return s
	}(),
	"plugin2": func() *structpb.Struct {
		s, err := structpb.NewStruct(map[string]any{
			"string-cfg": "fake-string",
			"number-cfg": 123,
		})
		if err != nil {
			panic(fmt.Sprintf("failed to create struct: %s", err))
		}
		return s
	}(),
}

var pluginsFixtures map[string]*pluginspb.Plugins = map[string]*pluginspb.Plugins{
	// Data source and provision use different plugins.
	"plugins1": {
		DataSource: StringPtr("fake-datasource"),
		Provision:  StringPtr("fake-provision"),
	},
	// Data source and provision use the same plugin.
	"plugins2": {
		DataSource: StringPtr("fake-plugin"),
		Provision:  StringPtr("fake-plugin"),
	},
}

func TrustZone(name string) *trust_zone_proto.TrustZone {
	tz, ok := trustZoneFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid trust zone fixture %s", name))
	}
	tz, err := proto.CloneTrustZone(tz)
	if err != nil {
		panic(fmt.Sprintf("failed to clone trust zone: %s", err))
	}
	return tz
}

func Cluster(name string) *clusterpb.Cluster {
	cluster, ok := clusterFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid cluster fixture %s", name))
	}
	cluster, err := proto.CloneCluster(cluster)
	if err != nil {
		panic(fmt.Sprintf("failed to clone cluster: %s", err))
	}
	return cluster
}

func AttestationPolicy(name string) *attestation_policy_proto.AttestationPolicy {
	ap, ok := attestationPolicyFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid attestation policy fixture %s", name))
	}
	ap, err := proto.CloneAttestationPolicy(ap)
	if err != nil {
		panic(fmt.Sprintf("failed to clone attestation policy: %s", err))
	}
	return ap
}

func PluginConfig(name string) *structpb.Struct {
	pc, ok := pluginConfigFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid plugin config fixture %s", name))
	}
	pc, err := proto.CloneStruct(pc)
	if err != nil {
		panic(fmt.Sprintf("failed to clone plugin config: %s", err))
	}
	return pc
}

func Plugins(name string) *pluginspb.Plugins {
	p, ok := pluginsFixtures[name]
	if !ok {
		panic(fmt.Sprintf("invalid plugins fixture %s", name))
	}
	p, err := proto.ClonePlugins(p)
	if err != nil {
		panic(fmt.Sprintf("failed to clone plugins: %s", err))
	}
	return p
}

func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}
