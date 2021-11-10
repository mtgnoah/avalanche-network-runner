package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/ava-labs/avalanche-network-runner-local/api"
	"github.com/ava-labs/avalanche-network-runner-local/local/mocks"
	"github.com/ava-labs/avalanche-network-runner-local/network"
	"github.com/ava-labs/avalanche-network-runner-local/network/node"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/stretchr/testify/assert"
)

var _ NewNodeProcessF = newMockProcessUndef
var _ NewNodeProcessF = newMockProcessSuccessful
var _ NewNodeProcessF = newMockProcessFailedStart

func newMockProcessUndef(node.Config, ...string) (NodeProcess, error) {
	return &mocks.NodeProcess{}, nil
}

func newMockProcessSuccessful(node.Config, ...string) (NodeProcess, error) {
	process := &mocks.NodeProcess{}
	process.On("Start").Return(nil)
	process.On("Wait").Return(nil)
	process.On("Stop").Return(nil)
	return process, nil
}

func newMockProcessFailedStart(node.Config, ...string) (NodeProcess, error) {
	process := &mocks.NodeProcess{}
	process.On("Start").Return(errors.New("Start failed"))
	process.On("Wait").Return(nil)
	process.On("Stop").Return(nil)
	return process, nil
}

func TestNewNetworkEmpty(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	networkConfig.NodeConfigs = nil
	net, err := NewNetwork(
		logging.NoLog{},
		networkConfig,
		api.NewAPIClient, // TODO change AvalancheGo so we can mock API clients
		newMockProcessUndef,
	)
	assert.NoError(err)
	// Assert that GetNodesNames() includes only the 1 node's name
	names, err := net.GetNodesNames()
	assert.NoError(err)
	assert.Len(names, 0)
}

// Start a network with one node.
func TestNewNetworkOneNode(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	networkConfig.NodeConfigs = networkConfig.NodeConfigs[:1]
	// Assert that the node's config is being passed correctly
	// to the function that starts the node process.
	newProcessF := func(config node.Config, _ ...string) (NodeProcess, error) {
		assert.True(config.IsBeacon)
		assert.EqualValues(networkConfig.NodeConfigs[0], config)
		return newMockProcessSuccessful(config)
	}
	net, err := NewNetwork(
		logging.NoLog{},
		networkConfig,
		api.NewAPIClient,
		newProcessF,
	)
	assert.NoError(err)
	// Assert that GetNodesNames() includes only the 1 node's name
	names, err := net.GetNodesNames()
	assert.NoError(err)
	assert.Contains(names, networkConfig.NodeConfigs[0].Name)
	assert.Len(names, 1)
}

func TestWrongNetworkConfigs(t *testing.T) {
	tests := map[string]struct {
		input network.Config
	}{
		"no ImplSpecificConfig": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					IsBeacon:    true,
					GenesisFile: []byte("nonempty"),
					ConfigFile:  []byte("nonempty"),
				},
			},
		}},
		"no ConfigFile": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					IsBeacon:    true,
					GenesisFile: []byte("nonempty"),
				},
			},
		}},
		"no GenesisFile": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					IsBeacon:   true,
					ConfigFile: []byte("nonempty"),
				},
			},
		}},
		"StakingKey but no StakingCert": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					IsBeacon:    true,
					GenesisFile: []byte("nonempty"),
					ConfigFile:  []byte("nonempty"),
					StakingKey:  []byte("nonempty"),
				},
			},
		}},
		"StakingCert but no StakingKey": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					IsBeacon:    true,
					GenesisFile: []byte("nonempty"),
					ConfigFile:  []byte("nonempty"),
					StakingCert: []byte("nonempty"),
				},
			},
		}},
		"no beacon node": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					GenesisFile: []byte("nonempty"),
					ConfigFile:  []byte("nonempty"),
				},
			},
		}},
		"repeated name": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					Name: "node0",
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					IsBeacon:    true,
					GenesisFile: []byte("nonempty"),
					ConfigFile:  []byte("nonempty"),
				},
				{
					Name: "node0",
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					IsBeacon:    true,
					GenesisFile: []byte("nonempty"),
					ConfigFile:  []byte("nonempty"),
				},
			},
		}},
		"invalid cert/key format": {input: network.Config{
			NodeConfigs: []node.Config{
				{
					ImplSpecificConfig: NodeConfig{
						BinaryPath: "pepe",
					},
					IsBeacon:    true,
					GenesisFile: []byte("nonempty"),
					ConfigFile:  []byte("nonempty"),
					StakingCert: []byte("nonempty"),
					StakingKey:  []byte("nonempty"),
				},
			},
		}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			_, err := NewNetwork(logging.NoLog{}, tc.input, api.NewAPIClient, newMockProcessSuccessful)
			assert.Error(err)
		})
	}
}

