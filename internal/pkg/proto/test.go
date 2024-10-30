package proto

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	attestation_policy_proto "github.com/cofide/cofide-api-sdk/gen/proto/attestation_policy/v1"
	federation_proto "github.com/cofide/cofide-api-sdk/gen/proto/federation/v1"
	trust_provider_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_provider/v1"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/proto/trust_zone/v1"
)

// IgnoreUnexported returns a `cmp.Option` that ignores the unexported fields in protobuf message types.
// This can be used in tests with `cmp.Diff` or `cmp.Equal`.
func IgnoreUnexported() cmp.Option {
	return cmpopts.IgnoreUnexported(
		attestation_policy_proto.AttestationPolicy{},
		federation_proto.Federation{},
		trust_zone_proto.TrustZone{},
		trust_provider_proto.TrustProvider{},
	)
}
