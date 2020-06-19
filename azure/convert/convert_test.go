/*
   Copyright 2020 Docker, Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package convert

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerinstance/mgmt/containerinstance"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/compose-spec/compose-go/types"

	"github.com/docker/api/compose"
	"github.com/docker/api/containers"
	"github.com/docker/api/context/store"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConvertTestSuite struct {
	suite.Suite
	ctx store.AciContext
}

func (suite *ConvertTestSuite) BeforeTest(suiteName, testName string) {
	suite.ctx = store.AciContext{
		SubscriptionID: "subID",
		ResourceGroup:  "rg",
		Location:       "eu",
	}
}

func (suite *ConvertTestSuite) TestProjectName() {
	project := compose.Project{
		Name: "TEST",
	}
	containerGroup, err := ToContainerGroup(suite.ctx, project)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), *containerGroup.Name, "test")
}

func (suite *ConvertTestSuite) TestContainerGroupToContainer() {
	myContainerGroup := containerinstance.ContainerGroup{
		ContainerGroupProperties: &containerinstance.ContainerGroupProperties{
			IPAddress: &containerinstance.IPAddress{
				Ports: &[]containerinstance.Port{{
					Port: to.Int32Ptr(80),
				}},
				IP: to.StringPtr("42.42.42.42"),
			},
		},
	}
	myContainer := containerinstance.Container{
		Name: to.StringPtr("myContainerID"),
		ContainerProperties: &containerinstance.ContainerProperties{
			Image:   to.StringPtr("sha256:666"),
			Command: to.StringSlicePtr([]string{"mycommand"}),
			Ports: &[]containerinstance.ContainerPort{{
				Port: to.Int32Ptr(80),
			}},
			EnvironmentVariables: nil,
			InstanceView: &containerinstance.ContainerPropertiesInstanceView{
				RestartCount: nil,
				CurrentState: &containerinstance.ContainerState{
					State: to.StringPtr("Running"),
				},
			},
			Resources: &containerinstance.ResourceRequirements{
				Limits: &containerinstance.ResourceLimits{
					MemoryInGB: to.Float64Ptr(9),
				},
			},
		},
	}

	var expectedContainer = containers.Container{
		ID:          "myContainerID",
		Status:      "Running",
		Image:       "sha256:666",
		Command:     "mycommand",
		MemoryLimit: 9,
		Ports: []containers.Port{{
			HostPort:      uint32(80),
			ContainerPort: uint32(80),
			Protocol:      "tcp",
			HostIP:        "42.42.42.42",
		}},
	}

	container, err := ContainerGroupToContainer("myContainerID", myContainerGroup, myContainer)
	Expect(err).To(BeNil())
	Expect(container).To(Equal(expectedContainer))
}

func (suite *ConvertTestSuite) TestComposeContainerGroupToContainerWithDnsSideCarSide() {
	project := compose.Project{
		Name: "",
		Config: types.Config{
			Services: []types.ServiceConfig{
				{
					Name:  "service1",
					Image: "image1",
				},
				{
					Name:  "service2",
					Image: "image2",
				},
			},
		},
	}

	group, err := ToContainerGroup(suite.ctx, project)
	Expect(err).To(BeNil())
	Expect(len(*group.Containers)).To(Equal(3))

	Expect(*(*group.Containers)[0].Name).To(Equal("service1"))
	Expect(*(*group.Containers)[1].Name).To(Equal("service2"))
	Expect(*(*group.Containers)[2].Name).To(Equal(ComposeDNSSidecarName))

	Expect(*(*group.Containers)[2].Command).To(Equal([]string{"sh", "-c", "echo 127.0.0.1 service1 >> /etc/hosts;echo 127.0.0.1 service2 >> /etc/hosts;sleep infinity"}))

	Expect(*(*group.Containers)[0].Image).To(Equal("image1"))
	Expect(*(*group.Containers)[1].Image).To(Equal("image2"))
	Expect(*(*group.Containers)[2].Image).To(Equal(dnsSidecarImage))
}

func (suite *ConvertTestSuite) TestComposeSingleContainerGroupToContainerNoDnsSideCarSide() {
	project := compose.Project{
		Name: "",
		Config: types.Config{
			Services: []types.ServiceConfig{
				{
					Name:  "service1",
					Image: "image1",
				},
			},
		},
	}

	group, err := ToContainerGroup(suite.ctx, project)
	Expect(err).To(BeNil())

	Expect(len(*group.Containers)).To(Equal(1))
	Expect(*(*group.Containers)[0].Name).To(Equal("service1"))
	Expect(*(*group.Containers)[0].Image).To(Equal("image1"))
}

func TestConvertTestSuite(t *testing.T) {
	RegisterTestingT(t)
	suite.Run(t, new(ConvertTestSuite))
}
