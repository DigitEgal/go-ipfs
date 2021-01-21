package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	config "github.com/ipfs/go-ipfs-config"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	"github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

type testPinMFSContext struct {
	ctx context.Context
	cfg *config.Config
	err error
}

func (x *testPinMFSContext) Context() context.Context {
	return x.ctx
}

func (x *testPinMFSContext) GetConfigNoCache() (*config.Config, error) {
	return x.cfg, x.err
}

type testPinMFSNode struct {
	err error
}

func (x *testPinMFSNode) RootNode() (ipld.Node, error) {
	return merkledag.NewRawNode([]byte{0x01}), x.err
}

func (x *testPinMFSNode) Identity() peer.ID {
	return peer.ID("test_id")
}

func (x *testPinMFSNode) PeerHost() host.Host {
	return nil
}

var testConfigPollInterval = time.Second

func TestPinMFSConfigError(t *testing.T) {
	ctx := &testPinMFSContext{
		ctx: context.Background(),
		cfg: nil,
		err: fmt.Errorf("couldn't read config"),
	}
	node := &testPinMFSNode{}
	errCh := make(chan error)
	go func() {
		if pinMFSOnChange(testConfigPollInterval, ctx, node, errCh) != nil {
			t.Errorf("pinMFSOnChange should always return no error")
		}
	}()
	if <-errCh != ctx.err {
		t.Errorf("error did not propagate")
	}
	if <-errCh != ctx.err {
		t.Errorf("error did not propagate")
	}
}

func TestPinMFSRootNodeError(t *testing.T) {
	ctx := &testPinMFSContext{
		ctx: context.Background(),
		cfg: &config.Config{
			Pinning: config.Pinning{},
		},
		err: nil,
	}
	node := &testPinMFSNode{
		err: fmt.Errorf("cannot create root node"),
	}
	errCh := make(chan error)
	go func() {
		if pinMFSOnChange(testConfigPollInterval, ctx, node, errCh) != nil {
			t.Errorf("pinMFSOnChange should always return no error")
		}
	}()
	if <-errCh != node.err {
		t.Errorf("error did not propagate")
	}
	if <-errCh != node.err {
		t.Errorf("error did not propagate")
	}
}

func TestPinMFSService(t *testing.T) {
	cfg_invalid_interval := &config.Config{
		Pinning: config.Pinning{
			RemoteServices: map[string]config.RemotePinningService{
				"disabled": {
					Policies: config.RemotePinningServicePolicies{
						MFS: config.RemotePinningServiceMFSPolicy{
							Enable: false,
						},
					},
				},
				"invalid_interval": {
					Policies: config.RemotePinningServicePolicies{
						MFS: config.RemotePinningServiceMFSPolicy{
							Enable:        true,
							RepinInterval: "INVALID_INTERVAL",
						},
					},
				},
			},
		},
	}
	cfg_valid_unnamed := &config.Config{
		Pinning: config.Pinning{
			RemoteServices: map[string]config.RemotePinningService{
				"valid_unnamed": {
					Policies: config.RemotePinningServicePolicies{
						MFS: config.RemotePinningServiceMFSPolicy{
							Enable:        true,
							PinName:       "",
							RepinInterval: "2s",
						},
					},
				},
			},
		},
	}
	cfg_valid_named := &config.Config{
		Pinning: config.Pinning{
			RemoteServices: map[string]config.RemotePinningService{
				"valid_named": {
					Policies: config.RemotePinningServicePolicies{
						MFS: config.RemotePinningServiceMFSPolicy{
							Enable:        true,
							PinName:       "pin_name",
							RepinInterval: "2s",
						},
					},
				},
			},
		},
	}
	testPinMFSServiceWithError(t, cfg_invalid_interval, "remote pinning service invalid_interval has invalid MFS.RepinInterval")
	testPinMFSServiceWithError(t, cfg_valid_unnamed, "error while listing remote pins: empty response from remote pinning service")
	testPinMFSServiceWithError(t, cfg_valid_named, "error while listing remote pins: empty response from remote pinning service")
}

func testPinMFSServiceWithError(t *testing.T, cfg *config.Config, expectedErrorPrefix string) {
	goctx, cancel := context.WithCancel(context.Background())
	ctx := &testPinMFSContext{
		ctx: goctx,
		cfg: cfg,
		err: nil,
	}
	node := &testPinMFSNode{
		err: nil,
	}
	errCh := make(chan error)
	go func() {
		if pinMFSOnChange(testConfigPollInterval, ctx, node, errCh) != nil {
			t.Errorf("pinMFSOnChange should always return no error")
		}
	}()
	defer cancel()
	// first pass through the pinning loop
	err := <-errCh
	if !strings.HasPrefix((err).Error(), expectedErrorPrefix) {
		t.Errorf("expecting error with prefix %q", expectedErrorPrefix)
	}
	// second pass through the pinning loop
	if !strings.HasPrefix((err).Error(), expectedErrorPrefix) {
		t.Errorf("expecting error with prefix %q", expectedErrorPrefix)
	}
}
