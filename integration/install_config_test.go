// Copyright 2017 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

import (
	"fmt"
	"os"
	"strings"

	check "gopkg.in/check.v1"
)

func (s *S) getInstallerConfig() string {
	hosts := len(provisioners)
	for _, m := range clusterManagers {
		if req, ok := m.(interface {
			RequiredNodes() int
		}); ok {
			hosts += req.RequiredNodes()
		}
	}
	// if no host is set, add one, so Tsuru can build platforms
	if hosts == 0 {
		hosts = 1
	}
	return fmt.Sprintf(`driver:
  name: virtualbox
  options:
    virtualbox-cpu-count: 2
    virtualbox-memory: 2048
docker-flags:
  - experimental
hosts:
  apps:
    size: %d
components:
  tsuru-image: tsuru/api:latest
  install-dashboard: false
`, hosts)
}

func (s *S) getPlatforms() []string {
	availablePlatforms := []string{
		"tsuru/python",
		"tsuru/go",
		"tsuru/cordova",
		"tsuru/elixir",
		"tsuru/java",
		"tsuru/nodejs",
		"tsuru/php",
		"tsuru/play",
		"tsuru/ruby",
		"tsuru/static",
		"tsuru/perl",
	}
	if _, ok := os.LookupEnv(integrationEnvID + "platforms"); !ok {
		return availablePlatforms
	}
	envPlatforms := s.env.All("platforms")
	selectedPlatforms := make([]string, 0, len(availablePlatforms))
	for _, name := range envPlatforms {
		name = strings.Trim(name, " ")
		for i, platform := range availablePlatforms {
			if name == platform || "tsuru/"+name == platform {
				selectedPlatforms = append(selectedPlatforms, platform)
				availablePlatforms = append(availablePlatforms[:i], availablePlatforms[i+1:]...)
				break
			}
		}
	}
	return selectedPlatforms
}

func (s *S) getProvisioners() []string {
	availableProvisioners := []string{"docker"}
	if _, ok := os.LookupEnv(integrationEnvID + "provisioners"); !ok {
		return availableProvisioners
	}
	selectedProvisioners := make([]string, 0, len(availableProvisioners))
	for _, provisioner := range s.env.All("provisioners") {
		provisioner = strings.Trim(provisioner, " ")
		for i, item := range availableProvisioners {
			if item == provisioner {
				selectedProvisioners = append(selectedProvisioners, provisioner)
				availableProvisioners = append(availableProvisioners[:i], availableProvisioners[i+1:]...)
				break
			}
		}
	}
	return selectedProvisioners
}

func (s *S) getClusterManagers(c *check.C) []ClusterManager {
	availableClusterManagers := map[string]ClusterManager{
		"minikube": &MinikubeClusterManager{env: s.env},
		"kubectl": &KubectlClusterManager{
			env:     s.env,
			config:  s.env.Get("kubectlconfig"),
			context: s.env.Get("kubectlctx"),
			binary:  s.env.Get("kubectlbinary"),
		},
		"kubeenv": &KubeenvClusterManager{
			env: s.env,
		},
	}
	managers := make([]ClusterManager, 0, len(availableClusterManagers))
	clusters := s.env.All("clusters")
	for _, cluster := range clusters {
		cluster = strings.Trim(cluster, " ")
		manager := availableClusterManagers[cluster]
		if manager == nil {
			continue
		}
		managers = append(managers, manager)
	}
	return managers
}

func installerName(env *Environment) string {
	name := env.Get("installername")
	if name == "" {
		name = "tsuru"
	}
	return name
}

func (s *S) config(c *check.C) {
	env := NewEnvironment()
	if !env.Has("enabled") {
		return
	}
	s.env = env
	platforms = s.getPlatforms()
	provisioners = s.getProvisioners()
	clusterManagers = s.getClusterManagers(c)
	installerConfig = s.getInstallerConfig()
}
