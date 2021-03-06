/*
Copyright 2015 The Kubernetes Authors.

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

package kubemark

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	clientv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
	proxyapp "github.com/sourcegraph/monorepo-test-1/kubernetes-14/cmd/kube-proxy/app"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-14/cmd/kube-proxy/app/options"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-14/pkg/api"
	clientset "github.com/sourcegraph/monorepo-test-1/kubernetes-14/pkg/client/clientset_generated/internalclientset"
	informers "github.com/sourcegraph/monorepo-test-1/kubernetes-14/pkg/client/informers/informers_generated/internalversion"
	proxyconfig "github.com/sourcegraph/monorepo-test-1/kubernetes-14/pkg/proxy/config"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-14/pkg/util"
	utiliptables "github.com/sourcegraph/monorepo-test-1/kubernetes-14/pkg/util/iptables"

	"github.com/golang/glog"
)

type HollowProxy struct {
	ProxyServer *proxyapp.ProxyServer
}

type FakeProxyHandler struct{}

func (*FakeProxyHandler) OnServiceUpdate(services []*api.Service)                  {}
func (*FakeProxyHandler) OnEndpointsAdd(endpoints *api.Endpoints)                  {}
func (*FakeProxyHandler) OnEndpointsUpdate(oldEndpoints, endpoints *api.Endpoints) {}
func (*FakeProxyHandler) OnEndpointsDelete(endpoints *api.Endpoints)               {}
func (*FakeProxyHandler) OnEndpointsSynced()                                       {}

type FakeProxier struct{}

func (*FakeProxier) OnServiceUpdate(services []*api.Service) {}
func (*FakeProxier) Sync()                                   {}
func (*FakeProxier) SyncLoop() {
	select {}
}

func NewHollowProxyOrDie(
	nodeName string,
	client clientset.Interface,
	eventClient v1core.EventsGetter,
	endpointsConfig *proxyconfig.EndpointsConfig,
	serviceConfig *proxyconfig.ServiceConfig,
	informerFactory informers.SharedInformerFactory,
	iptInterface utiliptables.Interface,
	broadcaster record.EventBroadcaster,
	recorder record.EventRecorder,
) *HollowProxy {
	// Create and start Hollow Proxy
	config := options.NewProxyConfig()
	config.OOMScoreAdj = util.Int32Ptr(0)
	config.ResourceContainer = ""
	config.NodeRef = &clientv1.ObjectReference{
		Kind:      "Node",
		Name:      nodeName,
		UID:       types.UID(nodeName),
		Namespace: "",
	}

	go endpointsConfig.Run(wait.NeverStop)
	go serviceConfig.Run(wait.NeverStop)
	go informerFactory.Start(wait.NeverStop)

	hollowProxy, err := proxyapp.NewProxyServer(client, eventClient, config, iptInterface, &FakeProxier{}, broadcaster, recorder, nil, "fake")
	if err != nil {
		glog.Fatalf("Error while creating ProxyServer: %v\n", err)
	}
	return &HollowProxy{
		ProxyServer: hollowProxy,
	}
}

func (hp *HollowProxy) Run() {
	if err := hp.ProxyServer.Run(); err != nil {
		glog.Fatalf("Error while running proxy: %v\n", err)
	}
}
