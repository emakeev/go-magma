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

package handlers_test

import (
	"context"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/go-magma/magma/lib/go/protos"
	"github.com/go-magma/magma/lib/go/security/key"
	"github.com/go-magma/magma/modules/lte/cloud/go/lte"
	ltePlugin "github.com/go-magma/magma/modules/lte/cloud/go/plugin"
	"github.com/go-magma/magma/modules/lte/cloud/go/services/lte/obsidian/handlers"
	lteModels "github.com/go-magma/magma/modules/lte/cloud/go/services/lte/obsidian/models"
	policyModels "github.com/go-magma/magma/modules/lte/cloud/go/services/policydb/obsidian/models"
	"github.com/go-magma/magma/orc8r/cloud/go/clock"
	"github.com/go-magma/magma/orc8r/cloud/go/obsidian"
	"github.com/go-magma/magma/orc8r/cloud/go/obsidian/tests"
	"github.com/go-magma/magma/orc8r/cloud/go/orc8r"
	"github.com/go-magma/magma/orc8r/cloud/go/plugin"
	"github.com/go-magma/magma/orc8r/cloud/go/pluginimpl"
	"github.com/go-magma/magma/orc8r/cloud/go/serde"
	"github.com/go-magma/magma/orc8r/cloud/go/services/configurator"
	"github.com/go-magma/magma/orc8r/cloud/go/services/configurator/test_init"
	"github.com/go-magma/magma/orc8r/cloud/go/services/device"
	deviceTestInit "github.com/go-magma/magma/orc8r/cloud/go/services/device/test_init"
	"github.com/go-magma/magma/orc8r/cloud/go/services/orchestrator/obsidian/models"
	"github.com/go-magma/magma/orc8r/cloud/go/services/state"
	stateTestInit "github.com/go-magma/magma/orc8r/cloud/go/services/state/test_init"
	"github.com/go-magma/magma/orc8r/cloud/go/services/state/test_utils"
	"github.com/go-magma/magma/orc8r/cloud/go/storage"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
)

func TestListNetworks(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)
	e := echo.New()

	obsidianHandlers := handlers.GetHandlers()
	listNetworks := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte", obsidian.GET).HandlerFunc

	// Test empty response
	tc := tests.Test{
		Method:         "GET",
		URL:            "/magma/v1/lte",
		Payload:        nil,
		Handler:        listNetworks,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler([]string{}),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	seedNetworks(t)

	tc = tests.Test{
		Method:         "GET",
		URL:            "/magma/v1/lte",
		Payload:        nil,
		Handler:        listNetworks,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler([]string{"n1", "n3"}),
	}
	tests.RunUnitTest(t, e, tc)
}

func TestCreateNetwork(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)
	e := echo.New()

	obsidianHandlers := handlers.GetHandlers()
	createNetwork := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte", obsidian.POST).HandlerFunc

	// test validation - include TDD and FDD configs
	payload := &lteModels.LteNetwork{
		Cellular:    lteModels.NewDefaultTDDNetworkConfig(),
		Description: "blah",
		DNS:         models.NewDefaultDNSConfig(),
		Features:    models.NewDefaultFeaturesConfig(),
		ID:          "n1",
		Name:        "foobar",
	}
	payload.Cellular.Ran.FddConfig = &lteModels.NetworkRanConfigsFddConfig{
		Earfcndl: 17000,
		Earfcnul: 18000,
	}
	tc := tests.Test{
		Method:         "POST",
		URL:            "/magma/v1/lte",
		Payload:        payload,
		Handler:        createNetwork,
		ExpectedStatus: 400,
		ExpectedError: "validation failure list:\n" +
			"only one of TDD or FDD configs can be set",
	}
	tests.RunUnitTest(t, e, tc)

	// happy path
	payload = &lteModels.LteNetwork{
		Cellular:    lteModels.NewDefaultTDDNetworkConfig(),
		Description: "Foo Bar",
		DNS:         models.NewDefaultDNSConfig(),
		Features:    models.NewDefaultFeaturesConfig(),
		ID:          "n1",
		Name:        "foobar",
	}
	tc = tests.Test{
		Method:         "POST",
		URL:            "/magma/v1/lte",
		Payload:        payload,
		Handler:        createNetwork,
		ExpectedStatus: 201,
	}
	tests.RunUnitTest(t, e, tc)

	actual, err := configurator.LoadNetwork("n1", true, true)
	assert.NoError(t, err)
	expected := configurator.Network{
		ID:          "n1",
		Type:        lte.NetworkType,
		Name:        "foobar",
		Description: "Foo Bar",
		Configs: map[string]interface{}{
			lte.CellularNetworkType:     lteModels.NewDefaultTDDNetworkConfig(),
			orc8r.DnsdNetworkType:       models.NewDefaultDNSConfig(),
			orc8r.NetworkFeaturesConfig: models.NewDefaultFeaturesConfig(),
		},
	}
	assert.Equal(t, expected, actual)
}

func TestGetNetwork(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)
	e := echo.New()

	obsidianHandlers := handlers.GetHandlers()
	getNetwork := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id", obsidian.GET).HandlerFunc

	// Test 404
	tc := tests.Test{
		Method:         "GET",
		URL:            "/magma/v1/lte/n1",
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getNetwork,
		ExpectedStatus: 404,
	}
	tests.RunUnitTest(t, e, tc)

	seedNetworks(t)

	expectedN1 := &lteModels.LteNetwork{
		Cellular:    lteModels.NewDefaultTDDNetworkConfig(),
		Description: "Foo Bar",
		DNS:         models.NewDefaultDNSConfig(),
		Features:    models.NewDefaultFeaturesConfig(),
		ID:          "n1",
		Name:        "foobar",
	}
	tc = tests.Test{
		Method:         "GET",
		URL:            "/magma/v1/lte/n1",
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getNetwork,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(expectedN1),
	}
	tests.RunUnitTest(t, e, tc)

	// get a non-LTE network
	tc = tests.Test{
		Method:         "GET",
		URL:            "/magma/v1/lte/n2",
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        getNetwork,
		ExpectedStatus: 400,
		ExpectedError:  "network n2 is not a <lte> network",
	}
	tests.RunUnitTest(t, e, tc)

	// get a network without any configs (poorly formed data)
	expectedN3 := &lteModels.LteNetwork{
		Description: "Bar Foo",
		ID:          "n3",
		Name:        "barfoo",
	}
	tc = tests.Test{
		Method:         "GET",
		URL:            "/magma/v1/lte/n3",
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n3"},
		Handler:        getNetwork,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(expectedN3),
	}
	tests.RunUnitTest(t, e, tc)
}

func TestUpdateNetwork(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)
	e := echo.New()

	obsidianHandlers := handlers.GetHandlers()
	updateNetwork := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id", obsidian.PUT).HandlerFunc

	// Test validation failure
	payloadN1 := &lteModels.LteNetwork{
		ID:          "n1",
		Name:        "updated foobar",
		Description: "Updated Foo Bar",
		Cellular:    lteModels.NewDefaultFDDNetworkConfig(),
		Features: &models.NetworkFeatures{
			Features: map[string]string{
				"bar": "baz",
				"baz": "quz",
			},
		},
		DNS: &models.NetworkDNSConfig{
			EnableCaching: swag.Bool(true),
			LocalTTL:      swag.Uint32(120),
			Records: []*models.DNSConfigRecord{
				{
					Domain:     "foobar.com",
					ARecord:    []strfmt.IPv4{"asdf", "hjkl"},
					AaaaRecord: []strfmt.IPv6{"abcd", "efgh"},
				},
				{
					Domain:  "facebook.com",
					ARecord: []strfmt.IPv4{"google.com"},
				},
			},
		},
	}
	tc := tests.Test{
		Method:         "PUT",
		URL:            "/magma/v1/lte/n1",
		Payload:        payloadN1,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        updateNetwork,
		ExpectedStatus: 400,
		ExpectedError: "validation failure list:\n" +
			"validation failure list:\n" +
			"validation failure list:\n" +
			"a_record.0 in body must be of type ipv4: \"asdf\"\n" +
			"aaaa_record.0 in body must be of type ipv6: \"abcd\"",
	}
	tests.RunUnitTest(t, e, tc)

	payloadN1.DNS.Records = []*models.DNSConfigRecord{
		{
			Domain:  "foobar.com",
			ARecord: []strfmt.IPv4{"127.0.0.1", "127.0.0.2"},
			AaaaRecord: []strfmt.IPv6{
				"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
				"1234:0db8:85a3:0000:0000:8a2e:0370:1234",
			},
		},
		{
			Domain:  "facebook.com",
			ARecord: []strfmt.IPv4{"127.0.0.3"},
		},
	}
	// Test 404
	tc = tests.Test{
		Method:         "PUT",
		URL:            "/magma/v1/lte/n1",
		Payload:        payloadN1,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        updateNetwork,
		ExpectedStatus: 404,
	}
	tests.RunUnitTest(t, e, tc)

	// seed networks, update n1 again
	seedNetworks(t)

	tc = tests.Test{
		Method:         "PUT",
		URL:            "/magma/v1/lte/n1",
		Payload:        payloadN1,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        updateNetwork,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actualN1, err := configurator.LoadNetwork("n1", true, true)
	assert.NoError(t, err)
	expected := configurator.Network{
		ID:          "n1",
		Type:        lte.NetworkType,
		Name:        "updated foobar",
		Description: "Updated Foo Bar",
		Configs: map[string]interface{}{
			lte.CellularNetworkType:     lteModels.NewDefaultFDDNetworkConfig(),
			orc8r.DnsdNetworkType:       payloadN1.DNS,
			orc8r.NetworkFeaturesConfig: payloadN1.Features,
		},
		Version: 1,
	}
	assert.Equal(t, expected, actualN1)

	// update n2, should be 400
	tc = tests.Test{
		Method:         "PUT",
		URL:            "/magma/v1/lte/n2",
		Payload:        payloadN1,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        updateNetwork,
		ExpectedStatus: 400,
		ExpectedError:  "network n2 is not a <lte> network",
	}
	tests.RunUnitTest(t, e, tc)
}