func TestImplSpecificConfigInterface(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	networkConfig.NodeConfigs[0].ImplSpecificConfig = "should not be string"
	_, err = NewNetwork(logging.NoLog{}, networkConfig, api.NewAPIClient, newMockProcessSuccessful)
	assert.Error(err)
}

func TestUnhealthyNetwork(t *testing.T) {
	t.Skip()
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	_, err = NewNetwork(logging.NoLog{}, networkConfig, api.NewAPIClient, newMockProcessSuccessful)
	assert.NoError(err)
	// TODO: needs to set fake fail health api
	//assert.Error(awaitNetworkHealthy(net))
}

func TestGeneratedNodesNames(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	for i := range networkConfig.NodeConfigs {
		networkConfig.NodeConfigs[i].Name = ""
	}
	net, err := NewNetwork(logging.NoLog{}, networkConfig, api.NewAPIClient, newMockProcessSuccessful)
	assert.NoError(err)
	nodeNameMap := make(map[string]bool)
	nodeNames, err := net.GetNodesNames()
	assert.NoError(err)
	for _, nodeName := range nodeNames {
		nodeNameMap[nodeName] = true
	}
	assert.EqualValues(len(nodeNameMap), len(networkConfig.NodeConfigs))
}

// TODO add byzantine node to conf
// TestNetworkFromConfig creates/waits/checks/stops a network from config file
// the check verify that all the nodes api clients are up
func TestNetworkFromConfig(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	net, err := NewNetwork(logging.NoLog{}, networkConfig, api.NewAPIClient, newMockProcessSuccessful)
	assert.NoError(err)
	// TODO: needs to set fake successful health api
	//assert.NoError(awaitNetworkHealthy(net))
	runningNodes := make(map[string]bool)
	for _, nodeConfig := range networkConfig.NodeConfigs {
		runningNodes[nodeConfig.Name] = true
	}
	checkNetwork(t, net, runningNodes, nil)
}

// TestNetworkNodeOps creates/waits/checks/stops a network created from an empty one
// nodes are first added one by one, then removed one by one. between all operations, a network check is performed
// the check verify that all the nodes api clients are up for started nodes, and down for removed nodes
// all nodes are taken from config file
func TestNetworkNodeOps(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	net, err := NewNetwork(logging.NoLog{}, network.Config{}, api.NewAPIClient, newMockProcessSuccessful)
	assert.NoError(err)
	runningNodes := make(map[string]bool)
	for _, nodeConfig := range networkConfig.NodeConfigs {
		_, err = net.AddNode(nodeConfig)
		assert.NoError(err)
		runningNodes[nodeConfig.Name] = true
		checkNetwork(t, net, runningNodes, nil)
	}
	// TODO: needs to set fake successful health api
	//assert.NoError(awaitNetworkHealthy(net))
	removedNodes := make(map[string]bool)
	for _, nodeConfig := range networkConfig.NodeConfigs {
		_, err := net.GetNode(nodeConfig.Name)
		assert.NoError(err)
		err = net.RemoveNode(nodeConfig.Name)
		assert.NoError(err)
		removedNodes[nodeConfig.Name] = true
		delete(runningNodes, nodeConfig.Name)
		checkNetwork(t, net, runningNodes, removedNodes)
	}
}

// TestNodeNotFound checks operations fail for unkown node
func TestNodeNotFound(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	net, err := NewNetwork(logging.NoLog{}, network.Config{}, api.NewAPIClient, newMockProcessSuccessful)
	assert.NoError(err)
	_, err = net.AddNode(networkConfig.NodeConfigs[0])
	assert.NoError(err)
	// get correct node
	_, err = net.GetNode(networkConfig.NodeConfigs[0].Name)
	assert.NoError(err)
	// get uncorrect node (non created)
	_, err = net.GetNode(networkConfig.NodeConfigs[1].Name)
	assert.Error(err)
	// remove uncorrect node (non created)
	err = net.RemoveNode(networkConfig.NodeConfigs[1].Name)
	assert.Error(err)
	// remove correct node
	err = net.RemoveNode(networkConfig.NodeConfigs[0].Name)
	assert.NoError(err)
	// get uncorrect node (removed)
	_, err = net.GetNode(networkConfig.NodeConfigs[0].Name)
	assert.Error(err)
	// remove uncorrect node (removed)
	err = net.RemoveNode(networkConfig.NodeConfigs[0].Name)
	assert.Error(err)
}

