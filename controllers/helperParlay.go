package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/plunder-app/plunder/pkg/apiserver"
	"github.com/plunder-app/plunder/pkg/parlay/parlaytypes"
	"github.com/plunder-app/plunder/pkg/plunderlogging"
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

func kubeCreateHostCommand(host, kubeVersion, dockerVersion string) parlaytypes.TreasureMap {

	// The Kubernetes standard is to define versions such as v1.x.x, however the OS packages are 1.x.x (missing the "v")
	kubeVersionFix := strings.Replace(kubeVersion, "v", "", -1)
	return parlaytypes.TreasureMap{
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

func kubeadmActions(kubeversion, cidr string) []parlaytypes.Action {
	return []parlaytypes.Action{
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
}

func useKubeToken() []parlaytypes.Action {
	return []parlaytypes.Action{
		parlaytypes.Action{
			ActionType:  "command",
			KeyName:     "joinToken",
			Name:        "Join Worker to cluster",
			CommandSudo: "root",
		},
	}
}

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

// TODO - needs a watch function

func parlayHelper(u *url.URL, c *http.Client, uri *string, b []byte) (result string, err error) {
	// Get the time
	t := time.Now()

	for {
		// Set Parlay API path and POST
		ep, resp := apiserver.FindFunctionEndpoint(u, c, "parlay", http.MethodPost)
		if resp.Error != "" {
			return result, fmt.Errorf(resp.Error)

		}
		u.Path = ep.Path

		response, err := apiserver.ParsePlunderPost(u, c, b)
		if err != nil {
			return result, err
		}

		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			return result, fmt.Errorf(resp.Error)

		}

		// Sleep for five seconds
		time.Sleep(5 * time.Second)

		// Set the parlay API get logs path and GET
		ep, resp = apiserver.FindFunctionEndpoint(u, c, "parlayLog", http.MethodGet)
		if resp.Error != "" {
			return result, fmt.Errorf(resp.Error)

		}

		if uri != nil {
			u.Path = ep.Path + "/" + *uri
		}

		response, err = apiserver.ParsePlunderGet(u, c)
		if err != nil {
			return result, err
		}
		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			return result, fmt.Errorf(resp.Error)

		}

		var logs plunderlogging.JSONLog

		err = json.Unmarshal(response.Payload, &logs)
		if err != nil {
			return result, err
		}

		if logs.State == "Completed" {
			result = fmt.Sprintf("Task has been succesfully completed in %s Seconds\n", time.Since(t).Round(time.Second))

			break
		}
	}
	return
}

func parlayInstaller(u *url.URL, c *http.Client, uri *string, b []byte) (result string, err error) {
	// Get the time
	t := time.Now()
	// Set Parlay API path and POST
	ep, resp := apiserver.FindFunctionEndpoint(u, c, "parlay", http.MethodPost)
	if resp.Error != "" {
		return result, fmt.Errorf(resp.Error)

	}
	u.Path = ep.Path

	response, err := apiserver.ParsePlunderPost(u, c, b)
	if err != nil {
		return result, err
	}

	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return result, fmt.Errorf(resp.Error)

	}
	for {
		// Sleep for five seconds
		time.Sleep(5 * time.Second)

		// Set the parlay API get logs path and GET
		ep, resp = apiserver.FindFunctionEndpoint(u, c, "parlayLog", http.MethodGet)
		if resp.Error != "" {
			return result, fmt.Errorf(resp.Error)

		}

		if uri != nil {
			u.Path = ep.Path + "/" + *uri
		}

		response, err = apiserver.ParsePlunderGet(u, c)
		if err != nil {
			return result, err
		}
		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			return result, fmt.Errorf(resp.Error)

		}

		var logs plunderlogging.JSONLog

		err = json.Unmarshal(response.Payload, &logs)
		if err != nil {
			return result, err
		}

		if logs.State == "Completed" {
			// Report completion message
			result = fmt.Sprintf("Task has been succesfully completed in %s Seconds\n", time.Since(t).Round(time.Second))
			break
		} else if logs.State == "Failed" {
			// Report error message
			result = fmt.Sprintf("Task has been failed after in %s Seconds\n", time.Since(t).Round(time.Second))
			break
		}
	}
	return
}
