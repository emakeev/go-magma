/*
Copyright 2020 The Magma Authors.

This source code is licensed under the BSD-style license found in the
LICENSE file in the root directory of this source tree.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test_init

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/go-magma/magma/gateway/go/mconfig"
	"github.com/go-magma/magma/modules/feg/cloud/go/protos"
	"github.com/go-magma/magma/modules/feg/gateway/registry"
	"github.com/go-magma/magma/modules/feg/gateway/services/s6a_proxy/servicers"
	"github.com/go-magma/magma/modules/feg/gateway/services/s6a_proxy/servicers/test"
	"github.com/go-magma/magma/orc8r/cloud/go/test_utils"
)

func StartTestService(t *testing.T) error {
	srv, lis := test_utils.NewTestService(t, registry.ModuleName, registry.S6A_PROXY)

	diamAddr := fmt.Sprintf("127.0.0.1:%d", 30000+rand.Intn(1000))

	// Create tmp mconfig test file & load configs from it
	fegConfigFmt := `{
		"configsByKey": {
			"s6a_proxy": {
				"@type": "type.googleapis.com/magma.mconfig.S6aConfig",
				"logLevel": "INFO",
				"server": {
					"protocol": "sctp",
					"address": "%s",
					"retransmits": 3,
					"watchdogInterval": 1,
					"retryCount": 5,
					"productName": "magma_test",
					"realm": "openair4G.eur",
					"host": "magma-oai.openair4G.eur"
				},
				"requestFailureThreshold": 0.5,
   				"minimumRequestThreshold": 1
			},
			"session_proxy": {
				"@type": "type.googleapis.com/magma.mconfig.SessionProxyConfig",
				"logLevel": "INFO",
				"gx": {
					"disableGx": false,
					"server": {
						 "protocol": "tcp",
						 "address": "",
						 "retransmits": 3,
						 "watchdogInterval": 1,
						 "retryCount": 5,
						 "productName": "magma",
		 				"realm": "magma.com",
		 				"host": "magma-fedgw.magma.com"
					}
				},
				"gy": {
					"disableGy": false,
					"server": {
						 "protocol": "tcp",
						 "address": "",
						 "retransmits": 3,
						 "watchdogInterval": 1,
						 "retryCount": 5,
						 "productName": "magma",
		 				 "realm": "magma.com",
		 				 "host": "magma-fedgw.magma.com"
					},
					"initMethod": "PER_KEY"
				},
				"requestFailureThreshold": 0.5,
   				"minimumRequestThreshold": 1
			}
		}
	}`

	err := mconfig.CreateLoadTempConfig(fmt.Sprintf(fegConfigFmt, diamAddr))
	if err != nil {
		return err
	}
	clientCfg, serverCfg := servicers.GetS6aProxyConfigs()
	err = test.StartTestS6aServer(serverCfg.Protocol, serverCfg.Addr)
	if err != nil {
		return err
	}
	service, err := servicers.NewS6aProxy(clientCfg, serverCfg)
	if err != nil {
		return err
	}

	protos.RegisterS6AProxyServer(srv.GrpcServer, service)
	protos.RegisterServiceHealthServer(srv.GrpcServer, service)
	go srv.RunTest(lis)
	return nil
}
