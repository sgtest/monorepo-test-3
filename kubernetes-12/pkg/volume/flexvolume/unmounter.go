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

package flexvolume

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-12/pkg/util/exec"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-12/pkg/volume"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-12/pkg/volume/util"
)

// FlexVolumeUnmounter is the disk that will be cleaned by this plugin.
type flexVolumeUnmounter struct {
	*flexVolume
	// Runner used to teardown the volume.
	runner exec.Interface
	volume.MetricsNil
}

var _ volume.Unmounter = &flexVolumeUnmounter{}

// Unmounter interface
func (f *flexVolumeUnmounter) TearDown() error {
	path := f.GetPath()
	return f.TearDownAt(path)
}

func (f *flexVolumeUnmounter) TearDownAt(dir string) error {

	if pathExists, pathErr := util.PathExists(dir); pathErr != nil {
		return fmt.Errorf("Error checking if path exists: %v", pathErr)
	} else if !pathExists {
		glog.Warningf("Warning: Unmount skipped because path does not exist: %v", dir)
		return nil
	}

	notmnt, err := isNotMounted(f.mounter, dir)
	if err != nil {
		return err
	}
	if notmnt {
		glog.Warningf("Warning: Path: %v already unmounted", dir)
	} else {
		call := f.plugin.NewDriverCall(unmountCmd)
		call.Append(dir)
		_, err := call.Run()
		if isCmdNotSupportedErr(err) {
			err = (*unmounterDefaults)(f).TearDownAt(dir)
		}
		if err != nil {
			return err
		}
	}

	// Flexvolume driver may remove the directory. Ignore if it does.
	if pathExists, pathErr := util.PathExists(dir); pathErr != nil {
		return fmt.Errorf("Error checking if path exists: %v", pathErr)
	} else if !pathExists {
		return nil
	}
	return os.Remove(dir)
}
