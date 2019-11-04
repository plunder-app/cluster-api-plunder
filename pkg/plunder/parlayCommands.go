package plunder

import "github.com/plunder-app/plunder/pkg/parlay/parlaytypes"

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
