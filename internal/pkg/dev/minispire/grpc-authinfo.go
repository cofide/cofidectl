package minispire

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"google.golang.org/grpc/credentials"
)

var (
	ErrInvalidConnection = errors.New("invalid connection")
)

type Conn struct {
	net.Conn
	Info AuthInfo
}

func (c *Conn) Close() error {
	return c.Conn.Close()
}

type grpcCredentials struct{}

func NewCredentials() credentials.TransportCredentials {
	return &grpcCredentials{}
}

func (c *grpcCredentials) ClientHandshake(_ context.Context, _ string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn.Close()
	return conn, AuthInfo{}, ErrInvalidConnection
}

func (c *grpcCredentials) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	wrappedCon, ok := conn.(*net.UnixConn)
	if !ok {
		conn.Close()
		log.Printf("invalid connection type: %T", conn)
		return conn, AuthInfo{}, ErrInvalidConnection
	}
	// get the caller's PID, UID, and GID
	sys, err := wrappedCon.SyscallConn()
	if err != nil {
		log.Printf("unable to get peer credentials: %v", err)
		conn.Close()
		return conn, AuthInfo{}, ErrInvalidConnection
	}

	auth := AuthInfo{}
	sys.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err != nil {
			log.Printf("unable to get peer credentials: %v", err)
			return
		}
		auth.Caller.PID = int32(cred.Pid)
		auth.Caller.UID = uint32(cred.Uid)
		auth.Caller.GID = uint32(cred.Gid)
	})

	// get binary path of the caller
	path, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", auth.Caller.PID))
	if err == nil {
		_, auth.Caller.BinaryName = filepath.Split(path)
	}

	return wrappedCon, AuthInfo{
		Caller: CallerInfo{
			PID:        auth.Caller.PID,
			UID:        auth.Caller.UID,
			GID:        auth.Caller.GID,
			BinaryName: auth.Caller.BinaryName,
		},
	}, nil
}

func (c *grpcCredentials) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "spire-attestation",
		SecurityVersion:  "0.2",
		ServerName:       "spire-agent",
	}
}

func (c *grpcCredentials) Clone() credentials.TransportCredentials {
	credentialsCopy := *c
	return &credentialsCopy
}

func (c *grpcCredentials) OverrideServerName(_ string) error {
	return nil
}

type CallerInfo struct {
	Addr       net.Addr
	PID        int32
	UID        uint32
	GID        uint32
	BinaryName string
}

type AuthInfo struct {
	Caller CallerInfo
}

// AuthType returns the authentication type and allows us to
// conform to the gRPC AuthInfo interface
func (AuthInfo) AuthType() string {
	return "spire-attestation"
}
