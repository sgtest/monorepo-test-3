/*
Copyright 2017 The Kubernetes Authors.

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

package populator

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/api/v1"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/client/clientset_generated/clientset/fake"
	containertest "github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/kubelet/container/testing"
	kubepod "github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/kubelet/pod"
	podtest "github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/kubelet/pod/testing"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/kubelet/secret"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/kubelet/status"
	statustest "github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/kubelet/status/testing"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/kubelet/volumemanager/cache"
	volumetesting "github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/volume/testing"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/volume/util/types"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-8/pkg/volume/util/volumehelper"
)

func TestFindAndAddNewPods_FindAndRemoveDeletedPods(t *testing.T) {
	fakeVolumePluginMgr, _ := volumetesting.GetTestVolumePluginMgr(t)
	fakeClient := &fake.Clientset{}

	fakeSecretManager := secret.NewFakeManager()
	fakePodManager := kubepod.NewBasicPodManager(
		podtest.NewFakeMirrorClient(), fakeSecretManager)

	fakesDSW := cache.NewDesiredStateOfWorld(fakeVolumePluginMgr)
	fakeRuntime := &containertest.FakeRuntime{}

	fakeStatusManager := status.NewManager(fakeClient, fakePodManager, &statustest.FakePodDeletionSafetyProvider{})

	dswp := &desiredStateOfWorldPopulator{
		kubeClient:                fakeClient,
		loopSleepDuration:         100 * time.Millisecond,
		getPodStatusRetryDuration: 2 * time.Second,
		podManager:                fakePodManager,
		podStatusProvider:         fakeStatusManager,
		desiredStateOfWorld:       fakesDSW,
		pods: processedPods{
			processedPods: make(map[types.UniquePodName]bool)},
		kubeContainerRuntime:     fakeRuntime,
		keepTerminatedPodVolumes: false,
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dswp-test-pod",
			UID:       "dswp-test-pod-uid",
			Namespace: "dswp-test",
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "dswp-test-volume-name",
					VolumeSource: v1.VolumeSource{
						GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{
							PDName: "dswp-test-fake-device",
						},
					},
				},
			},
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase("Running"),
		},
	}

	fakePodManager.AddPod(pod)

	podName := volumehelper.GetUniquePodName(pod)

	generatedVolumeName := "fake-plugin/" + pod.Spec.Volumes[0].Name

	dswp.findAndAddNewPods()

	if !dswp.pods.processedPods[podName] {
		t.Fatalf("Failed to record that the volumes for the specified pod: %s have been processed by the populator", podName)
	}

	expectedVolumeName := v1.UniqueVolumeName(generatedVolumeName)

	volumeExists := fakesDSW.VolumeExists(expectedVolumeName)
	if !volumeExists {
		t.Fatalf(
			"VolumeExists(%q) failed. Expected: <true> Actual: <%v>",
			expectedVolumeName,
			volumeExists)
	}

	if podExistsInVolume := fakesDSW.PodExistsInVolume(
		podName, expectedVolumeName); !podExistsInVolume {
		t.Fatalf(
			"DSW PodExistsInVolume returned incorrect value. Expected: <true> Actual: <%v>",
			podExistsInVolume)
	}

	verifyVolumeExistsInVolumesToMount(
		t, v1.UniqueVolumeName(generatedVolumeName), false /* expectReportedInUse */, fakesDSW)

	//let the pod be terminated
	podGet, exist := fakePodManager.GetPodByName(pod.Namespace, pod.Name)
	if !exist {
		t.Fatalf("Failed to get pod by pod name: %s and namespace: %s", pod.Name, pod.Namespace)
	}
	podGet.Status.Phase = v1.PodFailed

	//pod is added to fakePodManager but fakeRuntime can not get the pod,so here findAndRemoveDeletedPods() will remove the pod and volumes it is mounted
	dswp.findAndRemoveDeletedPods()

	if dswp.pods.processedPods[podName] {
		t.Fatalf("Failed to remove pods from desired state of world since they no longer exist")
	}

	volumeExists = fakesDSW.VolumeExists(expectedVolumeName)
	if volumeExists {
		t.Fatalf(
			"VolumeExists(%q) failed. Expected: <false> Actual: <%v>",
			expectedVolumeName,
			volumeExists)
	}

	if podExistsInVolume := fakesDSW.PodExistsInVolume(
		podName, expectedVolumeName); podExistsInVolume {
		t.Fatalf(
			"DSW PodExistsInVolume returned incorrect value. Expected: <false> Actual: <%v>",
			podExistsInVolume)
	}

	volumesToMount := fakesDSW.GetVolumesToMount()
	for _, volume := range volumesToMount {
		if volume.VolumeName == expectedVolumeName {
			t.Fatalf(
				"Found volume %v in the list of desired state of world volumes to mount. Expected not",
				expectedVolumeName)
		}
	}

}

func verifyVolumeExistsInVolumesToMount(t *testing.T, expectedVolumeName v1.UniqueVolumeName, expectReportedInUse bool, dsw cache.DesiredStateOfWorld) {
	volumesToMount := dsw.GetVolumesToMount()
	for _, volume := range volumesToMount {
		if volume.VolumeName == expectedVolumeName {
			if volume.ReportedInUse != expectReportedInUse {
				t.Fatalf(
					"Found volume %v in the list of VolumesToMount, but ReportedInUse incorrect. Expected: <%v> Actual: <%v>",
					expectedVolumeName,
					expectReportedInUse,
					volume.ReportedInUse)
			}

			return
		}
	}

	t.Fatalf(
		"Could not find volume %v in the list of desired state of world volumes to mount %+v",
		expectedVolumeName,
		volumesToMount)
}
