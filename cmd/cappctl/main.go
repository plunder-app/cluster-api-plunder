package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/plunder-app/plunder/pkg/apiserver"
	"github.com/plunder-app/plunder/pkg/plunderlogging"
	"github.com/plunder-app/plunder/pkg/services"
	"github.com/spf13/cobra"
)

var logLevel int

var managermentCluster struct {
	mac     string
	address string
}

func init() {
	initClusterCmd.Flags().StringVarP(&managermentCluster.mac, "mac", "m", "", "The Mac address of the node to use for provisioning")
	initClusterCmd.Flags().StringVarP(&managermentCluster.address, "address", "a", "", "The IP address to provision the management cluster with")
	cappctlCmd.PersistentFlags().IntVar(&logLevel, "logLevel", int(log.InfoLevel), "Set the logging level [0=panic, 3=warning, 5=debug]")

	cappctlCmd.AddCommand(initClusterCmd)
}

func main() {
	if err := cappctlCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var cappctlCmd = &cobra.Command{
	Use:   "cappctl",
	Short: "Cluster API Plunder control",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		return
	},
}

var initClusterCmd = &cobra.Command{
	Use:   "init-mgmt-cluster",
	Short: "Initialise Kubernetes Management Cluster",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.Level(logLevel))

		fmt.Println("Beginning the deployment of a new host")

		if managermentCluster.mac == "" {
			// Print a warning
			fmt.Printf("Will select an unleased server at random for management cluster in 5 seconds\n")
			// Wait to give the user time to cancel with ctrl+c
			time.Sleep(5 * time.Second)
			// Get a mac address
		}

		u, c, err := apiserver.BuildEnvironmentFromConfig("plunderclient.yaml", "")
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		d := services.DeploymentConfig{
			ConfigName: "preseed",
			MAC:        managermentCluster.mac,
			ConfigHost: services.HostConfig{
				IPAddress:  managermentCluster.address,
				ServerName: "Manager01",
			},
		}

		u.Path = apiserver.DeploymentAPIPath()
		b, err := json.Marshal(d)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
		response, err := apiserver.ParsePlunderPost(u, c, b)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			log.Debugln(response.Error)
			log.Fatalln(response.FriendlyError)
		}

		newMap := uptimeCommand(managermentCluster.address)

		// Marshall the parlay submission (runs the uptime command)
		b, err = json.Marshal(newMap)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		// Create the string that will be used to get the logs
		dashAddress := strings.Replace(managermentCluster.address, ".", "-", -1)

		// Get the time
		t := time.Now()

		for {
			// Set Parlay API path and POST
			u.Path = apiserver.ParlayAPIPath()
			response, err := apiserver.ParsePlunderPost(u, c, b)
			if err != nil {
				log.Fatalf("%s", err.Error())
			}

			// If an error has been returned then handle the error gracefully and terminate
			if response.FriendlyError != "" || response.Error != "" {
				log.Debugln(response.Error)
				log.Fatalln(response.FriendlyError)
			}

			// Sleep for five seconds
			time.Sleep(5 * time.Second)

			// Set the parlay API get logs path and GET
			u.Path = apiserver.ParlayAPIPath() + "/logs/" + dashAddress
			response, err = apiserver.ParsePlunderGet(u, c)
			if err != nil {
				log.Fatalf("%s", err.Error())
			}
			// If an error has been returned then handle the error gracefully and terminate
			if response.FriendlyError != "" || response.Error != "" {
				log.Debugln(response.Error)
				log.Fatalln(response.FriendlyError)
			}

			var logs plunderlogging.JSONLog

			err = json.Unmarshal(response.Payload, &logs)
			if err != nil {
				log.Fatalf("%s", err.Error())
			}

			if logs.State != "Completed" {
				fmt.Printf("\r\033[36mWaiting for Host to complete OS provisioning \033[m%.0f Seconds", time.Since(t).Seconds())
			} else {
				fmt.Printf("\r\033[32mHost has been succesfully provisioned OS in\033[m %s Seconds\n", time.Since(t).Round(time.Second))
				break
			}
		}

		fmt.Printf("This process can be exited with ctrl+c and monitored with pldrctl get logs %s -w 5\n", managermentCluster.address)

		// Begin the Kubernetes installation //
		fmt.Println("Beginning the installation and initialisation of Kubernetes")

		// Get the Kubernetes Installation commands
		kubeMap := kubeCreateHostCommand(managermentCluster.address)

		// Add the kubeadm steps
		kubeMap.Deployments[0].Actions = append(kubeMap.Deployments[0].Actions, kubeadmActions()...)

		// Marshall the parlay submission (runs the uptime command)
		b, err = json.Marshal(kubeMap)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
		// Set Parlay API path and POST
		u.Path = apiserver.ParlayAPIPath()
		response, err = apiserver.ParsePlunderPost(u, c, b)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			log.Debugln(response.Error)
			log.Fatalln(response.FriendlyError)
		}
		// Get the time
		t = time.Now()

		for {

			// Sleep for five seconds
			time.Sleep(5 * time.Second)

			// Set the parlay API get logs path and GET
			u.Path = apiserver.ParlayAPIPath() + "/logs/" + dashAddress
			response, err = apiserver.ParsePlunderGet(u, c)
			if err != nil {
				log.Fatalf("%s", err.Error())
			}
			// If an error has been returned then handle the error gracefully and terminate
			if response.FriendlyError != "" || response.Error != "" {
				log.Debugln(response.Error)
				log.Fatalln(response.FriendlyError)
			}

			var logs plunderlogging.JSONLog

			err = json.Unmarshal(response.Payload, &logs)
			if err != nil {
				log.Fatalf("%s", err.Error())
			}

			if logs.State != "Completed" {
				fmt.Printf("\r\033[36mWaiting for Kubernetes to complete installation \033[m%.0f Seconds", time.Since(t).Seconds())
			} else if logs.State == "Failed" {
				log.Fatalln("Kubernetes has failed to install")
			} else {
				fmt.Printf("\r\033[32mKubernetes has been succesfully installed on host %s in\033[m %s Seconds\n", managermentCluster.address, time.Since(t).Round(time.Second))
				break
			}
		}

		return
	},
}