func TestDeleteNetwork(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)
	e := echo.New()

	obsidianHandlers := handlers.GetHandlers()
	deleteNetwork := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id", obsidian.DELETE).HandlerFunc

	// Test 404
	tc := tests.Test{
		Method:         "DELETE",
		URL:            "/magma/v1/lte/n1",
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        deleteNetwork,
		ExpectedStatus: 404,
	}
	tests.RunUnitTest(t, e, tc)

	// seed networks, delete n1 again
	seedNetworks(t)
	tc.ExpectedStatus = 204
	tests.RunUnitTest(t, e, tc)

	// delete n1 again, should be 404
	tc.ExpectedStatus = 404
	tests.RunUnitTest(t, e, tc)

	// try to delete n2, should be 400 (not LTE network)
	tc = tests.Test{
		Method:         "DELETE",
		URL:            "/magma/v1/lte/n2",
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        deleteNetwork,
		ExpectedStatus: 400,
		ExpectedError:  "network n2 is not a <lte> network",
	}
	tests.RunUnitTest(t, e, tc)

	actual, err := configurator.ListNetworkIDs()
	assert.NoError(t, err)
	assert.Equal(t, []string{"n2", "n3"}, actual)
}

func TestCellularPartialGet(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)

	e := echo.New()
	testURLRoot := "/magma/v1/lte"

	seedNetworks(t)

	handlers := handlers.GetHandlers()
	getCellular := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular", testURLRoot), obsidian.GET).HandlerFunc
	getEpc := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular/epc", testURLRoot), obsidian.GET).HandlerFunc
	getRan := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular/ran", testURLRoot), obsidian.GET).HandlerFunc
	getFegNetworkID := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular/feg_network_id", testURLRoot), obsidian.GET).HandlerFunc

	// happy path
	tc := tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/cellular/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getCellular,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(lteModels.NewDefaultTDDNetworkConfig()),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	// 404
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/cellular/", testURLRoot, "n2"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        getCellular,
		ExpectedStatus: 404,
		ExpectedError:  "Not found",
	}
	tests.RunUnitTest(t, e, tc)

	// happy path
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/cellular/epc/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getEpc,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(lteModels.NewDefaultTDDNetworkConfig().Epc),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	// 404
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/cellular/epc/", testURLRoot, "n2"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        getEpc,
		ExpectedStatus: 404,
		ExpectedError:  "Not found",
	}
	tests.RunUnitTest(t, e, tc)

	// happy path
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/cellular/ran/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getRan,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(lteModels.NewDefaultTDDNetworkConfig().Ran),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	// 404
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/cellular/ran/", testURLRoot, "n2"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        getRan,
		ExpectedStatus: 404,
		ExpectedError:  "Not found",
	}
	tests.RunUnitTest(t, e, tc)

	// add 'n2' as FegNetworkID to n1
	cellularConfig := lteModels.NewDefaultTDDNetworkConfig()
	cellularConfig.FegNetworkID = "n2"
	err := configurator.UpdateNetworks([]configurator.NetworkUpdateCriteria{
		{
			ID: "n1",
			ConfigsToAddOrUpdate: map[string]interface{}{
				lte.CellularNetworkType: cellularConfig,
			},
		},
	})
	assert.NoError(t, err)

	// happy case FegNetworkID from cellular config
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/cellular/feg_network_id/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getFegNetworkID,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler("n2"),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)
}

func TestCellularPartialUpdate(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)

	e := echo.New()
	testURLRoot := "/magma/v1/lte"

	seedNetworks(t)
	handlers := handlers.GetHandlers()
	updateCellular := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular", testURLRoot), obsidian.PUT).HandlerFunc
	updateEpc := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular/epc", testURLRoot), obsidian.PUT).HandlerFunc
	updateRan := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular/ran", testURLRoot), obsidian.PUT).HandlerFunc
	updateFegNetworkID := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular/feg_network_id", testURLRoot), obsidian.PUT).HandlerFunc

	// happy path update cellular config
	tc := tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/", testURLRoot, "n2"),
		Payload:        lteModels.NewDefaultFDDNetworkConfig(),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        updateCellular,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actualN2, err := configurator.LoadNetwork("n2", true, true)
	assert.NoError(t, err)
	expected := configurator.Network{
		ID:          "n2",
		Type:        "blah",
		Name:        "foobar",
		Description: "Foo Bar",
		Configs: map[string]interface{}{
			lte.CellularNetworkType: lteModels.NewDefaultFDDNetworkConfig(),
		},
		Version: 1,
	}
	assert.Equal(t, expected, actualN2)

	// Validation error (celullar config has both tdd and fdd config)
	badCellularConfig := lteModels.NewDefaultTDDNetworkConfig()
	badCellularConfig.Ran.FddConfig = &lteModels.NetworkRanConfigsFddConfig{
		Earfcndl: 1,
		Earfcnul: 18001,
	}
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/", testURLRoot, "n2"),
		Payload:        badCellularConfig,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        updateCellular,
		ExpectedStatus: 400,
		ExpectedError:  "only one of TDD or FDD configs can be set",
	}
	tests.RunUnitTest(t, e, tc)

	// Fail to put epc config to a network without cellular network configs
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/epc/", testURLRoot, "n3"),
		Payload:        lteModels.NewDefaultTDDNetworkConfig().Epc,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n3"},
		Handler:        updateEpc,
		ExpectedStatus: 400,
		ExpectedError:  "No cellular network config found",
	}
	tests.RunUnitTest(t, e, tc)

	// happy path update epc config
	epcConfig := lteModels.NewDefaultTDDNetworkConfig().Epc
	epcConfig.RelayEnabled = swag.Bool(true)
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/epc/", testURLRoot, "n2"),
		Payload:        epcConfig,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        updateEpc,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actualN2, err = configurator.LoadNetwork("n2", true, true)
	assert.NoError(t, err)
	expected.Configs[lte.CellularNetworkType].(*lteModels.NetworkCellularConfigs).Epc = epcConfig
	expected.Version = 2
	assert.Equal(t, expected, actualN2)

	// Fail to put epc config to a network without cellular network configs
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/ran/", testURLRoot, "n3"),
		Payload:        lteModels.NewDefaultTDDNetworkConfig().Ran,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n3"},
		Handler:        updateRan,
		ExpectedStatus: 400,
		ExpectedError:  "No cellular network config found",
	}
	tests.RunUnitTest(t, e, tc)

	// Validation error
	ranConfig := lteModels.NewDefaultTDDNetworkConfig().Ran
	ranConfig.FddConfig = lteModels.NewDefaultFDDNetworkConfig().Ran.FddConfig
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/ran/", testURLRoot, "n2"),
		Payload:        ranConfig,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        updateRan,
		ExpectedStatus: 400,
		ExpectedError:  "only one of TDD or FDD configs can be set",
	}
	tests.RunUnitTest(t, e, tc)

	// happy case update ran config
	ranConfig = lteModels.NewDefaultFDDNetworkConfig().Ran
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/ran/", testURLRoot, "n2"),
		Payload:        ranConfig,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n2"},
		Handler:        updateRan,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)
	actualN2, err = configurator.LoadNetwork("n2", true, true)
	assert.NoError(t, err)
	expected.Configs[lte.CellularNetworkType].(*lteModels.NetworkCellularConfigs).Ran = ranConfig
	expected.Version = 3
	assert.Equal(t, expected, actualN2)

	// Validation Error (should not be able to add nonexistent networkID as fegNetworkID)
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/feg_network_id/", testURLRoot, "n1"),
		Payload:        tests.JSONMarshaler("bad-network-id"),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        updateFegNetworkID,
		ExpectedStatus: 400,
		ExpectedError:  "Network: bad-network-id does not exist",
	}
	tests.RunUnitTest(t, e, tc)

	// happy case
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/cellular/feg_network_id/", testURLRoot, "n1"),
		Payload:        tests.JSONMarshaler("n2"),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        updateFegNetworkID,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)
}

func TestCellularDelete(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)

	e := echo.New()
	testURLRoot := "/magma/v1/lte"

	seedNetworks(t)

	handlers := handlers.GetHandlers()
	deleteCellular := tests.GetHandlerByPathAndMethod(t, handlers,
		fmt.Sprintf("%s/:network_id/cellular", testURLRoot), obsidian.DELETE).HandlerFunc

	tc := tests.Test{
		Method:         "DELETE",
		URL:            fmt.Sprintf("%s/%s/cellular/", testURLRoot, "n1"),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        deleteCellular,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	_, err := configurator.LoadNetworkConfig("n1", lte.CellularNetworkType)
	assert.EqualError(t, err, "Not found")
}

func Test_GetNetworkSubscriberConfigHandlers(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)

	e := echo.New()
	testURLRoot := "/magma/v1/networks"

	seedNetworks(t)

	obsidianHandlers := handlers.GetHandlers()
	getSubscriberConfig := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config", obsidian.GET).HandlerFunc
	getRuleNames := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/rule_names", obsidian.GET).HandlerFunc
	getBaseNames := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/base_names", obsidian.GET).HandlerFunc

	// 404
	tc := tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getSubscriberConfig,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(&policyModels.NetworkSubscriberConfig{}),
	}
	tests.RunUnitTest(t, e, tc)

	subscriberConfig := &policyModels.NetworkSubscriberConfig{
		NetworkWideBaseNames: []policyModels.BaseName{"base1"},
		NetworkWideRuleNames: []string{"rule1"},
	}
	assert.NoError(t, configurator.UpdateNetworkConfig("n1", lte.NetworkSubscriberConfigType, subscriberConfig))

	// happy case
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getSubscriberConfig,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(subscriberConfig),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	// happy case
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/base_names/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getBaseNames,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(subscriberConfig.NetworkWideBaseNames),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	// happy case
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/rule_names/", testURLRoot, "n1"),
		Payload:        nil,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        getRuleNames,
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(subscriberConfig.NetworkWideRuleNames),
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)
}

