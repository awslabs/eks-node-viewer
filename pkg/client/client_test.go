/*
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

package client

import (
	"os"
	"path/filepath"
	"testing"
)

const testKubeconfig = `apiVersion: v1
kind: Config
current-context: cluster-a
clusters:
- name: cluster-a
  cluster:
    server: https://a.example.com
- name: cluster-b
  cluster:
    server: https://b.example.com
contexts:
- name: cluster-a
  context:
    cluster: cluster-a
    user: user
- name: cluster-b
  context:
    cluster: cluster-b
    user: user
users:
- name: user
  user:
    token: fake
`

func writeKubeconfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing kubeconfig: %s", err)
	}
	return path
}

func TestGetConfigContextName(t *testing.T) {
	kubeconfig := writeKubeconfig(t, testKubeconfig)

	for _, tc := range []struct {
		name        string
		kubeconfig  string
		context     string
		wantContext string
		wantHost    string
	}{
		{
			name:        "current context",
			kubeconfig:  kubeconfig,
			wantContext: "cluster-a",
			wantHost:    "https://a.example.com",
		},
		{
			name:        "context override",
			kubeconfig:  kubeconfig,
			context:     "cluster-b",
			wantContext: "cluster-b",
			wantHost:    "https://b.example.com",
		},
		{
			name:        "multiple kubeconfig files",
			kubeconfig:  filepath.Join(t.TempDir(), "missing") + ":" + kubeconfig,
			wantContext: "cluster-a",
			wantHost:    "https://a.example.com",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contextName, config, err := getConfig(tc.kubeconfig, tc.context)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if contextName != tc.wantContext {
				t.Errorf("expected context %q, got %q", tc.wantContext, contextName)
			}
			if config.Host != tc.wantHost {
				t.Errorf("expected host %q, got %q", tc.wantHost, config.Host)
			}
		})
	}
}
