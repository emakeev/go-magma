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

#define LOG_WITH_GLOG
#include <magma_logging.h>

#include <devmand/channels/cli/engine/Engine.h>
#include <devmand/test/cli/utils/Log.h>
#include <folly/json.h>
#include <gtest/gtest.h>

namespace devmand {
namespace test {
namespace cli {

using namespace devmand::channels::cli;

class EngineTest : public ::testing::Test {
 protected:
  void SetUp() override {
    devmand::test::utils::log::initLog();
  }
};

TEST_F(EngineTest, parseGrpcPlugins_noFailOnEmptyConfig) {
  map<string, string> actual = Engine::parseGrpcPlugins(dynamic::object());
  EXPECT_TRUE(actual.empty());
}

TEST_F(EngineTest, parseGrpcPlugins_wrongType) {
  dynamic config = dynamic::object();
  config["grpcPlugins"] = 2;
  map<string, string> actual = Engine::parseGrpcPlugins(config);
  EXPECT_TRUE(actual.empty());
}

TEST_F(EngineTest, parseGrpcPlugins_testSomePlugins) {
  dynamic config = dynamic::object();
  dynamic grpcPlugins = dynamic::array();
  {
    dynamic plugin = dynamic::object();
    plugin["id"] = "someapp";
    plugin["endpoint"] = "localhost:1234";
    grpcPlugins.push_back(plugin);
  }
  {
    dynamic plugin = dynamic::object();
    plugin["id"] = "someapp2";
    plugin["endpoint"] = "localhost:12345";
    grpcPlugins.push_back(plugin);
  }
  config["grpcPlugins"] = grpcPlugins;
  map<string, string> actual = Engine::parseGrpcPlugins(config);
  EXPECT_EQ(actual.size(), 2);
  map<string, string> expected;
  expected["someapp"] = "localhost:1234";
  expected["someapp2"] = "localhost:12345";
  EXPECT_EQ(actual, expected);
}

} // namespace cli
} // namespace test
} // namespace devmand