func Test_ModifyNetworkSubscriberConfigHandlers(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)

	e := echo.New()
	testURLRoot := "/magma/v1/networks"

	seedNetworks(t)

	obsidianHandlers := handlers.GetHandlers()
	putSubscriberConfig := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config", obsidian.PUT).HandlerFunc
	putRuleNames := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/rule_names", obsidian.PUT).HandlerFunc
	putBaseNames := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/base_names", obsidian.PUT).HandlerFunc
	postRuleName := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/rule_names/:rule_id", obsidian.POST).HandlerFunc
	postBaseName := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/base_names/:base_name", obsidian.POST).HandlerFunc
	deleteRuleName := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/rule_names/:rule_id", obsidian.DELETE).HandlerFunc
	deleteBaseName := tests.GetHandlerByPathAndMethod(t, obsidianHandlers, "/magma/v1/lte/:network_id/subscriber_config/base_names/:base_name", obsidian.DELETE).HandlerFunc

	subscriberConfig := &policyModels.NetworkSubscriberConfig{
		NetworkWideBaseNames: []policyModels.BaseName{"base1"},
		NetworkWideRuleNames: []string{"rule1"},
	}

	// non-existent network id
	tc := tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/base_names/", testURLRoot, "n32"),
		Payload:        tests.JSONMarshaler(subscriberConfig.NetworkWideBaseNames),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n32"},
		Handler:        putBaseNames,
		ExpectedStatus: 404,
		ExpectedError:  "Not found",
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/rule_names/", testURLRoot, "n32"),
		Payload:        tests.JSONMarshaler(subscriberConfig.NetworkWideRuleNames),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n32"},
		Handler:        putRuleNames,
		ExpectedStatus: 404,
		ExpectedError:  "Not found",
	}
	tests.RunUnitTest(t, e, tc)

	// add to non existent config
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/base_names/", testURLRoot, "n1"),
		Payload:        tests.JSONMarshaler(subscriberConfig.NetworkWideBaseNames),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        putBaseNames,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/rule_names/", testURLRoot, "n1"),
		Payload:        tests.JSONMarshaler(subscriberConfig.NetworkWideRuleNames),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        putRuleNames,
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)
	iSubscriberConfig, err := configurator.GetNetworkConfigsByType("n1", lte.NetworkSubscriberConfigType)
	assert.NoError(t, err)
	assert.Equal(t, subscriberConfig, iSubscriberConfig.(*policyModels.NetworkSubscriberConfig))

	newRuleNames := []string{"rule2"}
	// happy case
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/rule_names/", testURLRoot, "n1"),
		Payload:        tests.JSONMarshaler(newRuleNames),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        putRuleNames,
		ExpectedStatus: 204,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	newBaseNames := []policyModels.BaseName{"base2"}
	// happy case
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/base_names/", testURLRoot, "n1"),
		Payload:        tests.JSONMarshaler(newBaseNames),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        putBaseNames,
		ExpectedStatus: 204,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	iSubscriberConfig, err = configurator.GetNetworkConfigsByType("n1", lte.NetworkSubscriberConfigType)
	assert.NoError(t, err)
	actualSubscriberConfig := iSubscriberConfig.(*policyModels.NetworkSubscriberConfig)

	assert.ElementsMatch(t, newRuleNames, actualSubscriberConfig.NetworkWideRuleNames)
	assert.ElementsMatch(t, newBaseNames, actualSubscriberConfig.NetworkWideBaseNames)

	newSubscriberConfig := &policyModels.NetworkSubscriberConfig{
		NetworkWideBaseNames: []policyModels.BaseName{"base3"},
		NetworkWideRuleNames: []string{"rule3"},
	}
	// happy case
	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/", testURLRoot, "n1"),
		Payload:        tests.JSONMarshaler(newSubscriberConfig),
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		Handler:        putSubscriberConfig,
		ExpectedStatus: 204,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	iSubscriberConfig, err = configurator.GetNetworkConfigsByType("n1", lte.NetworkSubscriberConfigType)
	assert.NoError(t, err)
	actualSubscriberConfig = iSubscriberConfig.(*policyModels.NetworkSubscriberConfig)

	assert.Equal(t, newSubscriberConfig, actualSubscriberConfig)

	tc = tests.Test{
		Method:         "POST",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/rule_names/%s", testURLRoot, "n1", "rule4"),
		Payload:        tests.JSONMarshaler(newSubscriberConfig),
		ParamNames:     []string{"network_id", "rule_id"},
		ParamValues:    []string{"n1", "rule4"},
		Handler:        postRuleName,
		ExpectedStatus: 201,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	// posting twice shouldn't affect anything
	tc = tests.Test{
		Method:         "POST",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/rule_names/%s", testURLRoot, "n1", "rule4"),
		Payload:        tests.JSONMarshaler(newSubscriberConfig),
		ParamNames:     []string{"network_id", "rule_id"},
		ParamValues:    []string{"n1", "rule4"},
		Handler:        postRuleName,
		ExpectedStatus: 201,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "POST",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/base_names/%s", testURLRoot, "n1", "base4"),
		Payload:        tests.JSONMarshaler(newSubscriberConfig),
		ParamNames:     []string{"network_id", "base_name"},
		ParamValues:    []string{"n1", "base4"},
		Handler:        postBaseName,
		ExpectedStatus: 201,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)
	tc = tests.Test{
		Method:         "POST",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/base_names/%s", testURLRoot, "n1", "base4"),
		Payload:        tests.JSONMarshaler(newSubscriberConfig),
		ParamNames:     []string{"network_id", "base_name"},
		ParamValues:    []string{"n1", "base4"},
		Handler:        postBaseName,
		ExpectedStatus: 201,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	newSubscriberConfig = &policyModels.NetworkSubscriberConfig{
		NetworkWideBaseNames: []policyModels.BaseName{"base3", "base4"},
		NetworkWideRuleNames: []string{"rule3", "rule4"},
	}
	iSubscriberConfig, err = configurator.GetNetworkConfigsByType("n1", lte.NetworkSubscriberConfigType)
	assert.NoError(t, err)
	actualSubscriberConfig = iSubscriberConfig.(*policyModels.NetworkSubscriberConfig)
	assert.Equal(t, newSubscriberConfig, actualSubscriberConfig)

	tc = tests.Test{
		Method:         "DELETE",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/rule_names/%s", testURLRoot, "n1", "rule4"),
		Payload:        tests.JSONMarshaler(newSubscriberConfig),
		ParamNames:     []string{"network_id", "rule_id"},
		ParamValues:    []string{"n1", "rule4"},
		Handler:        deleteRuleName,
		ExpectedStatus: 204,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "DELETE",
		URL:            fmt.Sprintf("%s/%s/subscriber_config/base_names/%s", testURLRoot, "n1", "base4"),
		Payload:        tests.JSONMarshaler(newSubscriberConfig),
		ParamNames:     []string{"network_id", "base_name"},
		ParamValues:    []string{"n1", "base4"},
		Handler:        deleteBaseName,
		ExpectedStatus: 204,
		ExpectedError:  "",
	}
	tests.RunUnitTest(t, e, tc)

	newSubscriberConfig = &policyModels.NetworkSubscriberConfig{
		NetworkWideBaseNames: []policyModels.BaseName{"base3"},
		NetworkWideRuleNames: []string{"rule3"},
	}
	iSubscriberConfig, err = configurator.GetNetworkConfigsByType("n1", lte.NetworkSubscriberConfigType)
	assert.NoError(t, err)
	actualSubscriberConfig = iSubscriberConfig.(*policyModels.NetworkSubscriberConfig)
	assert.Equal(t, newSubscriberConfig, actualSubscriberConfig)
}

func TestCreateGateway(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	test_init.StartTestService(t)
	stateTestInit.StartTestService(t)
	deviceTestInit.StartTestService(t)

	// setup fixtures in backend
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)
	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{Type: orc8r.UpgradeTierEntityType, Key: "t1"},
			{Type: lte.CellularEnodebType, Key: "enb1"},
		},
	)
	assert.NoError(t, err)
	err = device.RegisterDevice(
		"n1", orc8r.AccessGatewayRecordType, "hw2",
		&models.GatewayDevice{
			HardwareID: "hw2",
			Key:        &models.ChallengeKey{KeyType: "ECHO"},
		},
	)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/gateways"
	hands := handlers.GetHandlers()
	createGateway := tests.GetHandlerByPathAndMethod(t, hands, testURLRoot, obsidian.POST).HandlerFunc

	// happy path, no device
	payload := &lteModels.MutableLteGateway{
		Device: &models.GatewayDevice{
			HardwareID: "hw1",
			Key:        &models.ChallengeKey{KeyType: "ECHO"},
		},
		ID:          "g1",
		Name:        "foobar",
		Description: "foo bar",
		Magmad: &models.MagmadGatewayConfigs{
			CheckinInterval:         15,
			CheckinTimeout:          10,
			AutoupgradePollInterval: 300,
			AutoupgradeEnabled:      swag.Bool(true),
		},
		Cellular:               newDefaultGatewayConfig(),
		ConnectedEnodebSerials: []string{"enb1"},
		Tier:                   "t1",
	}
	tc := tests.Test{
		Method:         "POST",
		URL:            testURLRoot,
		Handler:        createGateway,
		Payload:        payload,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 201,
	}
	tests.RunUnitTest(t, e, tc)

	actualEnts, _, err := configurator.LoadEntities(
		"n1", nil, nil, nil,
		[]storage.TypeAndKey{
			{Type: orc8r.MagmadGatewayType, Key: "g1"},
			{Type: lte.CellularGatewayType, Key: "g1"},
		},
		configurator.FullEntityLoadCriteria(),
	)
	assert.NoError(t, err)
	actualDevice, err := device.GetDevice("n1", orc8r.AccessGatewayRecordType, "hw1")
	assert.NoError(t, err)

	expectedEnts := configurator.NetworkEntities{
		{
			NetworkID: "n1", Type: lte.CellularGatewayType, Key: "g1",
			Name: string(payload.Name), Description: string(payload.Description),
			Config:             payload.Cellular,
			Associations:       []storage.TypeAndKey{{Type: lte.CellularEnodebType, Key: "enb1"}},
			ParentAssociations: []storage.TypeAndKey{{Type: orc8r.MagmadGatewayType, Key: "g1"}},
			GraphID:            "2",
		},
		{
			NetworkID: "n1", Type: orc8r.MagmadGatewayType, Key: "g1",
			Name: string(payload.Name), Description: string(payload.Description),
			PhysicalID:         "hw1",
			Config:             payload.Magmad,
			Associations:       []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			ParentAssociations: []storage.TypeAndKey{{Type: orc8r.UpgradeTierEntityType, Key: "t1"}},
			GraphID:            "2",
			Version:            1,
		},
	}
	assert.Equal(t, expectedEnts, actualEnts)
	assert.Equal(t, payload.Device, actualDevice)

	// valid magmad gateway, invalid cellular - nothing should change on backend
	payload = &lteModels.MutableLteGateway{
		Device: &models.GatewayDevice{
			HardwareID: "hw2",
			Key:        &models.ChallengeKey{KeyType: "ECHO"},
		},
		ID:          "g3",
		Name:        "foobar",
		Description: "foo bar",
		Magmad: &models.MagmadGatewayConfigs{
			CheckinInterval:         15,
			CheckinTimeout:          10,
			AutoupgradePollInterval: 300,
			AutoupgradeEnabled:      swag.Bool(true),
		},
		Cellular: newDefaultGatewayConfig(),
		// Invalid due to nonexistent enb
		ConnectedEnodebSerials: []string{"enb1", "dne"},
		Tier:                   "t1",
	}
	tc = tests.Test{
		Method:         "POST",
		URL:            testURLRoot,
		Handler:        createGateway,
		Payload:        payload,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 500,
		ExpectedError:  "failed to create gateway: rpc error: code = Internal desc = could not find entities matching [type:\"cellular_enodeb\"  key:\"dne\"]",
	}
	tests.RunUnitTest(t, e, tc)

	actualEnts, _, err = configurator.LoadEntities(
		"n1", nil, nil, nil,
		[]storage.TypeAndKey{
			{Type: orc8r.MagmadGatewayType, Key: "g3"},
			{Type: lte.CellularGatewayType, Key: "g3"},
		},
		configurator.FullEntityLoadCriteria(),
	)
	assert.NoError(t, err)
	// the device should get created regardless
	actualDevice, err = device.GetDevice("n1", orc8r.AccessGatewayRecordType, "hw2")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(actualEnts))
	assert.Equal(t, payload.Device, actualDevice)

	// Some composite validation failures - bad device key, missing required
	// non-EPS control fields when non-EPS service control is on
	pubkeyB64 := strfmt.Base64("fake key")
	payload = &lteModels.MutableLteGateway{
		Device: &models.GatewayDevice{
			HardwareID: "foo-bar-baz-890",
			Key: &models.ChallengeKey{
				KeyType: "SOFTWARE_ECDSA_SHA256",
				Key:     &pubkeyB64,
			},
		},
		ID:          "g4",
		Name:        "foobar",
		Description: "foo bar",
		Magmad: &models.MagmadGatewayConfigs{
			CheckinInterval:         15,
			CheckinTimeout:          10,
			AutoupgradePollInterval: 300,
			AutoupgradeEnabled:      swag.Bool(true),
		},
		Cellular:               newDefaultGatewayConfig(),
		ConnectedEnodebSerials: []string{},
		Tier:                   "t1",
	}
	payload.Cellular.NonEpsService = &lteModels.GatewayNonEpsConfigs{
		NonEpsServiceControl: swag.Uint32(1),
	}

	tc = tests.Test{
		Method:         "POST",
		URL:            testURLRoot,
		Handler:        createGateway,
		Payload:        payload,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 400,
		ExpectedError: "validation failure list:\n" +
			"validation failure list:\n" +
			"arfcn_2g in body is required\n" +
			"csfb_mcc in body is required\n" +
			"csfb_mnc in body is required\n" +
			"csfb_rat in body is required\n" +
			"lac in body is required\n" +
			"Failed to parse key: asn1: structure error: tags don't match (16 vs {class:1 tag:6 length:97 isCompound:true}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} publicKeyInfo @2",
	}
	tests.RunUnitTest(t, e, tc)
}

