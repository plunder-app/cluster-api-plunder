package plunder

import (
	"fmt"
	"strings"

	"github.com/plunder-app/plunder/pkg/parlay/parlaytypes"
)

func uptimeCommand(host string) parlaytypes.TreasureMap {
	return parlaytypes.TreasureMap{
		Deployments: []parlaytypes.Deployment{
			parlaytypes.Deployment{
				Name:     "Cluster-API OS Provisioned test",
				Parallel: false,
				Hosts:    []string{host},
				Actions: []parlaytypes.Action{
					parlaytypes.Action{
						ActionType: "command",
						Command:    "uptime",
						Name:       "Cluster-API provisioning uptime command",
					},
				},
			},
		},
	}
}

func destroyCommand(host string) parlaytypes.TreasureMap {
	return parlaytypes.TreasureMap{
		Deployments: []parlaytypes.Deployment{
			parlaytypes.Deployment{
				Name:     "Cluster-API de-provisioning",
				Parallel: false,
				Hosts:    []string{host},
				Actions: []parlaytypes.Action{
					parlaytypes.Action{
						ActionType:     "command",
						Command:        "tee /proc/sys/kernel/sysrq",
						CommandPipeCmd: "echo \"1\"",
						Name:           "Cluster-API machine [enable sysrq]",
						CommandSudo:    "root",
					},
					parlaytypes.Action{
						ActionType:  "command",
						Command:     "dd if=/dev/zero of=/dev/sda bs=1024k count=1000",
						Name:        "Cluster-API machine [disk wipe]",
						CommandSudo: "root",
					},
					parlaytypes.Action{
						ActionType:     "command",
						Command:        "tee /proc/sysrq-trigger",
						CommandPipeCmd: "echo \"b\"",
						Name:           "Cluster-API machine [reset]",
						CommandSudo:    "root",
						Timeout:        2,
					},
				},
			},
		},
	}
}

// ActionsKubernetes - this will take the inputs and generate all of the deployment details needed to install a version of Kubernetes / Docker
func (c *Client) ActionsKubernetes(host, kubeVersion, dockerVersion string) {

	// The Kubernetes standard is to define versions such as v1.x.x, however the OS packages are 1.x.x (missing the "v")
	kubeVersionFix := strings.Replace(kubeVersion, "v", "", -1)
	c.deploymentMap = &parlaytypes.TreasureMap{
		Deployments: []parlaytypes.Deployment{
			parlaytypes.Deployment{
				Name:     "Cluster-API OS Package provisioning",
				Parallel: false,
				Hosts:    []string{host},
				Actions: []parlaytypes.Action{
					parlaytypes.Action{
						ActionType:     "command",
						Command:        "tee /etc/apt/sources.list",
						CommandPipeCmd: "echo -e \"deb http://uk.archive.ubuntu.com/ubuntu/ bionic main restricted universe multiverse\"",
						Name:           "Cluster-API provisioning [reset Ubuntu repositories]",
						CommandSudo:    "root",
						IgnoreFailure:  true,
					},
					parlaytypes.Action{
						ActionType:    "command",
						Command:       "sudo apt-get update",
						Name:          "Cluster-API provisioning [Ubuntu package update]",
						CommandSudo:   "root",
						IgnoreFailure: false, //THIS IS INHERITED
					},
					parlaytypes.Action{
						ActionType:  "command",
						Command:     "apt-get install curl apt-transport-https gnupg-agent ca-certificates software-properties-common ethtool socat ebtables conntrack libnetfilter-conntrack3 -y",
						Name:        "Cluster-API provisioning [Ubuntu package installation]",
						CommandSudo: "root",
					},
					parlaytypes.Action{
						ActionType:     "command",
						Command:        "tee /etc/apt/sources.list.d/docker.list",
						CommandPipeCmd: "echo \"deb https://download.docker.com/linux/ubuntu xenial stable\"",
						Name:           "Cluster-API provisioning [set Docker Repository]",
						CommandSudo:    "root",
					},
					parlaytypes.Action{
						ActionType:     "command",
						Command:        "tee /etc/apt/sources.list.d/kubernetes.list",
						CommandPipeCmd: "echo \"deb https://apt.kubernetes.io/ kubernetes-xenial main\"",
						Name:           "Cluster-API provisioning [set Kubernetes Repository]",
						CommandSudo:    "root",
					},
					parlaytypes.Action{
						ActionType:  "command",
						Command:     "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -",
						Name:        "Cluster-API provisioning [add Docker GPG Key]",
						CommandSudo: "root",
					},
					parlaytypes.Action{
						ActionType:  "command",
						Command:     "curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -",
						Name:        "Cluster-API provisioning [add Kubernetes GPG Key]",
						CommandSudo: "root",
					},
					parlaytypes.Action{
						ActionType:    "command",
						Command:       "sudo apt-get update",
						Name:          "Cluster-API provisioning [Ubuntu package update]",
						CommandSudo:   "root",
						IgnoreFailure: true,
					},
					parlaytypes.Action{
						ActionType:  "command",
						Command:     fmt.Sprintf("apt-get install -y docker-ce=%s kubelet=%s-00 kubeadm=%s-00 kubectl=%s-00 kubernetes-cni cri-tools", dockerVersion, kubeVersionFix, kubeVersionFix, kubeVersionFix),
						Name:        fmt.Sprintf("Cluster-API provisioning [install Kubernetes (%s) packages]", kubeVersion),
						CommandSudo: "root",
					},
					parlaytypes.Action{
						ActionType:  "command",
						Command:     "systemctl enable kubelet.service",
						Name:        "Cluster-API provisioning [enable Kubernetes Kubelet]",
						CommandSudo: "root",
					},
				},
			},
		},
	}
}

