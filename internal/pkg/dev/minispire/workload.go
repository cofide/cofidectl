// Copyright 2024 Cofide Limited.
// SPDX-License-Identifier: Apache-2.0

package minispire

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/cofide/cofide-sdk-go/pkg/id"
	"github.com/spiffe/go-spiffe/v2/bundle/jwtbundle"
	pb "github.com/spiffe/go-spiffe/v2/proto/spiffe/workload"
	spiffeid "github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc/peer"
)

type Config struct {
	CA     *InMemoryCA
	Domain string
}

type svidData struct {
	certBytes []byte
	keyBytes  []byte
	expiry    time.Time
}

// WorkloadHandler implements the Workload API interface
type WorkloadHandler struct {
	c Config

	svids map[string]svidData

	pb.UnimplementedSpiffeWorkloadAPIServer
}

func NewWorkloadHandler(c Config) *WorkloadHandler {
	return &WorkloadHandler{
		c:     c,
		svids: make(map[string]svidData),
	}
}

func (w *WorkloadHandler) FetchX509SVID(req *pb.X509SVIDRequest, resp pb.SpiffeWorkloadAPI_FetchX509SVIDServer) error {
	ctx := resp.Context()
	spiffeID, err := w.generateSpiffeID(ctx)
	if err != nil {
		return err
	}

	log.Printf("Issuing SVID for %s", spiffeID.String())

	if data, ok := w.svids[spiffeID.String()]; ok && time.Now().Add(2*time.Minute).Before(data.expiry) {
		err := resp.Send(&pb.X509SVIDResponse{
			Svids: []*pb.X509SVID{
				{
					SpiffeId:    spiffeID.String(),
					X509Svid:    data.certBytes,
					X509SvidKey: data.keyBytes,
				},
			},
			FederatedBundles: map[string][]byte{
				w.c.Domain: w.c.CA.GetCACert(),
			},
		})
		if err != nil {
			return err
		}

		return w.waitForCertUpdateAndSendSVIDs(req, resp, data.expiry)
	}
	var key crypto.Signer

	if w.c.CA.KeyType == KeyTypeRSA {
		var err error
		key, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
	} else if w.c.CA.KeyType == KeyTypeECDSAP256 {
		var err error
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
	}

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}

	certURL, err := url.Parse(spiffeID.String())
	if err != nil {
		return err
	}
	svidCsr := &x509.CertificateRequest{
		URIs: []*url.URL{certURL},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, svidCsr, key)
	if err != nil {
		return err
	}

	svidBytes, notAfter, err := w.c.CA.Sign(csrBytes)
	if err != nil {
		return err
	}

	w.svids[spiffeID.String()] = svidData{
		certBytes: svidBytes,
		keyBytes:  pkcs8Key,
		expiry:    notAfter,
	}

	err = resp.Send(&pb.X509SVIDResponse{
		Svids: []*pb.X509SVID{
			{
				SpiffeId:    spiffeID.String(),
				X509Svid:    svidBytes,
				X509SvidKey: pkcs8Key,
			},
		},
		FederatedBundles: map[string][]byte{
			w.c.Domain: w.c.CA.GetCACert(),
		},
	})
	if err != nil {
		return err
	}

	return w.waitForCertUpdateAndSendSVIDs(req, resp, notAfter)
}

func (w *WorkloadHandler) waitForCertUpdateAndSendSVIDs(req *pb.X509SVIDRequest, resp pb.SpiffeWorkloadAPI_FetchX509SVIDServer, expiry time.Time) error {
	// this is over simplified logic, however ideal for a local development instance
	time.Sleep(time.Until(expiry.Add(-2 * time.Minute)))

	return w.FetchX509SVID(req, resp)
}

func (w *WorkloadHandler) FetchX509Bundles(req *pb.X509BundlesRequest, resp pb.SpiffeWorkloadAPI_FetchX509BundlesServer) error {
	return resp.Send(&pb.X509BundlesResponse{
		Bundles: map[string][]byte{
			w.c.Domain: w.c.CA.GetCACert(),
		},
	})
}

func (w *WorkloadHandler) FetchJWTSVID(ctx context.Context, req *pb.JWTSVIDRequest) (*pb.JWTSVIDResponse, error) {
	resp := new(pb.JWTSVIDResponse)

	sid, err := w.generateSpiffeID(ctx)
	if err != nil {
		return nil, err
	}

	token, err := w.c.CA.SignWorkloadJWTSVID(ctx, WorkloadJWTSVIDParams{
		SPIFFEID: sid.ToSpiffeID(),
		TTL:      time.Minute * 5,
		Audience: req.Audience,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT SVID: %v", err)
	}

	fmt.Printf("JWT SVID issued: %s\n", token)

	resp.Svids = append(resp.Svids, &pb.JWTSVID{
		SpiffeId: sid.String(),
		Svid:     token,
	})

	return resp, nil
}

func (w *WorkloadHandler) FetchJWTBundles(req *pb.JWTBundlesRequest, stream pb.SpiffeWorkloadAPI_FetchJWTBundlesServer) error {
	ca, err := x509.ParseCertificate(w.c.CA.GetCACert())
	if err != nil {
		return err
	}

	bundle := jwtbundle.FromJWTAuthorities(spiffeid.RequireTrustDomainFromString(w.c.Domain), map[string]crypto.PublicKey{"kid": ca.PublicKey})
	bundleBytes, err := bundle.Marshal()
	if err != nil {
		return err
	}

	err = stream.Send(&pb.JWTBundlesResponse{
		Bundles: map[string][]byte{
			w.c.Domain: bundleBytes,
		},
	})
	if err != nil {
		return err
	}
	return w.waitForCertUpdateAndSendJWTBundle(req, stream)
}

func (w *WorkloadHandler) waitForCertUpdateAndSendJWTBundle(req *pb.JWTBundlesRequest, stream pb.SpiffeWorkloadAPI_FetchJWTBundlesServer) error {
	// this is over simplified logic, however ideal for a local development instance
	time.Sleep(time.Minute)

	return w.FetchJWTBundles(req, stream)
}

func (w *WorkloadHandler) ValidateJWTSVID(ctx context.Context, req *pb.ValidateJWTSVIDRequest) (*pb.ValidateJWTSVIDResponse, error) {
	if req.Audience == "" {
		return nil, errors.New("audience must be specified")
	}
	if req.Svid == "" {
		return nil, errors.New("svid must be specified")
	}
	svid, err := spiffeid.FromString(req.Svid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SPIFFE ID: %v", err)
	}
	claims, err := w.c.CA.ValidateWorkloadJWTSVID(req.String(), svid)
	if err != nil {
		return nil, fmt.Errorf("failed to validate JWT SVID: %v", err)
	}

	if !claims.Audience.Contains(req.Audience) {
		return nil, errors.New("audience does not match")
	}

	return &pb.ValidateJWTSVIDResponse{
		SpiffeId: svid.String(),
	}, nil
}

func (w *WorkloadHandler) generateSpiffeID(ctx context.Context) (*id.SPIFFEID, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("unable to get peer info")
	}
	ai, ok := peer.AuthInfo.(AuthInfo)
	if !ok {
		return nil, errors.New("unable to get auth info")
	}
	info := map[string]string{
		"uid": fmt.Sprint(ai.Caller.UID),
		"pid": fmt.Sprint(ai.Caller.PID),
		"gid": fmt.Sprint(ai.Caller.GID),
	}
	if ai.Caller.BinaryName != "" { // can be empty if the the user mini-spire is running as cannot read the ps data
		info["bin"] = ai.Caller.BinaryName
	}

	return id.NewID(w.c.Domain, info)
}