func TestListAndGetGateways(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	clock.SetAndFreezeClock(t, time.Unix(1000000, 0))
	defer clock.UnfreezeClock(t)

	test_init.StartTestService(t)
	stateTestInit.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/gateways"

	handlers := handlers.GetHandlers()
	listGateways := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.GET).HandlerFunc
	getGateway := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/:gateway_id", testURLRoot), obsidian.GET).HandlerFunc

	// Create 2 gateways, 1 with state and device, the other without
	// g2 will associate to 2 enodebs
	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{Type: lte.CellularEnodebType, Key: "enb1"},
			{Type: lte.CellularEnodebType, Key: "enb2"},
			{
				Type: lte.CellularGatewayType, Key: "g1",
				Config: &lteModels.GatewayCellularConfigs{
					Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
					Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
				},
			},
			{
				Type: lte.CellularGatewayType, Key: "g2",
				Config: &lteModels.GatewayCellularConfigs{
					Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
					Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
				},
				Associations: []storage.TypeAndKey{
					{Type: lte.CellularEnodebType, Key: "enb1"},
					{Type: lte.CellularEnodebType, Key: "enb2"},
				},
			},
			{
				Type: orc8r.MagmadGatewayType, Key: "g1",
				Name: "foobar", Description: "foo bar",
				PhysicalID: "hw1",
				Config: &models.MagmadGatewayConfigs{
					AutoupgradeEnabled:      swag.Bool(true),
					AutoupgradePollInterval: 300,
					CheckinInterval:         15,
					CheckinTimeout:          5,
				},
				Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			},
			{
				Type: orc8r.MagmadGatewayType, Key: "g2",
				Name: "barfoo", Description: "bar foo",
				PhysicalID: "hw2",
				Config: &models.MagmadGatewayConfigs{
					AutoupgradeEnabled:      swag.Bool(true),
					AutoupgradePollInterval: 300,
					CheckinInterval:         15,
					CheckinTimeout:          5,
				},
				Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g2"}},
			},
			{
				Type: orc8r.UpgradeTierEntityType, Key: "t1",
				Associations: []storage.TypeAndKey{
					{Type: orc8r.MagmadGatewayType, Key: "g1"},
					{Type: orc8r.MagmadGatewayType, Key: "g2"},
				},
			},
		},
	)
	assert.NoError(t, err)
	err = device.RegisterDevice("n1", orc8r.AccessGatewayRecordType, "hw1", &models.GatewayDevice{HardwareID: "hw1", Key: &models.ChallengeKey{KeyType: "ECHO"}})
	assert.NoError(t, err)
	ctx := test_utils.GetContextWithCertificate(t, "hw1")
	test_utils.ReportGatewayStatus(t, ctx, models.NewDefaultGatewayStatus("hw1"))

	expected := map[string]*lteModels.LteGateway{
		"g1": {
			ID: "g1",
			Device: &models.GatewayDevice{
				HardwareID: "hw1",
				Key:        &models.ChallengeKey{KeyType: "ECHO"},
			},
			Name: "foobar", Description: "foo bar",
			Tier: "t1",
			Magmad: &models.MagmadGatewayConfigs{
				AutoupgradeEnabled:      swag.Bool(true),
				AutoupgradePollInterval: 300,
				CheckinInterval:         15,
				CheckinTimeout:          5,
			},
			Cellular: &lteModels.GatewayCellularConfigs{
				Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
				Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
			},
			Status: models.NewDefaultGatewayStatus("hw1"),
		},
		"g2": {
			ID:   "g2",
			Name: "barfoo", Description: "bar foo",
			Tier: "t1",
			Magmad: &models.MagmadGatewayConfigs{
				AutoupgradeEnabled:      swag.Bool(true),
				AutoupgradePollInterval: 300,
				CheckinInterval:         15,
				CheckinTimeout:          5,
			},
			Cellular: &lteModels.GatewayCellularConfigs{
				Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
				Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
			},
			ConnectedEnodebSerials: []string{"enb1", "enb2"},
		},
	}
	expected["g1"].Status.CheckinTime = uint64(time.Unix(1000000, 0).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)))
	expected["g1"].Status.CertExpirationTime = time.Unix(1000000, 0).Add(time.Hour * 4).Unix()

	tc := tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        listGateways,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(expected),
	}
	tests.RunUnitTest(t, e, tc)

	expectedGet := &lteModels.LteGateway{
		ID: "g1",
		Device: &models.GatewayDevice{
			HardwareID: "hw1",
			Key:        &models.ChallengeKey{KeyType: "ECHO"},
		},
		Name: "foobar", Description: "foo bar",
		Tier: "t1",
		Magmad: &models.MagmadGatewayConfigs{
			AutoupgradeEnabled:      swag.Bool(true),
			AutoupgradePollInterval: 300,
			CheckinInterval:         15,
			CheckinTimeout:          5,
		},
		Cellular: &lteModels.GatewayCellularConfigs{
			Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
			Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
		},
		Status: models.NewDefaultGatewayStatus("hw1"),
	}
	expectedGet.Status.CheckinTime = uint64(time.Unix(1000000, 0).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)))
	expectedGet.Status.CertExpirationTime = time.Unix(1000000, 0).Add(time.Hour * 4).Unix()
	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getGateway,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 200,
		ExpectedResult: expectedGet,
	}
	tests.RunUnitTest(t, e, tc)

	expectedGet = &lteModels.LteGateway{
		ID:   "g2",
		Name: "barfoo", Description: "bar foo",
		Tier: "t1",
		Magmad: &models.MagmadGatewayConfigs{
			AutoupgradeEnabled:      swag.Bool(true),
			AutoupgradePollInterval: 300,
			CheckinInterval:         15,
			CheckinTimeout:          5,
		},
		Cellular: &lteModels.GatewayCellularConfigs{
			Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
			Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
		},
		ConnectedEnodebSerials: []string{"enb1", "enb2"},
	}
	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getGateway,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g2"},
		ExpectedStatus: 200,
		ExpectedResult: expectedGet,
	}
	tests.RunUnitTest(t, e, tc)
}

