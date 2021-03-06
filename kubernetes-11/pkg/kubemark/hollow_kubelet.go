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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeletapp "github.com/sourcegraph/monorepo-test-1/kubernetes-11/cmd/kubelet/app"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/cmd/kubelet/app/options"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/api"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/apis/componentconfig"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/apis/componentconfig/v1alpha1"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/client/clientset_generated/clientset"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/kubelet"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/kubelet/cadvisor"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/kubelet/cm"
	containertest "github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/kubelet/container/testing"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/kubelet/dockertools"
	kubetypes "github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/kubelet/types"
	kubeio "github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/util/io"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/util/mount"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/util/oom"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/volume/empty_dir"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/volume/secret"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-11/test/utils"

	"github.com/golang/glog"
)

type HollowKubelet struct {
	KubeletFlags         *options.KubeletFlags
	KubeletConfiguration *componentconfig.KubeletConfiguration
	KubeletDeps          *kubelet.KubeletDeps
}

func NewHollowKubelet(
	nodeName string,
	client *clientset.Clientset,
	cadvisorInterface cadvisor.Interface,
	dockerClient dockertools.DockerInterface,
	kubeletPort, kubeletReadOnlyPort int,
	containerManager cm.ContainerManager,
	maxPods int, podsPerCore int,
) *HollowKubelet {
	// -----------------
	// Static config
	// -----------------
	f, c := GetHollowKubeletConfig(nodeName, kubeletPort, kubeletReadOnlyPort, maxPods, podsPerCore)

	// -----------------
	// Injected objects
	// -----------------
	volumePlugins := empty_dir.ProbeVolumePlugins()
	volumePlugins = append(volumePlugins, secret.ProbeVolumePlugins()...)
	d := &kubelet.KubeletDeps{
		KubeClient:        client,
		DockerClient:      dockerClient,
		CAdvisorInterface: cadvisorInterface,
		Cloud:             nil,
		OSInterface:       &containertest.FakeOS{},
		ContainerManager:  containerManager,
		VolumePlugins:     volumePlugins,
		TLSOptions:        nil,
		OOMAdjuster:       oom.NewFakeOOMAdjuster(),
		Writer:            &kubeio.StdWriter{},
		Mounter:           mount.New("" /* default mount path */),
	}

	return &HollowKubelet{
		KubeletFlags:         f,
		KubeletConfiguration: c,
		KubeletDeps:          d,
	}
}

// Starts this HollowKubelet and blocks.
func (hk *HollowKubelet) Run() {
	kubeletapp.RunKubelet(hk.KubeletFlags, hk.KubeletConfiguration, hk.KubeletDeps, false, false)
	select {}
}

// Builds a KubeletConfiguration for the HollowKubelet, ensuring that the
// usual defaults are applied for fields we do not override.
func GetHollowKubeletConfig(
	nodeName string,
	kubeletPort int,
	kubeletReadOnlyPort int,
	maxPods int,
	podsPerCore int) (*options.KubeletFlags, *componentconfig.KubeletConfiguration) {

	testRootDir := utils.MakeTempDirOrDie("hollow-kubelet.", "")
	manifestFilePath := utils.MakeTempDirOrDie("manifest", testRootDir)
	glog.Infof("Using %s as root dir for hollow-kubelet", testRootDir)

	// Flags struct
	f := &options.KubeletFlags{
		HostnameOverride: nodeName,
	}

	// Config struct
	// Do the external -> internal conversion to make sure that defaults
	// are set for fields not overridden in NewHollowKubelet.
	tmp := &v1alpha1.KubeletConfiguration{}
	api.Scheme.Default(tmp)
	c := &componentconfig.KubeletConfiguration{}
	api.Scheme.Convert(tmp, c, nil)

	c.RootDirectory = testRootDir
	c.ManifestURL = ""
	c.Address = "0.0.0.0" /* bind address */
	c.Port = int32(kubeletPort)
	c.ReadOnlyPort = int32(kubeletReadOnlyPort)
	c.MasterServiceNamespace = metav1.NamespaceDefault
	c.PodManifestPath = manifestFilePath
	c.FileCheckFrequency.Duration = 20 * time.Second
	c.HTTPCheckFrequency.Duration = 20 * time.Second
	c.MinimumGCAge.Duration = 1 * time.Minute
	c.NodeStatusUpdateFrequency.Duration = 10 * time.Second
	c.SyncFrequency.Duration = 10 * time.Second
	c.OutOfDiskTransitionFrequency.Duration = 5 * time.Minute
	c.EvictionPressureTransitionPeriod.Duration = 5 * time.Minute
	c.MaxPods = int32(maxPods)
	c.PodsPerCore = int32(podsPerCore)
	c.ClusterDNS = []string{}
	c.DockerExecHandlerName = "native"
	c.ImageGCHighThresholdPercent = 90
	c.ImageGCLowThresholdPercent = 80
	c.LowDiskSpaceThresholdMB = 256
	c.VolumeStatsAggPeriod.Duration = time.Minute
	c.CgroupRoot = ""
	c.ContainerRuntime = "docker"
	c.CPUCFSQuota = true
	c.RuntimeCgroups = ""
	c.EnableControllerAttachDetach = false
	c.EnableCustomMetrics = false
	c.EnableDebuggingHandlers = true
	c.EnableServer = true
	c.CgroupsPerQOS = false
	// hairpin-veth is used to allow hairpin packets. Note that this deviates from
	// what the "real" kubelet currently does, because there's no way to
	// set promiscuous mode on docker0.
	c.HairpinMode = componentconfig.HairpinVeth
	c.MaxContainerCount = 100
	c.MaxOpenFiles = 1024
	c.MaxPerPodContainerCount = 2
	c.RegisterNode = true
	c.RegisterSchedulable = true
	c.RegistryBurst = 10
	c.RegistryPullQPS = 5.0
	c.ResolverConfig = kubetypes.ResolvConfDefault
	c.KubeletCgroups = "/kubelet"
	c.SerializeImagePulls = true
	c.SystemCgroups = ""
	c.ProtectKernelDefaults = false

	// TODO(mtaufen): Note that PodInfraContainerImage was being set to the empty value before,
	//                but this may not have been intentional. (previous code (SimpleKubelet)
	//                was peeling it off of a componentconfig.KubeletConfiguration{}, but may
	//                have actually wanted the default).
	//                The default will be present in the KubeletConfiguration contstructed above.

	return f, c

}
