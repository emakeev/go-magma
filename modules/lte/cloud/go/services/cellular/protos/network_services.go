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

package protos

import (
	"fmt"

	"github.com/go-magma/magma/modules/lte/cloud/go/protos/mconfig"
)

var networkServiceNameToEnumMap = map[string]NetworkEPCConfig_NetworkServices{
	"metering":           NetworkEPCConfig_METERING,
	"dpi":                NetworkEPCConfig_DPI,
	"policy_enforcement": NetworkEPCConfig_ENFORCEMENT,
}
var networkServiceEnumToNameMap = map[NetworkEPCConfig_NetworkServices]string{}

var networkServiceEnumToPipelineDServiceMap = map[NetworkEPCConfig_NetworkServices]mconfig.PipelineD_NetworkServices{
	NetworkEPCConfig_METERING:    mconfig.PipelineD_METERING,
	NetworkEPCConfig_DPI:         mconfig.PipelineD_DPI,
	NetworkEPCConfig_ENFORCEMENT: mconfig.PipelineD_ENFORCEMENT,
}

var defaultPipelineServiceEnums = []mconfig.PipelineD_NetworkServices{
	mconfig.PipelineD_ENFORCEMENT,
}

func init() {
	for name, enum := range networkServiceNameToEnumMap {
		networkServiceEnumToNameMap[enum] = name
	}
}

// GetNetworkServiceName returns the corresponding name presented to the user given a network service enum for storage
func GetNetworkServiceName(enum NetworkEPCConfig_NetworkServices) (string, error) {
	name, ok := networkServiceEnumToNameMap[enum]
	if !ok {
		return name, fmt.Errorf("Unknown network service enum: %s", enum)
	}
	return name, nil
}

// GetNetworkServiceEnum returns the corresponding enum for storage given a network service name presented to the user,
func GetNetworkServiceEnum(name string) (NetworkEPCConfig_NetworkServices, error) {
	enum, ok := networkServiceNameToEnumMap[name]
	if !ok {
		return enum, fmt.Errorf("Unknown network service name: %s", name)
	}
	return enum, nil
}

func getPipelineDService(enum NetworkEPCConfig_NetworkServices) (mconfig.PipelineD_NetworkServices, error) {
	apps, ok := networkServiceEnumToPipelineDServiceMap[enum]
	if !ok {
		return apps, fmt.Errorf("Unknown network service enum: %s", enum)
	}
	return apps, nil
}

// GetPipelineDServicesConfig returns a corresponding list of apps in PipelineD in the same order given a list of network service enums from storage
// If the list is empty, then it returns a default list of services
func GetPipelineDServicesConfig(networkServices []NetworkEPCConfig_NetworkServices) ([]mconfig.PipelineD_NetworkServices, error) {
	if networkServices == nil || len(networkServices) == 0 {
		return defaultPipelineServiceEnums, nil
	}
	var apps []mconfig.PipelineD_NetworkServices
	for _, service := range networkServices {
		translatedApps, err := getPipelineDService(service)
		if err != nil {
			return apps, err
		}
		apps = append(apps, translatedApps)
	}
	return apps, nil
}
