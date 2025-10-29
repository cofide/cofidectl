// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	clusterpb "github.com/cofide/cofide-api-sdk/gen/go/proto/cluster/v1alpha1"
	datasourcepb "github.com/cofide/cofide-api-sdk/gen/go/proto/cofidectl/datasource_plugin/v1alpha2"
	trust_zone_proto "github.com/cofide/cofide-api-sdk/gen/go/proto/trust_zone/v1alpha1"
	"github.com/cofide/cofidectl/internal/pkg/config"
	"github.com/cofide/cofidectl/internal/pkg/test/fixtures"
	"github.com/cofide/cofidectl/pkg/plugin/datasource"
	"github.com/cofide/cofidectl/pkg/plugin/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fakeOIDCIssuerURL = "https://some.oidc"

func TestValidateOpts(t *testing.T) {
	// https://github.com/spiffe/spiffe/blob/main/standards/SPIFFE-ID.md#21-trust-domain
	tt := []struct {
		name          string
		oidcIssuerURL string
		errExpected   bool
	}{
		{oidcIssuerURL: "https://valid.oidc", errExpected: false},
		{oidcIssuerURL: "https://validwithport.oidc:644", errExpected: false},
		{oidcIssuerURL: "h://invalid.oidc", errExpected: true},
		{oidcIssuerURL: "http://valid.oidc", errExpected: true},
		{oidcIssuerURL: "https://valid.oidc/", errExpected: false},
	}

	for _, tc := range tt {
		t.Run(tc.oidcIssuerURL, func(t *testing.T) {
			err := validateOpts(addOpts{kubernetesClusterOIDCIssuerURL: tc.oidcIssuerURL})
			assert.Equal(t, tc.errExpected, err != nil)
		})
	}
}

func TestClusterCommand_addCluster(t *testing.T) {
	tests := []struct {
		name                 string
		clusterName          string
		trustZoneName        string
		injectFailure        bool
		withOIDCIssuer       bool
		withKubeCACert       bool
		wantErr              bool
		wantErrMessage       string
		nonExistentTrustZone bool
	}{
		{
			name:          "success",
			clusterName:   "local2",
			trustZoneName: "tz2",
		},
		{
			name:           "success with OIDC issuer",
			clusterName:    "cluster-oidc",
			trustZoneName:  "tz2",
			withOIDCIssuer: true,
		},
		{
			name:           "success with kube CA cert",
			clusterName:    "cluster-ca-cert",
			trustZoneName:  "tz2",
			withKubeCACert: true,
		},
		{
			name:           "already exists",
			clusterName:    "local1",
			trustZoneName:  "tz1",
			wantErr:        true,
			wantErrMessage: "cluster local1 already exists in trust zone tz1-id in local config",
		},
		{
			name:           "datastore failure",
			clusterName:    "local2",
			trustZoneName:  "tz1",
			injectFailure:  true,
			wantErr:        true,
			wantErrMessage: "fake add failure",
		},
		{
			name:                 "non-existent trust zone",
			clusterName:          "local2",
			trustZoneName:        "tz3",
			wantErr:              true,
			wantErrMessage:       "failed to get trust zone tz3: failed to find trust zone tz3 in local config",
			nonExistentTrustZone: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := newFakeDataSource(t, defaultConfig())
			if tt.injectFailure {
				ds = &failingDS{LocalDataSource: ds.(*local.LocalDataSource)}
			}
			opts := addOpts{
				name:      tt.clusterName,
				trustZone: tt.trustZoneName,
				context:   "kind-local1",
				profile:   "kubernetes",
			}

			if tt.withOIDCIssuer {
				opts.kubernetesClusterOIDCIssuerURL = fakeOIDCIssuerURL
			}

			if tt.withKubeCACert {
				caString, err := getFakeKubeCACert()
				require.NoError(t, err)

				tmpFile, err := os.CreateTemp("", "cert-*.pem")
				require.NoError(t, err)
				defer func() {
					err := tmpFile.Close()
					require.NoError(t, err)
					err = os.Remove(tmpFile.Name())
					require.NoError(t, err)
				}()

				_, err = tmpFile.WriteString(caString)
				require.NoError(t, err)

				opts.kubernetesClusterCACert = tmpFile.Name()
			}

			var tz *trust_zone_proto.TrustZone
			if !tt.nonExistentTrustZone {
				var err error
				tz, err = ds.GetTrustZoneByName(tt.trustZoneName)
				require.NoError(t, err)
			}

			c := ClusterCommand{}
			err := c.addCluster(context.Background(), opts, ds)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMessage)

				if !tt.nonExistentTrustZone {
					// Check that cluster was not added.
					clusters, err := ds.ListClusters(&datasourcepb.ListClustersRequest_Filter{TrustZoneId: tz.Id})
					require.NoError(t, err)
					require.Len(t, clusters, 1)
					require.Equal(t, "local1", *clusters[0].Name)
				}
			} else {
				require.NoError(t, err)

				// Check that cluster was added.
				cluster, err := ds.GetClusterByName(tt.clusterName, *tz.Id)
				require.NoError(t, err)
				require.NotNil(t, cluster)

				if tt.withOIDCIssuer {
					assert.Equal(t, fakeOIDCIssuerURL, cluster.GetOidcIssuerUrl())
				}

				if tt.withKubeCACert {
					caBytes, err := os.ReadFile(opts.kubernetesClusterCACert)
					require.NoError(t, err)
					assert.Equal(t, caBytes, cluster.GetOidcIssuerCaCert())
				}
			}
		})
	}
}

func getFakeKubeCACert() (string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("failed to generate private key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Fake Kubernetes CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", fmt.Errorf("failed to create certificate: %w", err)
	}

	var certPEM bytes.Buffer
	if err := pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", fmt.Errorf("failed to encode cert to PEM: %w", err)
	}

	return certPEM.String(), nil
}

type failingDS struct {
	*local.LocalDataSource
}

// AddCluster fails unconditionally
func (f *failingDS) AddCluster(cluster *clusterpb.Cluster) (*clusterpb.Cluster, error) {
	return nil, errors.New("fake add failure")
}

func newFakeDataSource(t *testing.T, cfg *config.Config) datasource.DataSource {
	configLoader, err := config.NewMemoryLoader(cfg)
	require.Nil(t, err)
	lds, err := local.NewLocalDataSource(configLoader)
	require.Nil(t, err)
	return lds
}

func defaultConfig() *config.Config {
	return &config.Config{
		TrustZones: []*trust_zone_proto.TrustZone{
			fixtures.TrustZone("tz1"),
			fixtures.TrustZone("tz2"),
		},
		Clusters: []*clusterpb.Cluster{
			fixtures.Cluster("local1"),
		},
		Plugins: fixtures.Plugins("plugins1"),
	}
}
