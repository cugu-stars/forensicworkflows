// Copyright (c) 2019 Siemens AG
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
//
// Author(s): Jonas Plum

package daggy_test

import (
	"github.com/forensicanalysis/forensicworkflows/daggy"
	"github.com/forensicanalysis/forensicworkflows/plugins/process"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"

	"github.com/forensicanalysis/forensicstore/goforensicstore"
)

func setup() (storeDir, pluginDir string, err error) {
	tempDir, err := ioutil.TempDir("", "forensicstoreprocesstest")
	if err != nil {
		return "", "", err
	}
	storeDir = filepath.Join(tempDir, "test")
	err = os.MkdirAll(storeDir, 0755)
	if err != nil {
		return "", "", err
	}

	pluginDir = filepath.Join(tempDir, "plugins", "process")
	err = os.MkdirAll(pluginDir, 0755)
	if err != nil {
		return "", "", err
	}

	err = copy.Copy(filepath.Join("..", "test"), storeDir)
	if err != nil {
		return "", "", err
	}
	err = copy.Copy(filepath.Join("..", "plugins", "process"), pluginDir)
	if err != nil {
		return "", "", err
	}

	infos, err := ioutil.ReadDir(pluginDir)
	if err != nil {
		return "", "", err
	}
	for _, info := range infos {
		err := os.Chmod(filepath.Join(pluginDir, info.Name()), 0755)
		if err != nil {
			return "", "", err
		}
	}

	return storeDir, pluginDir, nil
}

func cleanup(folders ...string) (err error) {
	for _, folder := range folders {
		err := os.RemoveAll(folder)
		if err != nil {
			return err
		}
	}
	return nil
}

func Test_processJob(t *testing.T) {
	log.Println("Start setup")
	storeDir, pluginDir, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("Setup done")
	defer cleanup(storeDir, pluginDir)

	type args struct {
		taskName string
		task     daggy.Task
	}
	tests := []struct {
		name      string
		storeName string
		args      args
		wantType  string
		wantCount int
		wantErr   bool
	}{
		{"test plugin", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "example"}}, "example", 0, false},
		{"test script not existing", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "foo"}}, "", 0, true},
		{"unknown type", "example1.forensicstore", args{" testtask", daggy.Task{Type: "foo", Command: "foo"}}, "", 0, true},
		{"test docker", "example1.forensicstore", args{" testtask", daggy.Task{Type: "docker", Image: "alpine", Command: "true"}}, "", 0, false},

		{"test hotfixes", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "hotfixes"}}, "hotfix", 14, false},
		{"test networking", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "networking"}}, "known_network", 9, false},
		{"test run-keys", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "run-keys"}}, "runkey", 10, false},
		{"test services", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "services"}}, "service", 624, false},
		{"test shimcache", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "shimcache"}}, "shimcache", 391, false},
		{"test software", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "software"}}, "uninstall_entry", 6, false},
		{"test prefetch", "example1.forensicstore", args{" testtask", daggy.Task{Type: "plugin", Command: "prefetch"}}, "prefetch", 261, false},
		{"test plaso", "example1.forensicstore", args{" testtask", daggy.Task{Type: "dockerfile", Dockerfile: "plaso"}}, "event", 72, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := daggy.Workflow{Tasks: map[string]daggy.Task{tt.args.taskName: tt.args.task}}
			workflow.SetupGraph()
			if err := workflow.Run(filepath.Join(storeDir, tt.storeName), pluginDir, process.Plugins, nil); (err != nil) != tt.wantErr {
				t.Errorf("runTask() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				store, err := goforensicstore.NewJSONLite(filepath.Join(storeDir, tt.storeName))
				if err != nil {
					t.Fatal(err)
				}

				log.Println("Start select")
				if tt.wantCount > 0 {
					items, err := store.Select(tt.wantType)
					if err != nil {
						t.Fatal(err)
					}
					if tt.wantCount != len(items) {
						t.Errorf("runTask() error, wrong number of resuls = %d, want %d (%v)", len(items), tt.wantCount, len(items))
					}
				}
			}
		})
	}
}
