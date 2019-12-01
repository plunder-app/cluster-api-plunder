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
	"fmt"

	"github.com/plunder-app/cluster-api-plunder/pkg/plunder"

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

	infrav1 "github.com/plunder-app/cluster-api-plunder/api/v1alpha1"
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

	// Generate a new Plunder client
	c, err := plunder.NewClient()
	if err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the inceptionmachine instance.
	plunderMachine := &infrav1.PlunderMachine{}

	err = r.Get(ctx, req.NamespacedName, plunderMachine)
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
		return r.reconcileMachineDelete(c, log, machine, plunderMachine, cluster, plunderCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileMachine(c, log, machine, plunderMachine, cluster, plunderCluster)
}

// SetupWithManager - will add the managment of resources of type PlunderMachine
func (r *PlunderMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.PlunderMachine{}).
		Complete(r)
}

func (r *PlunderMachineReconciler) reconcileMachine(c *plunder.Client, log logr.Logger, machine *clusterv1.Machine, plunderMachine *infrav1.PlunderMachine, cluster *clusterv1.Cluster, plunderCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {
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

	installMAC, err := c.FindMachine()
	if err != nil {
		return ctrl.Result{}, err
	}

	// Hopefully we found an unleased server!
	if installMAC == "" {
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeWarning, "No Hardware found", "Plunder has no available hardware to provision")
		return ctrl.Result{}, fmt.Errorf("No free hardware for provisioning")
	}

	log.Info(fmt.Sprintf("Found Hardware %s", installMAC))

	// If the deployment type is left blank then we default to the provider default
	if plunderMachine.Spec.DeploymentType == nil {
		deploymentType := infrav1.DeploymentDefault
		plunderMachine.Spec.DeploymentType = &deploymentType
	}

	// If the IP address is blank we (error for now)
	if plunderMachine.Spec.IPAddress == nil {
		return ctrl.Result{}, fmt.Errorf("An IP Adress is required to provision at this time")
		// TODO (EPIC) implement IPAM
	}

	//Check the role of the machine
	if util.IsControlPlaneMachine(machine) {
		log.Info(fmt.Sprintf("Provisioning Control plane node %s", machine.Name))
		plunderMachine.Status.MachineName = fmt.Sprintf("%s-%s", machine.Name, StringWithCharset(5, charset))

	} else {
		log.Info(fmt.Sprintf("Provisioning Worker node %s", machine.Name))
		plunderMachine.Status.MachineName = fmt.Sprintf("%s-%s", machine.Name, StringWithCharset(5, charset))
	}

	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", "Plunder has begun provisioning the Operating System")

	err = c.ProvisionMachine(plunderMachine.Status.MachineName, installMAC, *plunderMachine.Spec.IPAddress, *plunderMachine.Spec.DeploymentType)
	if err != nil {
		return ctrl.Result{}, err
	}

	provisioningResult, err := c.ProvisionMachineWait(*plunderMachine.Spec.IPAddress)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", *provisioningResult)
	log.Info(*provisioningResult)

	if plunderMachine.Spec.DockerVersion == nil {
		ver := infrav1.DockerVersionDefault
		plunderMachine.Spec.DockerVersion = &ver
	}

	if machine.Spec.Version == nil {
		ver := infrav1.KubernetesVersionDefault
		machine.Spec.Version = &ver
	}

	c.ActionsKubernetes(*plunderMachine.Spec.IPAddress, *machine.Spec.Version, *plunderMachine.Spec.DockerVersion)

	if util.IsControlPlaneMachine(machine) {
		// Add the kubeadm steps for a control plane
		err = c.ActionsControlPlane(*machine.Spec.Version, cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
		if err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderInstall", "Kubernetes Control Plane installation has begun")
		log.Info("Kubernetes Control Plane installation has begun")
	} else {
		// Add the kubeadm steps for a worker machine
		err = c.ActionsWorker()
		if err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderInstall", "Kubernetes worker installation has begun")
		log.Info("Kubernetes worker installation has begun")
	}

	provisioningResult, err = c.ProvisionKubernetes()
	if err != nil {
		return ctrl.Result{}, err
	}

	// Report the results of the installation
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderInstall", *provisioningResult)
	log.Info(*provisioningResult)

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
	plunderMachine.Status.IPAdress = *plunderMachine.Spec.IPAddress

	return ctrl.Result{}, nil

}

func (r *PlunderMachineReconciler) reconcileMachineDelete(c *plunder.Client, logger logr.Logger, machine *clusterv1.Machine, plunderMachine *infrav1.PlunderMachine, cluster *clusterv1.Cluster, plunderCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {
	logger.Info(fmt.Sprintf("Deleting Machine %s", plunderMachine.Name))
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderDelete", "Plunder has begun removing the host")
	err := c.DeleteMachine(plunderMachine.Status.IPAdress)
	if err != nil {

		plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
		logger.Info(fmt.Sprintf("Removing Machine [%s] from config, it may need removing manually", plunderMachine.Name))
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderDelete", "Machine removed. Plunder couldn't succesfully remove the physical host, it may need removing manually")

		return ctrl.Result{}, err

	}

	// Machine is deleted so remove the finalizer.
	plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderDelete", "Machine removed succesfully")
	return ctrl.Result{}, nil

}