func TestUpdateGateway(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	clock.SetAndFreezeClock(t, time.Unix(1000000, 0))
	defer clock.UnfreezeClock(t)

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/gateways/:gateway_id"
	handlers := handlers.GetHandlers()
	updateGateway := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.PUT).HandlerFunc

	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{Type: lte.CellularEnodebType, Key: "enb1"},
			{Type: lte.CellularEnodebType, Key: "enb2"},
			{Type: lte.CellularEnodebType, Key: "enb3"},
			{
				Type: lte.CellularGatewayType, Key: "g1",
				Config: &lteModels.GatewayCellularConfigs{
					Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
					Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
				},
				Associations: []storage.TypeAndKey{
					{Type: lte.CellularEnodebType, Key: "enb1"},
					{Type: lte.CellularEnodebType, Key: "enb2"},
				},
			},
			{
				Type: orc8r.MagmadGatewayType, Key: "g1",
				Name: "foobar", Description: "foo bar",
				PhysicalID: "hw1",
				Config: &models.MagmadGatewayConfigs{
					AutoupgradeEnabled:      swag.Bool(true),
					AutoupgradePollInterval: 300,
					CheckinInterval:         15,
					CheckinTimeout:          5,
				},
				Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			},
			{
				Type: orc8r.UpgradeTierEntityType, Key: "t1",
				Associations: []storage.TypeAndKey{
					{Type: orc8r.MagmadGatewayType, Key: "g1"},
				},
			},
		},
	)
	assert.NoError(t, err)
	err = device.RegisterDevice("n1", orc8r.AccessGatewayRecordType, "hw1", &models.GatewayDevice{HardwareID: "hw1", Key: &models.ChallengeKey{KeyType: "ECHO"}})
	assert.NoError(t, err)

	// update everything
	privateKey, err := key.GenerateKey("P256", 0)
	assert.NoError(t, err)
	marshaledPubKey, err := x509.MarshalPKIXPublicKey(key.PublicKey(privateKey))
	assert.NoError(t, err)
	pubkeyB64 := strfmt.Base64(marshaledPubKey)
	payload := &lteModels.MutableLteGateway{
		Device: &models.GatewayDevice{
			HardwareID: "hw1",
			Key:        &models.ChallengeKey{KeyType: "SOFTWARE_ECDSA_SHA256", Key: &pubkeyB64},
		},
		ID:          "g1",
		Name:        "barbaz",
		Description: "bar baz",
		Magmad: &models.MagmadGatewayConfigs{
			CheckinInterval:         25,
			CheckinTimeout:          15,
			AutoupgradePollInterval: 200,
			AutoupgradeEnabled:      swag.Bool(false),
			FeatureFlags:            map[string]bool{"foo": false},
			DynamicServices:         []string{"d1", "d2"},
		},
		Tier: "t1",
		Cellular: &lteModels.GatewayCellularConfigs{
			Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(false), IPBlock: "172.10.10.0/24"},
			Ran: &lteModels.GatewayRanConfigs{Pci: 123, TransmitEnabled: swag.Bool(false)},
		},
		ConnectedEnodebSerials: []string{"enb1", "enb3"},
	}

	tc := tests.Test{
		Method:         "PUT",
		URL:            testURLRoot,
		Handler:        updateGateway,
		Payload:        payload,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actualEnts, _, err := configurator.LoadEntities(
		"n1", nil, nil, nil,
		[]storage.TypeAndKey{
			{Type: orc8r.MagmadGatewayType, Key: "g1"},
			{Type: lte.CellularGatewayType, Key: "g1"},
			{Type: orc8r.UpgradeTierEntityType, Key: "t1"},
		},
		configurator.FullEntityLoadCriteria(),
	)
	assert.NoError(t, err)
	actualDevice, err := device.GetDevice("n1", orc8r.AccessGatewayRecordType, "hw1")
	assert.NoError(t, err)

	expectedEnts := configurator.NetworkEntities{
		{
			NetworkID: "n1", Type: lte.CellularGatewayType, Key: "g1",
			Name: string(payload.Name), Description: string(payload.Description),
			Config:             payload.Cellular,
			ParentAssociations: []storage.TypeAndKey{{Type: orc8r.MagmadGatewayType, Key: "g1"}},
			Associations: []storage.TypeAndKey{
				{Type: lte.CellularEnodebType, Key: "enb1"},
				{Type: lte.CellularEnodebType, Key: "enb3"},
			},
			GraphID: "10",
			Version: 1,
		},
		{
			NetworkID: "n1", Type: orc8r.MagmadGatewayType, Key: "g1",
			Name: string(payload.Name), Description: string(payload.Description),
			PhysicalID:         "hw1",
			Config:             payload.Magmad,
			Associations:       []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			ParentAssociations: []storage.TypeAndKey{{Type: orc8r.UpgradeTierEntityType, Key: "t1"}},
			GraphID:            "10",
			Version:            1,
		},
		{
			NetworkID: "n1", Type: orc8r.UpgradeTierEntityType, Key: "t1",
			Associations: []storage.TypeAndKey{{Type: orc8r.MagmadGatewayType, Key: "g1"}},
			GraphID:      "10",
		},
	}
	assert.Equal(t, expectedEnts, actualEnts)
	assert.Equal(t, payload.Device, actualDevice)
}

func TestDeleteGateway(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})
	clock.SetAndFreezeClock(t, time.Unix(1000000, 0))
	defer clock.UnfreezeClock(t)

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/gateways/:gateway_id"
	handlers := handlers.GetHandlers()
	deleteGateway := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.DELETE).HandlerFunc

	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{Type: lte.CellularEnodebType, Key: "enb1"},
			{Type: lte.CellularEnodebType, Key: "enb2"},
			{
				Type: lte.CellularGatewayType, Key: "g1",
				Config: &lteModels.GatewayCellularConfigs{
					Epc: &lteModels.GatewayEpcConfigs{NatEnabled: swag.Bool(true), IPBlock: "192.168.0.0/24"},
					Ran: &lteModels.GatewayRanConfigs{Pci: 260, TransmitEnabled: swag.Bool(true)},
				},
				Associations: []storage.TypeAndKey{
					{Type: lte.CellularEnodebType, Key: "enb1"},
					{Type: lte.CellularEnodebType, Key: "enb2"},
				},
			},
			{
				Type: orc8r.MagmadGatewayType, Key: "g1",
				Name: "foobar", Description: "foo bar",
				PhysicalID: "hw1",
				Config: &models.MagmadGatewayConfigs{
					AutoupgradeEnabled:      swag.Bool(true),
					AutoupgradePollInterval: 300,
					CheckinInterval:         15,
					CheckinTimeout:          5,
				},
				Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			},
			{
				Type: orc8r.UpgradeTierEntityType, Key: "t1",
				Associations: []storage.TypeAndKey{
					{Type: orc8r.MagmadGatewayType, Key: "g1"},
				},
			},
		},
	)
	assert.NoError(t, err)
	err = device.RegisterDevice("n1", orc8r.AccessGatewayRecordType, "hw1", &models.GatewayDevice{HardwareID: "hw1", Key: &models.ChallengeKey{KeyType: "ECHO"}})
	assert.NoError(t, err)

	tc := tests.Test{
		Method:         "DELETE",
		URL:            testURLRoot,
		Handler:        deleteGateway,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actualEnts, _, err := configurator.LoadEntities(
		"n1", nil, nil, nil,
		[]storage.TypeAndKey{
			{Type: orc8r.MagmadGatewayType, Key: "g1"},
			{Type: lte.CellularGatewayType, Key: "g1"},
			{Type: orc8r.UpgradeTierEntityType, Key: "t1"},
		},
		configurator.FullEntityLoadCriteria(),
	)
	assert.NoError(t, err)
	actualDevice, err := device.GetDevice("n1", orc8r.AccessGatewayRecordType, "hw1")
	assert.Nil(t, actualDevice)
	assert.EqualError(t, err, "Not found")

	expectedEnts := configurator.NetworkEntities{
		{NetworkID: "n1", Type: orc8r.UpgradeTierEntityType, Key: "t1", GraphID: "11"},
	}
	assert.Equal(t, expectedEnts, actualEnts)
}

func TestGetCellularGatewayConfig(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/gateways/:gateway_id"
	handlers := handlers.GetHandlers()
	getCellular := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular", testURLRoot), obsidian.GET).HandlerFunc
	getEpc := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular/epc", testURLRoot), obsidian.GET).HandlerFunc
	getRan := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular/ran", testURLRoot), obsidian.GET).HandlerFunc
	getNonEps := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular/non_eps", testURLRoot), obsidian.GET).HandlerFunc
	getEnodebs := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/connected_enodeb_serials", testURLRoot), obsidian.GET).HandlerFunc

	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{Type: lte.CellularEnodebType, Key: "enb1"},
			{Type: lte.CellularEnodebType, Key: "enb2"},
			{
				Type: lte.CellularGatewayType, Key: "g1",
				Config: newDefaultGatewayConfig(),
				Associations: []storage.TypeAndKey{
					{Type: lte.CellularEnodebType, Key: "enb1"},
					{Type: lte.CellularEnodebType, Key: "enb2"},
				},
			},
			{
				Type: orc8r.MagmadGatewayType, Key: "g1",
				Name: "foobar", Description: "foo bar",
				PhysicalID:   "hw1",
				Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			},
		},
	)
	assert.NoError(t, err)

	// 404
	tc := tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/cellular", testURLRoot),
		Handler:        getCellular,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g2"},
		ExpectedResult: newDefaultGatewayConfig(),
		ExpectedStatus: 404,
		ExpectedError:  "Not found",
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/cellular", testURLRoot),
		Handler:        getCellular,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedResult: newDefaultGatewayConfig(),
		ExpectedStatus: 200,
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/cellular/epc", testURLRoot),
		Handler:        getEpc,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedResult: newDefaultGatewayConfig().Epc,
		ExpectedStatus: 200,
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/cellular/ran", testURLRoot),
		Handler:        getRan,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedResult: newDefaultGatewayConfig().Ran,
		ExpectedStatus: 200,
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/cellular/non_eps", testURLRoot),
		Handler:        getNonEps,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedResult: newDefaultGatewayConfig().NonEpsService,
		ExpectedStatus: 200,
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            fmt.Sprintf("%s/cellular/connected_enodeb_serial", testURLRoot),
		Handler:        getEnodebs,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedResult: tests.JSONMarshaler([]string{"enb1", "enb2"}),
		ExpectedStatus: 200,
	}
	tests.RunUnitTest(t, e, tc)
}

