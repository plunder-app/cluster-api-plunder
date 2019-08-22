package main

import (
	"github.com/plunder-app/plunder/pkg/parlay/parlaytypes"
)

func uptimeCommand(host string) parlaytypes.TreasureMap {
	return parlaytypes.TreasureMap{
		Deployments: []parlaytypes.Deployment{
			parlaytypes.Deployment{
				Name:     "Cluster-API provisioning",
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

func kubeCreateHostCommand(host string) parlaytypes.TreasureMap {
	return parlaytypes.TreasureMap{
		Deployments: []parlaytypes.Deployment{
			parlaytypes.Deployment{
				Name:     "Cluster-API provisioning",
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
						IgnoreFailure: true,
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
						Command:     "apt-get install -y docker-ce=18.06.1~ce~3-0~ubuntu kubelet=1.14.1-00 kubeadm=1.14.1-00 kubectl=1.14.1-00 kubernetes-cni cri-tools",
						Name:        "Cluster-API provisioning [install Kubernetes (1.14.1) packages]",
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

func kubeadmActions() []parlaytypes.Action {
	return []parlaytypes.Action{
		parlaytypes.Action{
			ActionType:  "command",
			Command:     "kubeadm init --kubernetes-version \"v1.14.1\" --pod-network-cidr=172.16.0.0/16",
			Name:        "Cluster-API provisioning [Initialise Kubernetes v1.14.1 Cluster]",
			CommandSudo: "root",
		},
		parlaytypes.Action{
			ActionType:  "command",
			Command:     "rm -rf ~/.kube ; mkdir -p ~/.kube ; sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config ; sudo chown $(id -u):$(id -g) $HOME/.kube/config",
			Name:        "Cluster-API provisioning [Set kubeconfig]",
			CommandSudo: "root",
		},
	}
}