// TestStoppedNetwork checks operations fail for an already stopped network
func TestStoppedNetwork(t *testing.T) {
	assert := assert.New(t)
	networkConfig, err := defaultNetworkConfig()
	assert.NoError(err)
	net, err := NewNetwork(logging.NoLog{}, network.Config{}, api.NewAPIClient, newMockProcessSuccessful)
	assert.NoError(err)
	_, err = net.AddNode(networkConfig.NodeConfigs[0])
	assert.NoError(err)
	// first GetNodesNames should return some nodes
	_, err = net.GetNodesNames()
	assert.NoError(err)
	err = net.Stop(context.TODO())
	assert.NoError(err)
	// Stop failure
	assert.EqualValues(net.Stop(context.TODO()), errStopped)
	// AddNode failure
	_, err = net.AddNode(networkConfig.NodeConfigs[1])
	assert.EqualValues(err, errStopped)
	// GetNode failure
	_, err = net.GetNode(networkConfig.NodeConfigs[0].Name)
	assert.EqualValues(err, errStopped)
	// second GetNodesNames should return no nodes
	_, err = net.GetNodesNames()
	assert.EqualValues(err, errStopped)
	// RemoveNode failure
	assert.EqualValues(net.RemoveNode(networkConfig.NodeConfigs[0].Name), errStopped)
	// Healthy failure
	assert.EqualValues(awaitNetworkHealthy(net), errStopped)
}

func checkNetwork(t *testing.T, net network.Network, runningNodes map[string]bool, removedNodes map[string]bool) {
	assert := assert.New(t)
	nodeNames, err := net.GetNodesNames()
	assert.NoError(err)
	assert.EqualValues(len(nodeNames), len(runningNodes))
	for nodeName := range runningNodes {
		_, err := net.GetNode(nodeName)
		assert.NoError(err)
	}
	for nodeName := range removedNodes {
		_, err := net.GetNode(nodeName)
		assert.Error(err)
	}
}

func awaitNetworkHealthy(net network.Network) error {
	healthyCh := net.Healthy()
	err, ok := <-healthyCh
	if ok {
		return err
	}
	return nil
}

func defaultNetworkConfig() (network.Config, error) {
	// TODO remove test files when we can auto-generate genesis
	// and other files
	networkConfig := network.Config{
		LogLevel: "DEBUG",
		Name:     "My Network",
	}
	genesisFile, err := os.ReadFile("test_files/genesis.json")
	if err != nil {
		return networkConfig, err
	}
	cchainConfigFile, err := os.ReadFile("test_files/cchain_config.json")
	if err != nil {
		return networkConfig, err
	}
	for i := 1; i <= 6; i++ {
		nodeConfig := node.Config{
			GenesisFile:      genesisFile,
			CChainConfigFile: cchainConfigFile,
			Name:             fmt.Sprintf("node%d", i),
		}
		configFile, err := os.ReadFile(fmt.Sprintf("test_files/node%d/config.json", i))
		if err != nil {
			return networkConfig, err
		}
		nodeConfig.ConfigFile = configFile
		certFile, err := os.ReadFile(fmt.Sprintf("test_files/node%d/staking.crt", i))
		if err == nil {
			nodeConfig.StakingCert = certFile
		}
		keyFile, err := os.ReadFile(fmt.Sprintf("test_files/node%d/staking.key", i))
		if err == nil {
			nodeConfig.StakingKey = keyFile
		}
		localNodeConf := NodeConfig{
			BinaryPath: "pepito",
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
		}
		nodeConfig.ImplSpecificConfig = localNodeConf
		networkConfig.NodeConfigs = append(networkConfig.NodeConfigs, nodeConfig)
	}
	networkConfig.NodeConfigs[0].IsBeacon = true
	return networkConfig, nil
}
