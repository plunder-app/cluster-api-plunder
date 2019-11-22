/*
Copyright 2019 The Kubernetes Authors.

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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/plunder-app/cluster-api-provider-plunder/api/v1alpha1"
	"github.com/plunder-app/plunder/pkg/apiserver"
	"github.com/plunder-app/plunder/pkg/parlay/parlaytypes"
	"github.com/plunder-app/plunder/pkg/services"
)

// PlunderMachineReconciler reconciles a PlunderMachine object
type PlunderMachineReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=plundermachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=plundermachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events;secrets,verbs=get;list;watch;create;update;patch

// Reconcile - This is called when a resource of plunderMachine is created/modified/delted
func (r *PlunderMachineReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx := context.Background()
	log := r.Log.WithValues("plundermachine", req.NamespacedName)

	// your Plunder Machine logic begins here

	// Fetch the inceptionmachine instance.
	plunderMachine := &infrav1.PlunderMachine{}

	err := r.Get(ctx, req.NamespacedName, plunderMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log = log.WithName(plunderMachine.APIVersion)

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, plunderMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}
	log = log.WithName(fmt.Sprintf("machine=%s", machine.Name))

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	log = log.WithName(fmt.Sprintf("cluster=%s", cluster.Name))

	// Fetch the Plunder Cluster
	plunderCluster := &infrav1.PlunderCluster{}
	plunderClusterName := types.NamespacedName{
		Namespace: plunderMachine.Namespace, // get the name from the machine
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, plunderClusterName, plunderCluster); err != nil {
		log.Info("The Plunder Cluster is not available yet")
		return ctrl.Result{}, nil
	}

	log = log.WithName(fmt.Sprintf("plunderCluster=%s", plunderCluster.Name))

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(plunderMachine, r)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to Patch the PlunderMachine object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, plunderMachine); err != nil {
			log.Error(err, "failed to patch PlunderMachine")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// Handle deleted clusters
	if !plunderMachine.DeletionTimestamp.IsZero() {
		return r.reconcileMachineDelete(log, machine, plunderMachine, cluster, plunderCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileMachine(log, machine, plunderMachine, cluster, plunderCluster)
}

// SetupWithManager - will add the managment of resources of type PlunderMachine
func (r *PlunderMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.PlunderMachine{}).
		Complete(r)
}

func (r *PlunderMachineReconciler) reconcileMachine(log logr.Logger, machine *clusterv1.Machine, plunderMachine *infrav1.PlunderMachine, cluster *clusterv1.Cluster, plunderCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {
	log.Info("Reconciling Machine")
	// If the DockerMachine doesn't have finalizer, add it.
	if !util.Contains(plunderMachine.Finalizers, infrav1.MachineFinalizer) {
		plunderMachine.Finalizers = append(plunderMachine.Finalizers, infrav1.MachineFinalizer)
	}

	// if the machine is already provisioned, return
	if plunderMachine.Spec.ProviderID != nil {
		plunderMachine.Status.Ready = true

		return ctrl.Result{}, nil
	}

	// Make sure bootstrap data is available and populated, may be needed in the future (bootstrap is included in machine Controlelr)
	if machine.Spec.Bootstrap.Data == nil {
		log.Info("The Plunder Provider currently doesn't require bootstrap data")
	}

	var installMAC string

	// This next step will get all unleased servers from plunder

	// Find a machine for provisioning
	u, c, err := apiserver.BuildEnvironmentFromConfig("plunderclient.yaml", "")
	if err != nil {
		return ctrl.Result{}, err
	}
	// ep, resp := apiserver.FindFunctionEndpoint(u, c, "dhcp", http.MethodGet)
	// if resp.Error != "" {
	// 	return ctrl.Result{}, fmt.Errorf(resp.Error)

	// }

	// u.Path = path.Join(u.Path, ep.Path+"/unleased")

	// response, err := apiserver.ParsePlunderGet(u, c)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }
	// // If an error has been returned then handle the error gracefully and terminate
	// if response.FriendlyError != "" || response.Error != "" {
	// 	return ctrl.Result{}, err
	// }
	// var unleased []services.Lease

	// err = json.Unmarshal(response.Payload, &unleased)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// // Iterate through all known addresses and find a free one that looks "recent"
	// for i := range unleased {
	// 	if time.Since(unleased[i].Expiry).Minutes() < 10 {
	// 		installMAC = unleased[i].MAC
	// 	}
	// }

	installMAC, err = findUnleasedServer(u, c)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Hopefully we found an unleased server!
	if installMAC == "" {
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeWarning, "No Hardware found", "Plunder has no available hardware to provision")
		return ctrl.Result{}, fmt.Errorf("No free hardware for provisioning")
	}

	log.Info(fmt.Sprintf("Found Hardware %s", installMAC))

	d := services.DeploymentConfig{
		ConfigName: "preseed",
		MAC:        installMAC,
		ConfigHost: services.HostConfig{},
	}

	if plunderMachine.Spec.IPAddress != nil {
		d.ConfigHost.IPAddress = *plunderMachine.Spec.IPAddress
	} else {
		return ctrl.Result{}, fmt.Errorf("An IP Adress is required to provision at this time")
		// TODO (EPIC) implement IPAM
	}

	//Check the role of the machine
	if util.IsControlPlaneMachine(machine) {
		log.Info(fmt.Sprintf("Provisioning Control plane node %s", machine.Name))
		d.ConfigHost.ServerName = fmt.Sprintf("controlplane-%s", StringWithCharset(5, charset))

	} else {
		log.Info(fmt.Sprintf("Provisioning Worker node %s", machine.Name))
		d.ConfigHost.ServerName = fmt.Sprintf("worker-%s", StringWithCharset(5, charset))
	}

	// Marshall the deployment into data
	b, err := json.Marshal(d)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = createDeployment(u, c, b)
	if err != nil {
		return ctrl.Result{}, err
	}
	// ep, resp := apiserver.FindFunctionEndpoint(u, c, "deployment", http.MethodPost)
	// if resp.Error != "" {
	// 	return ctrl.Result{}, fmt.Errorf(resp.Error)

	// }

	// u.Path = ep.Path

	// response, err := apiserver.ParsePlunderPost(u, c, b)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }
	// // If an error has been returned then handle the error gracefully and terminate
	// if response.FriendlyError != "" || response.Error != "" {
	// 	return ctrl.Result{}, fmt.Errorf(resp.Error)

	// }

	newMap := uptimeCommand(d.ConfigHost.IPAddress)

	// Marshall the parlay submission (runs the uptime command)
	b, err = json.Marshal(newMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create the string that will be used to get the logs
	dashAddress := strings.Replace(d.ConfigHost.IPAddress, ".", "-", -1)

	// TEST Provision CODE
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", "Plunder has begun provisioning the Operating System")

	provisioningResult, err := parlayHelper(u, c, &dashAddress, b)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", provisioningResult)
	log.Info(provisioningResult)

	// for {
	// 	// Set Parlay API path and POST
	// 	ep, resp = apiserver.FindFunctionEndpoint(u, c, "parlay", http.MethodPost)
	// 	if resp.Error != "" {
	// 		return ctrl.Result{}, fmt.Errorf(resp.Error)

	// 	}
	// 	u.Path = ep.Path

	// 	response, err := apiserver.ParsePlunderPost(u, c, b)
	// 	if err != nil {
	// 		return ctrl.Result{}, err
	// 	}

	// 	// If an error has been returned then handle the error gracefully and terminate
	// 	if response.FriendlyError != "" || response.Error != "" {
	// 		return ctrl.Result{}, fmt.Errorf(resp.Error)

	// 	}

	// 	// Sleep for five seconds
	// 	time.Sleep(5 * time.Second)

	// 	// Set the parlay API get logs path and GET
	// 	ep, resp = apiserver.FindFunctionEndpoint(u, c, "parlayLog", http.MethodGet)
	// 	if resp.Error != "" {
	// 		return ctrl.Result{}, fmt.Errorf(resp.Error)

	// 	}
	// 	u.Path = ep.Path + "/" + dashAddress

	// 	response, err = apiserver.ParsePlunderGet(u, c)
	// 	if err != nil {
	// 		return ctrl.Result{}, err
	// 	}
	// 	// If an error has been returned then handle the error gracefully and terminate
	// 	if response.FriendlyError != "" || response.Error != "" {
	// 		return ctrl.Result{}, fmt.Errorf(resp.Error)

	// 	}

	// 	var logs plunderlogging.JSONLog

	// 	err = json.Unmarshal(response.Payload, &logs)
	// 	if err != nil {
	// 		return ctrl.Result{}, err
	// 	}

	// 	if logs.State == "Completed" {
	// 		provisioningResult := fmt.Sprintf("Host has been succesfully provisioned OS in %s Seconds\n", time.Since(t).Round(time.Second))
	// 		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", provisioningResult)

	// 		log.Info(provisioningResult)
	// 		break
	// 	}
	//}

	var kubeMap parlaytypes.TreasureMap

	// Check if a version of Docker is defined, if not fall back to the default
	// if plunderMachine.Spec.DockerVersion != nil {
	// 	kubeMap = kubeCreateHostCommand(*plunderMachine.Spec.IPAddress, *machine.Spec.Version, *plunderMachine.Spec.DockerVersion)
	// } else {
	// 	kubeMap = kubeCreateHostCommand(*plunderMachine.Spec.IPAddress, *machine.Spec.Version, infrav1.DockerVersionDefault)
	// }

	// // Check if a version of Docker is defined, if not fall back to the default
	// if machine.Spec.Version != nil {
	// 	kubeMap = kubeCreateHostCommand(*plunderMachine.Spec.IPAddress, *machine.Spec.Version, *machine.Spec.Version)
	// } else {
	// 	kubeMap = kubeCreateHostCommand(*plunderMachine.Spec.IPAddress, *machine.Spec.Version, infrav1.DockerVersionDefault)
	// }
	if plunderMachine.Spec.DockerVersion == nil {
		ver := infrav1.DockerVersionDefault
		plunderMachine.Spec.DockerVersion = &ver
	}

	if machine.Spec.Version == nil {
		ver := infrav1.KubernetesVersionDefault
		machine.Spec.Version = &ver
	}

	kubeMap = kubeCreateHostCommand(*plunderMachine.Spec.IPAddress, *machine.Spec.Version, *plunderMachine.Spec.DockerVersion)

	if util.IsControlPlaneMachine(machine) {
		// Add the kubeadm steps for a control plane
		kubeMap.Deployments[0].Actions = append(kubeMap.Deployments[0].Actions, kubeadmActions(*machine.Spec.Version, cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0])...)
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderInstall", "Kubernetes Control Plane installation has begun")
		log.Info("Kubernetes Control Plane installation has begun")
	} else {
		// Add the kubeadm steps for a worker machine
		kubeMap.Deployments[0].Actions = append(kubeMap.Deployments[0].Actions, useKubeToken()...)
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderInstall", "Kubernetes worker installation has begun")
		log.Info("Kubernetes worker installation has begun")
	}

	// Marshall the parlay submission (runs the uptime command)
	b, err = json.Marshal(kubeMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	provisioningResult, err = parlayInstaller(u, c, &dashAddress, b)
	if err != nil {
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderInstall", fmt.Sprintf("Kubernetes Package installation has failed [%s]", err.Error()))
		return ctrl.Result{}, err
	}

	// Report the results of the installation
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderInstall", provisioningResult)
	log.Info(provisioningResult)

	// TODO - Attempt to create the machine

	// // if the machine is a control plane added, update the load balancer configuration
	// if util.IsControlPlaneMachine(machine) {}

	// DEPLOY THE MACHINE
	//clusterDeploy(nil)

	providerID := fmt.Sprintf("plunder://%s", installMAC)

	plunderMachine.Spec.ProviderID = &providerID
	// Mark the inceptionMachine ready
	plunderMachine.Status.Ready = true
	// Set the object status
	plunderMachine.Status.MACAddress = installMAC
	plunderMachine.Status.IPAdress = d.ConfigHost.IPAddress

	return ctrl.Result{}, nil

}

func (r *PlunderMachineReconciler) reconcileMachineDelete(logger logr.Logger, machine *clusterv1.Machine, plunderMachine *infrav1.PlunderMachine, cluster *clusterv1.Cluster, plunderCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {
	logger.Info(fmt.Sprintf("Deleting Machine %s", plunderMachine.Name))
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderDelete", "Plunder has begun removing the host")

	u, c, err := apiserver.BuildEnvironmentFromConfig("plunderclient.yaml", "")
	if err != nil {
		return ctrl.Result{}, err
	}

	// Build the parlay map for running the OS destroy commands
	destroyMap := destroyCommand(plunderMachine.Status.IPAdress)

	// Marshall the parlay map
	b, err := json.Marshal(destroyMap)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Set Parlay API path and POST the commands to the running machine
	ep, resp := apiserver.FindFunctionEndpoint(u, c, "parlay", http.MethodPost)
	if resp.Error != "" || resp.FriendlyError != "" {
		return ctrl.Result{}, fmt.Errorf(resp.FriendlyError)
	}

	u.Path = ep.Path
	response, err := apiserver.ParsePlunderPost(u, c, b)
	if err != nil {

		return ctrl.Result{}, fmt.Errorf(response.Error)
	}

	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		// TODO - if this error occurs it's because the machine doesn't exist
		plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
		logger.Info(fmt.Sprintf("Removing Machine [%s] from config, it may need removing manually", plunderMachine.Name))
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderDelete", "Machine removed. Plunder couldn't succesfully remove the physical host, it may need removing manually")

		return ctrl.Result{}, fmt.Errorf(resp.FriendlyError)

	}

	// Remove the server via it's address from the list of deployements in Plunder
	ep, resp = apiserver.FindFunctionEndpoint(u, c, "deploymentAddress", http.MethodDelete)
	if resp.Error != "" {
		return ctrl.Result{}, fmt.Errorf(resp.Error)

	}
	u.Path = ep.Path + "/" + strings.Replace(plunderMachine.Status.IPAdress, ".", "-", -1)
	response, err = apiserver.ParsePlunderDelete(u, c)
	if err != nil {
		return ctrl.Result{}, err
	}

	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return ctrl.Result{}, fmt.Errorf(resp.Error)

	}

	// Machine is deleted so remove the finalizer.
	plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderDelete", "Machine removed succesfully")
	return ctrl.Result{}, nil

}