func TestUpdateCellularGatewayConfig(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/gateways/:gateway_id"
	handlers := handlers.GetHandlers()
	updateCellular := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular", testURLRoot), obsidian.PUT).HandlerFunc
	updateEpc := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular/epc", testURLRoot), obsidian.PUT).HandlerFunc
	updateRan := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular/ran", testURLRoot), obsidian.PUT).HandlerFunc
	updateNonEps := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/cellular/non_eps", testURLRoot), obsidian.PUT).HandlerFunc
	updateEnodebs := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/connected_enodeb_serials", testURLRoot), obsidian.PUT).HandlerFunc
	postEnodeb := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/connected_enodeb_serials", testURLRoot), obsidian.POST).HandlerFunc
	deleteEnodeb := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/connected_enodeb_serials", testURLRoot), obsidian.DELETE).HandlerFunc

	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{Type: lte.CellularEnodebType, Key: "enb1"},
			{Type: lte.CellularEnodebType, Key: "enb2"},
			{Type: lte.CellularGatewayType, Key: "g1"},
			{
				Type: orc8r.MagmadGatewayType, Key: "g1",
				Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			},
		},
	)
	assert.NoError(t, err)

	tc := tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/cellular", testURLRoot),
		Handler:        updateCellular,
		Payload:        newDefaultGatewayConfig(),
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected := map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "6",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  newDefaultGatewayConfig(),
			GraphID: "6",
			Version: 1,
		},
	}

	entities, _, err := configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())

	modifiedCellularConfig := newDefaultGatewayConfig()
	modifiedCellularConfig.Epc.NatEnabled = swag.Bool(false)
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/cellular/epc", testURLRoot),
		Handler:        updateEpc,
		Payload:        modifiedCellularConfig.Epc,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected = map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "6",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  modifiedCellularConfig,
			GraphID: "6",
			Version: 2,
		},
	}
	entities, _, err = configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())

	modifiedCellularConfig.Ran.TransmitEnabled = swag.Bool(false)
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/cellular/ran", testURLRoot),
		Handler:        updateRan,
		Payload:        modifiedCellularConfig.Ran,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected = map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "6",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  modifiedCellularConfig,
			GraphID: "6",
			Version: 3,
		},
	}
	entities, _, err = configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())

	// validation failure
	modifiedCellularConfig.NonEpsService.NonEpsServiceControl = swag.Uint32(1)
	modifiedCellularConfig.NonEpsService.CsfbMcc = "0"
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/cellular/ran", testURLRoot),
		Handler:        updateNonEps,
		Payload:        modifiedCellularConfig.NonEpsService,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 400,
		ExpectedError:  "validation failure list:\ncsfb_mcc in body should match '^(\\d{3})$'",
	}
	tests.RunUnitTest(t, e, tc)

	// happy case
	modifiedCellularConfig.NonEpsService.CsfbMcc = "123"
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/cellular/ran", testURLRoot),
		Handler:        updateNonEps,
		Payload:        modifiedCellularConfig.NonEpsService,
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected = map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "6",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  modifiedCellularConfig,
			GraphID: "6",
			Version: 4,
		},
	}
	entities, _, err = configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())

	// connected enodeBs - happy case
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/connected_enodeb_serial", testURLRoot),
		Handler:        updateEnodebs,
		Payload:        tests.JSONMarshaler([]string{"enb1", "enb2"}),
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected = map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "2",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  modifiedCellularConfig,
			GraphID: "2",
			Version: 5,
			Associations: []storage.TypeAndKey{
				{Type: lte.CellularEnodebType, Key: "enb1"},
				{Type: lte.CellularEnodebType, Key: "enb2"},
			},
		},
	}
	entities, _, err = configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())

	_, err = configurator.CreateEntity("n1", configurator.NetworkEntity{Type: lte.CellularEnodebType, Key: "enb3"})
	assert.NoError(t, err)

	// happy case
	tc = tests.Test{
		Method:         "POST",
		URL:            fmt.Sprintf("%s/connected_enodeb_serial", testURLRoot),
		Handler:        postEnodeb,
		Payload:        tests.JSONMarshaler("enb3"),
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected = map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "10",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  modifiedCellularConfig,
			GraphID: "10",
			Version: 6,
			Associations: []storage.TypeAndKey{
				{Type: lte.CellularEnodebType, Key: "enb1"},
				{Type: lte.CellularEnodebType, Key: "enb2"},
				{Type: lte.CellularEnodebType, Key: "enb3"},
			},
		},
	}
	entities, _, err = configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())

	// happy case
	tc = tests.Test{
		Method:         "DELETE",
		URL:            fmt.Sprintf("%s/connected_enodeb_serial", testURLRoot),
		Handler:        deleteEnodeb,
		Payload:        tests.JSONMarshaler("enb3"),
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected = map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "10",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  modifiedCellularConfig,
			GraphID: "10",
			Version: 7,
			Associations: []storage.TypeAndKey{
				{Type: lte.CellularEnodebType, Key: "enb1"},
				{Type: lte.CellularEnodebType, Key: "enb2"},
			},
		},
	}
	entities, _, err = configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())

	// Clear enb serial list
	tc = tests.Test{
		Method:         "PUT",
		URL:            fmt.Sprintf("%s/connected_enodeb_serial", testURLRoot),
		Handler:        updateEnodebs,
		Payload:        tests.JSONMarshaler([]string{}),
		ParamNames:     []string{"network_id", "gateway_id"},
		ParamValues:    []string{"n1", "g1"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	expected = map[storage.TypeAndKey]configurator.NetworkEntity{
		storage.TypeAndKey{Type: orc8r.MagmadGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      orc8r.MagmadGatewayType, Key: "g1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularGatewayType, Key: "g1"}},
			GraphID:      "10",
			Version:      0,
		},
		storage.TypeAndKey{Type: lte.CellularGatewayType, Key: "g1"}: {
			NetworkID: "n1",
			Type:      lte.CellularGatewayType, Key: "g1",
			Config:  modifiedCellularConfig,
			GraphID: "10",
			Version: 8,
		},
	}
	entities, _, err = configurator.LoadEntities("n1", nil, swag.String("g1"), nil, nil, configurator.EntityLoadCriteria{LoadConfig: true, LoadAssocsFromThis: true})
	assert.NoError(t, err)
	assert.Equal(t, expected, entities.ToEntitiesByID())
}

func TestListAndGetEnodebs(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/enodebs"

	handlers := handlers.GetHandlers()
	listEnodebs := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.GET).HandlerFunc
	getEnodeb := tests.GetHandlerByPathAndMethod(t, handlers, fmt.Sprintf("%s/:enodeb_serial", testURLRoot), obsidian.GET).HandlerFunc

	_, err = configurator.CreateEntities("n1", []configurator.NetworkEntity{
		{
			Type:        lte.CellularEnodebType,
			Key:         "abcdefg",
			Name:        "abc enodeb",
			Description: "abc enodeb description",
			PhysicalID:  "abcdefg",
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           20,
				CellID:                 swag.Uint32(1234),
				DeviceClass:            "Baicells Nova-233 G2 OD FDD",
				Earfcndl:               39450,
				Pci:                    260,
				SpecialSubframePattern: 7,
				SubframeAssignment:     2,
				Tac:                    1,
				TransmitEnabled:        swag.Bool(true),
			},
		},
		{
			Type:        lte.CellularEnodebType,
			Key:         "vwxyz",
			Name:        "xyz enodeb",
			Description: "xyz enodeb description",
			PhysicalID:  "vwxyz",
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           15,
				CellID:                 swag.Uint32(4321),
				DeviceClass:            "Baicells Nova-243 OD TDD",
				Earfcndl:               39550,
				Pci:                    261,
				SpecialSubframePattern: 8,
				SubframeAssignment:     3,
				Tac:                    2,
				TransmitEnabled:        swag.Bool(false),
			},
		},
		{
			Type: lte.CellularGatewayType, Key: "gw1",
			Associations: []storage.TypeAndKey{{Type: lte.CellularEnodebType, Key: "abcdefg"}},
		},
	})
	assert.NoError(t, err)

	expected := map[string]*lteModels.Enodeb{
		"abcdefg": {
			AttachedGatewayID: "gw1",
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           20,
				CellID:                 swag.Uint32(1234),
				DeviceClass:            "Baicells Nova-233 G2 OD FDD",
				Earfcndl:               39450,
				Pci:                    260,
				SpecialSubframePattern: 7,
				SubframeAssignment:     2,
				Tac:                    1,
				TransmitEnabled:        swag.Bool(true),
			},
			Name:        "abc enodeb",
			Description: "abc enodeb description",
			Serial:      "abcdefg",
		},
		"vwxyz": {
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           15,
				CellID:                 swag.Uint32(4321),
				DeviceClass:            "Baicells Nova-243 OD TDD",
				Earfcndl:               39550,
				Pci:                    261,
				SpecialSubframePattern: 8,
				SubframeAssignment:     3,
				Tac:                    2,
				TransmitEnabled:        swag.Bool(false),
			},
			Name:        "xyz enodeb",
			Description: "xyz enodeb description",
			Serial:      "vwxyz",
		},
	}
	tc := tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        listEnodebs,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(expected),
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getEnodeb,
		ParamNames:     []string{"network_id", "enodeb_serial"},
		ParamValues:    []string{"n1", "abcdefg"},
		ExpectedStatus: 200,
		ExpectedResult: expected["abcdefg"],
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getEnodeb,
		ParamNames:     []string{"network_id", "enodeb_serial"},
		ParamValues:    []string{"n1", "vwxyz"},
		ExpectedStatus: 200,
		ExpectedResult: expected["vwxyz"],
	}
	tests.RunUnitTest(t, e, tc)

	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getEnodeb,
		ParamNames:     []string{"network_id", "enodeb_serial"},
		ParamValues:    []string{"n1", "hello"},
		ExpectedStatus: 404,
		ExpectedError:  "Not Found",
	}
	tests.RunUnitTest(t, e, tc)
}

func TestCreateEnodeb(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/enodebs"

	handlers := handlers.GetHandlers()
	createEnodeb := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.POST).HandlerFunc

	tc := tests.Test{
		Method:  "POST",
		URL:     testURLRoot,
		Handler: createEnodeb,
		Payload: &lteModels.Enodeb{
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           15,
				CellID:                 swag.Uint32(4321),
				DeviceClass:            "Baicells Nova-243 OD TDD",
				Earfcndl:               39550,
				Pci:                    261,
				SpecialSubframePattern: 8,
				SubframeAssignment:     3,
				Tac:                    2,
				TransmitEnabled:        swag.Bool(false),
			},
			Name:        "foobar",
			Description: "foobar description",
			Serial:      "abcdef",
		},
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 201,
	}
	tests.RunUnitTest(t, e, tc)

	actual, err := configurator.LoadEntity("n1", lte.CellularEnodebType, "abcdef", configurator.FullEntityLoadCriteria())
	assert.NoError(t, err)
	expected := configurator.NetworkEntity{
		NetworkID: "n1",
		Type:      lte.CellularEnodebType, Key: "abcdef",
		Name:        "foobar",
		Description: "foobar description",
		PhysicalID:  "abcdef",
		GraphID:     "2",
		Config: &lteModels.EnodebConfiguration{
			BandwidthMhz:           15,
			CellID:                 swag.Uint32(4321),
			DeviceClass:            "Baicells Nova-243 OD TDD",
			Earfcndl:               39550,
			Pci:                    261,
			SpecialSubframePattern: 8,
			SubframeAssignment:     3,
			Tac:                    2,
			TransmitEnabled:        swag.Bool(false),
		},
	}
	assert.Equal(t, expected, actual)

	tc = tests.Test{
		Method:  "POST",
		URL:     testURLRoot,
		Handler: createEnodeb,
		Payload: &lteModels.Enodeb{
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           15,
				CellID:                 swag.Uint32(4321),
				DeviceClass:            "Baicells Nova-243 OD TDD",
				Earfcndl:               39550,
				Pci:                    261,
				SpecialSubframePattern: 8,
				SubframeAssignment:     3,
				Tac:                    2,
				TransmitEnabled:        swag.Bool(false),
			},
			Name:              "foobar",
			Serial:            "abcdef",
			AttachedGatewayID: "gw1",
		},
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 400,
		ExpectedError:  "attached_gateway_id is a read-only property",
	}
	tests.RunUnitTest(t, e, tc)
}

