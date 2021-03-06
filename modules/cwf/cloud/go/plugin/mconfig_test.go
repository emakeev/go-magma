/*
 * Copyright 2020 The Magma Authors.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree.
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package plugin_test

import (
	"testing"

	"github.com/go-magma/magma/lib/go/protos"
	orcmconfig "github.com/go-magma/magma/lib/go/protos/mconfig"
	"github.com/go-magma/magma/modules/cwf/cloud/go/cwf"
	"github.com/go-magma/magma/modules/cwf/cloud/go/plugin"
	cwfmconfig "github.com/go-magma/magma/modules/cwf/cloud/go/protos/mconfig"
	"github.com/go-magma/magma/modules/cwf/cloud/go/services/cwf/obsidian/models"
	fegmconfig "github.com/go-magma/magma/modules/feg/cloud/go/protos/mconfig"
	ltemconfig "github.com/go-magma/magma/modules/lte/cloud/go/protos/mconfig"
	"github.com/go-magma/magma/orc8r/cloud/go/orc8r"
	orc8rplugin "github.com/go-magma/magma/orc8r/cloud/go/plugin"
	"github.com/go-magma/magma/orc8r/cloud/go/services/configurator"
	"github.com/go-magma/magma/orc8r/cloud/go/storage"

	"github.com/go-openapi/swag"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestBuilder_Build(t *testing.T) {
	orc8rplugin.RegisterPluginForTests(t, &plugin.CwfOrchestratorPlugin{})
	builder := &plugin.Builder{}

	// empty case: no cwf associated to magmad gateway
	nw := configurator.Network{ID: "n1"}
	gw := configurator.NetworkEntity{
		Type: orc8r.MagmadGatewayType, Key: "gw1",
		Associations: []storage.TypeAndKey{
			{Type: cwf.CwfGatewayType, Key: "gw1"},
		},
	}
	graph := configurator.EntityGraph{
		Entities: []configurator.NetworkEntity{gw},
	}

	actual := map[string]proto.Message{}
	expected := map[string]proto.Message{}
	err := builder.Build("n1", "gw1", graph, nw, actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// Network config exists
	nw.Configs = map[string]interface{}{
		cwf.CwfNetworkType: defaultnwConfig,
	}
	cwfGW := configurator.NetworkEntity{
		Type: cwf.CwfGatewayType, Key: "gw1",
		Config:             defaultgwConfig,
		ParentAssociations: []storage.TypeAndKey{gw.GetTypeAndKey()},
	}
	graph = configurator.EntityGraph{
		Entities: []configurator.NetworkEntity{cwfGW, gw},
		Edges: []configurator.GraphEdge{
			{From: gw.GetTypeAndKey(), To: cwfGW.GetTypeAndKey()},
		},
	}
	actual = map[string]proto.Message{}
	expected = map[string]proto.Message{
		"eap_aka": &fegmconfig.EapAkaConfig{LogLevel: 1,
			Timeout: &fegmconfig.EapAkaConfig_Timeouts{
				ChallengeMs:            20000,
				ErrorNotificationMs:    10000,
				SessionMs:              43200000,
				SessionAuthenticatedMs: 5000,
			},
			PlmnIds: nil,
		},
		"aaa_server": &fegmconfig.AAAConfig{LogLevel: 1,
			IdleSessionTimeoutMs: 21600000,
			AccountingEnabled:    false,
			CreateSessionOnAuth:  false,
		},
		"pipelined": &ltemconfig.PipelineD{
			LogLevel:      protos.LogLevel_INFO,
			UeIpBlock:     "192.168.128.0/24", // Unused by CWF
			NatEnabled:    false,
			DefaultRuleId: "",
			Services: []ltemconfig.PipelineD_NetworkServices{
				ltemconfig.PipelineD_DPI,
				ltemconfig.PipelineD_ENFORCEMENT,
			},
			AllowedGrePeers: []*ltemconfig.PipelineD_AllowedGrePeer{
				{Ip: "1.2.3.4/24"},
				{Ip: "1.1.1.1/24", Key: 111},
			},
			LiUes: &ltemconfig.PipelineD_LiUes{
				Imsis:   []string{"IMSI001010000000013"},
				Ips:     []string{"192.16.8.1"},
				Macs:    []string{"00:33:bb:aa:cc:33"},
				Msisdns: []string{"57192831"},
			},
			IpdrExportDst: &ltemconfig.PipelineD_IPDRExportDst{
				Ip:   "192.168.128.88",
				Port: 2040,
			},
		},
		"sessiond": &ltemconfig.SessionD{
			LogLevel:     protos.LogLevel_INFO,
			RelayEnabled: true,
			WalletExhaustDetection: &ltemconfig.WalletExhaustDetection{
				TerminateOnExhaust: true,
				Method:             ltemconfig.WalletExhaustDetection_GxTrackedRules,
			},
		},
		"redirectd": &ltemconfig.RedirectD{
			LogLevel: protos.LogLevel_INFO,
		},
		"directoryd": &orcmconfig.DirectoryD{
			LogLevel: protos.LogLevel_INFO,
		},
		"health": &cwfmconfig.CwfGatewayHealthConfig{
			CpuUtilThresholdPct: 0,
			MemUtilThresholdPct: 0,
			GreProbeInterval:    0,
			IcmpProbePktCount:   0,
			GrePeers: []*cwfmconfig.CwfGatewayHealthConfigGrePeer{
				{Ip: "1.2.3.4/24"},
				{Ip: "1.1.1.1/24"},
			},
		},
	}
	err = builder.Build("n1", "gw1", graph, nw, actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

var defaultnwConfig = &models.NetworkCarrierWifiConfigs{
	EapAka: &models.EapAka{
		Timeout: &models.EapAkaTimeout{
			ChallengeMs:            20000,
			ErrorNotificationMs:    10000,
			SessionMs:              43200000,
			SessionAuthenticatedMs: 5000,
		},
		PlmnIds: nil,
	},
	AaaServer: &models.AaaServer{
		IDLESessionTimeoutMs: 21600000,
		AccountingEnabled:    false,
		CreateSessionOnAuth:  false,
	},
	NetworkServices: []string{"dpi", "policy_enforcement"},
	LiUes: &models.LiUes{
		Imsis:   []string{"IMSI001010000000013"},
		Ips:     []string{"192.16.8.1"},
		Macs:    []string{"00:33:bb:aa:cc:33"},
		Msisdns: []string{"57192831"},
	},
	DefaultRuleID: swag.String(""),
}

var defaultgwConfig = &models.GatewayCwfConfigs{
	AllowedGrePeers: models.AllowedGrePeers{
		{IP: "1.2.3.4/24"},
		{IP: "1.1.1.1/24", Key: swag.Uint32(111)},
	},
	IPDRExportDst: &models.IPDRExportDst{
		IP:   "192.168.128.88",
		Port: 2040,
	},
}