// ActionsControlPlane will add the additional deployment actions for building the deployment plane for Kubernetes
func (c *Client) ActionsControlPlane(kubeversion, cidr string) error {
	if c.deploymentMap == nil {
		return fmt.Errorf("The Kubernetes deployment couldn't be found, can't apply Control plane creation commands")
	}
	// Generate the control plane actions
	cp := []parlaytypes.Action{
		parlaytypes.Action{
			ActionType:  "command",
			Command:     fmt.Sprintf("kubeadm init --kubernetes-version \"%s\" --pod-network-cidr=%s", kubeversion, cidr),
			Name:        fmt.Sprintf("Cluster-API provisioning [Initialise Kubernetes %s Cluster]", kubeversion),
			CommandSudo: "root",
		},
		parlaytypes.Action{
			ActionType:  "command",
			Command:     "rm -rf ~/.kube ; mkdir -p ~/.kube ; sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config ; sudo chown $(id -u):$(id -g) $HOME/.kube/config",
			Name:        "Cluster-API provisioning [Set kubeconfig]",
			CommandSudo: "root",
		},
		parlaytypes.Action{
			ActionType:       "command",
			Command:          "kubeadm token create --print-join-command 2>/dev/null",
			Name:             "Generate a join token for workers",
			CommandSaveAsKey: "joinToken",
			CommandSudo:      "root",
		},
	}
	// Add to the deployment actions
	c.deploymentMap.Deployments[0].Actions = append(c.deploymentMap.Deployments[0].Actions, cp...)
	return nil
}

// ActionsWorker will add the additional deployment actions for adding a worker to an existing cluster
func (c *Client) ActionsWorker() error {
	if c.deploymentMap == nil {
		return fmt.Errorf("The Kubernetes deployment couldn't be found, can't apply Control plane creation commands")
	}
	// Generate the worker actions
	wrkr := []parlaytypes.Action{
		parlaytypes.Action{
			ActionType:  "command",
			KeyName:     "joinToken",
			Name:        "Join Worker to cluster",
			CommandSudo: "root",
		},
	}
	// Add to the deployment actions
	c.deploymentMap.Deployments[0].Actions = append(c.deploymentMap.Deployments[0].Actions, wrkr...)
	return nil
}

// TODO - will be needed if a worker needs a token after the main one has expired
func createKubeToken() []parlaytypes.Action {
	return []parlaytypes.Action{
		parlaytypes.Action{
			ActionType:       "command",
			Command:          "kubeadm token create --print-join-command 2>/dev/null",
			Name:             "Generate a join token for workers",
			CommandSaveAsKey: "joinToken",
			CommandSudo:      "root",
		},
	}
}