func TestUpdateEnodeb(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/enodebs/:enodeb_serial"

	handlers := handlers.GetHandlers()
	updateEnodeb := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.PUT).HandlerFunc

	_, err = configurator.CreateEntities("n1", []configurator.NetworkEntity{
		{
			Type:        lte.CellularEnodebType,
			Key:         "abcdefg",
			Name:        "abc enodeb",
			Description: "abc enodeb description",
			PhysicalID:  "abcdefg",
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           20,
				CellID:                 swag.Uint32(1234),
				DeviceClass:            "Baicells Nova-233 G2 OD FDD",
				Earfcndl:               39450,
				Pci:                    260,
				SpecialSubframePattern: 7,
				SubframeAssignment:     2,
				Tac:                    1,
				TransmitEnabled:        swag.Bool(true),
			},
		},
	})
	assert.NoError(t, err)

	tc := tests.Test{
		Method:  "PUT",
		URL:     testURLRoot,
		Handler: updateEnodeb,
		Payload: &lteModels.Enodeb{
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           15,
				CellID:                 swag.Uint32(4321),
				DeviceClass:            "Baicells Nova-243 OD TDD",
				Earfcndl:               39550,
				Pci:                    261,
				SpecialSubframePattern: 8,
				SubframeAssignment:     3,
				Tac:                    2,
				TransmitEnabled:        swag.Bool(false),
			},
			Name:        "foobar",
			Description: "new description",
			Serial:      "abcdefg",
		},
		ParamNames:     []string{"network_id", "enodeb_serial"},
		ParamValues:    []string{"n1", "abcdefg"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actual, err := configurator.LoadEntity("n1", lte.CellularEnodebType, "abcdefg", configurator.FullEntityLoadCriteria())
	assert.NoError(t, err)
	expected := configurator.NetworkEntity{
		NetworkID: "n1",
		Type:      lte.CellularEnodebType, Key: "abcdefg",
		Name:        "foobar",
		Description: "new description",
		PhysicalID:  "abcdefg",
		GraphID:     "2",
		Config: &lteModels.EnodebConfiguration{
			BandwidthMhz:           15,
			CellID:                 swag.Uint32(4321),
			DeviceClass:            "Baicells Nova-243 OD TDD",
			Earfcndl:               39550,
			Pci:                    261,
			SpecialSubframePattern: 8,
			SubframeAssignment:     3,
			Tac:                    2,
			TransmitEnabled:        swag.Bool(false),
		},
		Version: 1,
	}
	assert.Equal(t, expected, actual)

	tc = tests.Test{
		Method:  "PUT",
		URL:     testURLRoot,
		Handler: updateEnodeb,
		Payload: &lteModels.Enodeb{
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           15,
				CellID:                 swag.Uint32(4321),
				DeviceClass:            "Baicells Nova-243 OD TDD",
				Earfcndl:               39550,
				Pci:                    261,
				SpecialSubframePattern: 8,
				SubframeAssignment:     3,
				Tac:                    2,
				TransmitEnabled:        swag.Bool(false),
			},
			Name:              "foobar",
			Serial:            "abcdef",
			AttachedGatewayID: "gw1",
		},
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 400,
		ExpectedError:  "attached_gateway_id is a read-only property",
	}
}

func TestDeleteEnodeb(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/enodebs/:enodeb_serial"

	handlers := handlers.GetHandlers()
	deleteEnodeb := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.DELETE).HandlerFunc

	_, err = configurator.CreateEntities("n1", []configurator.NetworkEntity{
		{
			Type:       lte.CellularEnodebType,
			Key:        "abcdefg",
			Name:       "abc enodeb",
			PhysicalID: "abcdefg",
			Config: &lteModels.EnodebConfiguration{
				BandwidthMhz:           20,
				CellID:                 swag.Uint32(1234),
				DeviceClass:            "Baicells Nova-233 G2 OD FDD",
				Earfcndl:               39450,
				Pci:                    260,
				SpecialSubframePattern: 7,
				SubframeAssignment:     2,
				Tac:                    1,
				TransmitEnabled:        swag.Bool(true),
			},
		},
	})
	assert.NoError(t, err)

	tc := tests.Test{
		Method:         "DELETE",
		URL:            testURLRoot,
		Handler:        deleteEnodeb,
		ParamNames:     []string{"network_id", "enodeb_serial"},
		ParamValues:    []string{"n1", "abcdefg"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	_, err = configurator.LoadEntity("n1", lte.CellularEnodebType, "abcdefg", configurator.FullEntityLoadCriteria())
	assert.EqualError(t, err, "Not found")
}

func TestGetEnodebState(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	deviceTestInit.StartTestService(t)
	stateTestInit.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/enodebs/:enodeb_serial/state"

	handlers := handlers.GetHandlers()
	getEnodebState := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.GET).HandlerFunc

	_, err = configurator.CreateEntities("n1",
		[]configurator.NetworkEntity{
			{
				Type: lte.CellularEnodebType, Key: "serial1",
				PhysicalID: "serial1",
			},
			{
				Type: orc8r.MagmadGatewayType, Key: "gw1",
				PhysicalID:   "hwid1",
				Associations: []storage.TypeAndKey{{Type: lte.CellularEnodebType, Key: "serial1"}},
			},
		})
	assert.NoError(t, err)

	// 404
	tc := tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getEnodebState,
		ParamNames:     []string{"network_id", "enodeb_serial"},
		ParamValues:    []string{"n1", "serial1"},
		ExpectedStatus: 404,
		ExpectedError:  "Not found",
	}
	tests.RunUnitTest(t, e, tc)

	// report state
	clock.SetAndFreezeClock(t, time.Unix(1000000, 0))
	defer clock.UnfreezeClock(t)

	// encode the appropriate certificate into context
	ctx := test_utils.GetContextWithCertificate(t, "hwid1")
	reportEnodebState(t, ctx, "serial1", lteModels.NewDefaultEnodebStatus())
	expected := lteModels.NewDefaultEnodebStatus()
	expected.TimeReported = uint64(time.Unix(1000000, 0).UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)))
	expected.ReportingGatewayID = "gw1"

	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getEnodebState,
		ParamNames:     []string{"network_id", "enodeb_serial"},
		ParamValues:    []string{"n1", "serial1"},
		ExpectedStatus: 200,
		ExpectedResult: expected,
	}
	tests.RunUnitTest(t, e, tc)
}

func TestCreateApn(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/apns"
	handlers := handlers.GetHandlers()
	createApn := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.POST).HandlerFunc

	// default apn profile should always succeed
	payload := &lteModels.Apn{
		ApnName: "foo",
		ApnConfiguration: &lteModels.ApnConfiguration{
			Ambr: &lteModels.AggregatedMaximumBitrate{
				MaxBandwidthDl: swag.Uint32(100),
				MaxBandwidthUl: swag.Uint32(100),
			},
			QosProfile: &lteModels.QosProfile{
				ClassID:                 swag.Int32(9),
				PreemptionCapability:    swag.Bool(true),
				PreemptionVulnerability: swag.Bool(false),
				PriorityLevel:           swag.Uint32(15),
			},
		},
	}
	tc := tests.Test{
		Method:         "POST",
		URL:            testURLRoot,
		Payload:        payload,
		Handler:        createApn,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 201,
	}
	tests.RunUnitTest(t, e, tc)

	actual, err := configurator.LoadEntity("n1", lte.ApnEntityType, "foo", configurator.FullEntityLoadCriteria())
	assert.NoError(t, err)
	expected := configurator.NetworkEntity{
		NetworkID: "n1",
		Type:      lte.ApnEntityType,
		Key:       "foo",
		Config:    payload.ApnConfiguration,
		GraphID:   "2",
	}
	assert.Equal(t, expected, actual)
}

func TestListApns(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/apns"
	handlers := handlers.GetHandlers()
	listApns := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.GET).HandlerFunc

	tc := tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        listApns,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(map[string]*lteModels.Apn{}),
	}
	tests.RunUnitTest(t, e, tc)

	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{
				Type: lte.ApnEntityType, Key: "oai.ipv4",
				Config: &lteModels.ApnConfiguration{
					Ambr: &lteModels.AggregatedMaximumBitrate{
						MaxBandwidthDl: swag.Uint32(200),
						MaxBandwidthUl: swag.Uint32(200),
					},
					QosProfile: &lteModels.QosProfile{
						ClassID:                 swag.Int32(9),
						PreemptionCapability:    swag.Bool(true),
						PreemptionVulnerability: swag.Bool(false),
						PriorityLevel:           swag.Uint32(15),
					},
				},
			},
			{
				Type: lte.ApnEntityType, Key: "oai.ims",
				Config: &lteModels.ApnConfiguration{
					Ambr: &lteModels.AggregatedMaximumBitrate{
						MaxBandwidthDl: swag.Uint32(100),
						MaxBandwidthUl: swag.Uint32(100),
					},
					QosProfile: &lteModels.QosProfile{
						ClassID:                 swag.Int32(5),
						PreemptionCapability:    swag.Bool(true),
						PreemptionVulnerability: swag.Bool(false),
						PriorityLevel:           swag.Uint32(5),
					},
				},
			},
		},
	)
	assert.NoError(t, err)

	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        listApns,
		ParamNames:     []string{"network_id"},
		ParamValues:    []string{"n1"},
		ExpectedStatus: 200,
		ExpectedResult: tests.JSONMarshaler(map[string]*lteModels.Apn{
			"oai.ipv4": {
				ApnName: "oai.ipv4",
				ApnConfiguration: &lteModels.ApnConfiguration{
					Ambr: &lteModels.AggregatedMaximumBitrate{
						MaxBandwidthDl: swag.Uint32(200),
						MaxBandwidthUl: swag.Uint32(200),
					},
					QosProfile: &lteModels.QosProfile{
						ClassID:                 swag.Int32(9),
						PreemptionCapability:    swag.Bool(true),
						PreemptionVulnerability: swag.Bool(false),
						PriorityLevel:           swag.Uint32(15),
					},
				},
			},
			"oai.ims": {
				ApnName: "oai.ims",
				ApnConfiguration: &lteModels.ApnConfiguration{
					Ambr: &lteModels.AggregatedMaximumBitrate{
						MaxBandwidthDl: swag.Uint32(100),
						MaxBandwidthUl: swag.Uint32(100),
					},
					QosProfile: &lteModels.QosProfile{
						ClassID:                 swag.Int32(5),
						PreemptionCapability:    swag.Bool(true),
						PreemptionVulnerability: swag.Bool(false),
						PriorityLevel:           swag.Uint32(5),
					},
				},
			},
		}),
	}
	tests.RunUnitTest(t, e, tc)
}

func TestGetApn(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/apns/:apn_name"
	handlers := handlers.GetHandlers()
	getApn := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.GET).HandlerFunc

	tc := tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getApn,
		ParamNames:     []string{"network_id", "apn_name"},
		ParamValues:    []string{"n1", "oai.ipv4"},
		ExpectedStatus: 404,
		ExpectedError:  "Not Found",
	}
	tests.RunUnitTest(t, e, tc)

	_, err = configurator.CreateEntity(
		"n1",
		configurator.NetworkEntity{
			Type: lte.ApnEntityType, Key: "oai.ipv4",
			Config: &lteModels.ApnConfiguration{
				Ambr: &lteModels.AggregatedMaximumBitrate{
					MaxBandwidthDl: swag.Uint32(200),
					MaxBandwidthUl: swag.Uint32(200),
				},
				QosProfile: &lteModels.QosProfile{
					ClassID:                 swag.Int32(9),
					PreemptionCapability:    swag.Bool(true),
					PreemptionVulnerability: swag.Bool(false),
					PriorityLevel:           swag.Uint32(15),
				},
			},
		},
	)
	assert.NoError(t, err)

	tc = tests.Test{
		Method:         "GET",
		URL:            testURLRoot,
		Handler:        getApn,
		ParamNames:     []string{"network_id", "apn_name"},
		ParamValues:    []string{"n1", "oai.ipv4"},
		ExpectedStatus: 200,
		ExpectedResult: &lteModels.Apn{
			ApnName: "oai.ipv4",
			ApnConfiguration: &lteModels.ApnConfiguration{
				Ambr: &lteModels.AggregatedMaximumBitrate{
					MaxBandwidthDl: swag.Uint32(200),
					MaxBandwidthUl: swag.Uint32(200),
				},
				QosProfile: &lteModels.QosProfile{
					ClassID:                 swag.Int32(9),
					PreemptionCapability:    swag.Bool(true),
					PreemptionVulnerability: swag.Bool(false),
					PriorityLevel:           swag.Uint32(15),
				},
			},
		},
	}
	tests.RunUnitTest(t, e, tc)
}

func TestUpdateApn(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/apns/:apn_name"
	handlers := handlers.GetHandlers()
	updateApn := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.PUT).HandlerFunc

	// 404
	payload := &lteModels.Apn{
		ApnName: "oai.ipv4",
		ApnConfiguration: &lteModels.ApnConfiguration{
			Ambr: &lteModels.AggregatedMaximumBitrate{
				MaxBandwidthDl: swag.Uint32(100),
				MaxBandwidthUl: swag.Uint32(100),
			},
			QosProfile: &lteModels.QosProfile{
				ClassID:                 swag.Int32(5),
				PreemptionCapability:    swag.Bool(true),
				PreemptionVulnerability: swag.Bool(false),
				PriorityLevel:           swag.Uint32(5),
			},
		},
	}

	tc := tests.Test{
		Method:         "PUT",
		URL:            testURLRoot,
		Handler:        updateApn,
		Payload:        payload,
		ParamNames:     []string{"network_id", "apn_name"},
		ParamValues:    []string{"n1", "oai.ipv4"},
		ExpectedStatus: 404,
		ExpectedError:  "Not Found",
	}
	tests.RunUnitTest(t, e, tc)

	// Add the APN Configuration
	_, err = configurator.CreateEntity(
		"n1",
		configurator.NetworkEntity{
			Type: lte.ApnEntityType, Key: "oai.ipv4",
			Config: &lteModels.ApnConfiguration{
				Ambr: &lteModels.AggregatedMaximumBitrate{
					MaxBandwidthDl: swag.Uint32(200),
					MaxBandwidthUl: swag.Uint32(200),
				},
				QosProfile: &lteModels.QosProfile{
					ClassID:                 swag.Int32(9),
					PreemptionCapability:    swag.Bool(true),
					PreemptionVulnerability: swag.Bool(false),
					PriorityLevel:           swag.Uint32(15),
				},
			},
		},
	)
	assert.NoError(t, err)

	tc = tests.Test{
		Method:         "PUT",
		URL:            testURLRoot,
		Handler:        updateApn,
		Payload:        payload,
		ParamNames:     []string{"network_id", "apn_name"},
		ParamValues:    []string{"n1", "oai.ipv4"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actual, err := configurator.LoadEntity("n1", lte.ApnEntityType, "oai.ipv4", configurator.FullEntityLoadCriteria())
	assert.NoError(t, err)
	expected := configurator.NetworkEntity{
		NetworkID: "n1",
		Type:      lte.ApnEntityType,
		Key:       "oai.ipv4",
		Config:    payload.ApnConfiguration,
		GraphID:   "2",
		Version:   1,
	}
	assert.Equal(t, expected, actual)
}

func TestDeleteApn(t *testing.T) {
	_ = plugin.RegisterPluginForTests(t, &pluginimpl.BaseOrchestratorPlugin{})
	_ = plugin.RegisterPluginForTests(t, &ltePlugin.LteOrchestratorPlugin{})

	test_init.StartTestService(t)
	err := configurator.CreateNetwork(configurator.Network{ID: "n1"})
	assert.NoError(t, err)

	e := echo.New()
	testURLRoot := "/magma/v1/lte/:network_id/apns/:apn_name"
	handlers := handlers.GetHandlers()
	deleteApn := tests.GetHandlerByPathAndMethod(t, handlers, testURLRoot, obsidian.DELETE).HandlerFunc

	_, err = configurator.CreateEntities(
		"n1",
		[]configurator.NetworkEntity{
			{
				Type: lte.ApnEntityType, Key: "oai.ipv4",
				Config: &lteModels.ApnConfiguration{
					Ambr: &lteModels.AggregatedMaximumBitrate{
						MaxBandwidthDl: swag.Uint32(200),
						MaxBandwidthUl: swag.Uint32(200),
					},
					QosProfile: &lteModels.QosProfile{
						ClassID:                 swag.Int32(9),
						PreemptionCapability:    swag.Bool(true),
						PreemptionVulnerability: swag.Bool(false),
						PriorityLevel:           swag.Uint32(15),
					},
				},
			},
			{
				Type: lte.ApnEntityType, Key: "oai.ims",
				Config: &lteModels.ApnConfiguration{
					Ambr: &lteModels.AggregatedMaximumBitrate{
						MaxBandwidthDl: swag.Uint32(100),
						MaxBandwidthUl: swag.Uint32(100),
					},
					QosProfile: &lteModels.QosProfile{
						ClassID:                 swag.Int32(5),
						PreemptionCapability:    swag.Bool(true),
						PreemptionVulnerability: swag.Bool(false),
						PriorityLevel:           swag.Uint32(5),
					},
				},
			},
		},
	)
	assert.NoError(t, err)

	tc := tests.Test{
		Method:         "DELETE",
		URL:            testURLRoot,
		Handler:        deleteApn,
		ParamNames:     []string{"network_id", "apn_name"},
		ParamValues:    []string{"n1", "oai.ipv4"},
		ExpectedStatus: 204,
	}
	tests.RunUnitTest(t, e, tc)

	actual, err := configurator.LoadAllEntitiesInNetwork("n1", lte.ApnEntityType, configurator.FullEntityLoadCriteria())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actual))
	expected := configurator.NetworkEntity{
		NetworkID: "n1",
		Type:      lte.ApnEntityType,
		Key:       "oai.ims",
		Config: &lteModels.ApnConfiguration{
			Ambr: &lteModels.AggregatedMaximumBitrate{
				MaxBandwidthDl: swag.Uint32(100),
				MaxBandwidthUl: swag.Uint32(100),
			},
			QosProfile: &lteModels.QosProfile{
				ClassID:                 swag.Int32(5),
				PreemptionCapability:    swag.Bool(true),
				PreemptionVulnerability: swag.Bool(false),
				PriorityLevel:           swag.Uint32(5),
			},
		},
		GraphID: "4",
		Version: 0,
	}
	assert.Equal(t, expected, actual[0])
}

func reportEnodebState(t *testing.T, ctx context.Context, enodebSerial string, req *lteModels.EnodebState) {
	client, err := state.GetStateClient()
	assert.NoError(t, err)

	serializedEnodebState, err := serde.Serialize(state.SerdeDomain, lte.EnodebStateType, req)
	assert.NoError(t, err)
	states := []*protos.State{
		{
			Type:     lte.EnodebStateType,
			DeviceID: enodebSerial,
			Value:    serializedEnodebState,
		},
	}
	_, err = client.ReportStates(
		ctx,
		&protos.ReportStatesRequest{States: states},
	)
	assert.NoError(t, err)
}

// n1, n3 are lte networks, n2 is not
func seedNetworks(t *testing.T) {
	_, err := configurator.CreateNetworks(
		[]configurator.Network{
			{
				ID:          "n1",
				Type:        lte.NetworkType,
				Name:        "foobar",
				Description: "Foo Bar",
				Configs: map[string]interface{}{
					lte.CellularNetworkType:     lteModels.NewDefaultTDDNetworkConfig(),
					orc8r.NetworkFeaturesConfig: models.NewDefaultFeaturesConfig(),
					orc8r.DnsdNetworkType:       models.NewDefaultDNSConfig(),
				},
			},
			{
				ID:          "n2",
				Type:        "blah",
				Name:        "foobar",
				Description: "Foo Bar",
				Configs:     map[string]interface{}{},
			},
			{
				ID:          "n3",
				Type:        lte.NetworkType,
				Name:        "barfoo",
				Description: "Bar Foo",
				Configs:     map[string]interface{}{},
			},
		},
	)
	assert.NoError(t, err)
}

func newDefaultGatewayConfig() *lteModels.GatewayCellularConfigs {
	return &lteModels.GatewayCellularConfigs{
		Ran: &lteModels.GatewayRanConfigs{
			Pci:             260,
			TransmitEnabled: swag.Bool(true),
		},
		Epc: &lteModels.GatewayEpcConfigs{
			NatEnabled: swag.Bool(true),
			IPBlock:    "192.168.128.0/24",
		},
		NonEpsService: &lteModels.GatewayNonEpsConfigs{
			CsfbMcc:              "001",
			CsfbMnc:              "01",
			Lac:                  swag.Uint32(1),
			CsfbRat:              swag.Uint32(0),
			Arfcn2g:              []uint32{},
			NonEpsServiceControl: swag.Uint32(0),
		},
	}
}