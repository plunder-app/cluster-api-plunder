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

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/plunder-app/cluster-api-plunder/api/v1alpha1"
	"github.com/plunder-app/cluster-api-plunder/pkg/plunder"
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

func (r *PlunderMachineReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx := context.Background()
	log := r.Log.WithValues("plundermachine", req.NamespacedName)

	// your Plunder Machine logic begins here

	// Attempt to speak with the provisioning (plunder) server
	// TODO - may need moving lower
	client, err := plunder.NewClient()
	if err != nil {
		return ctrl.Result{}, err
	}

	// We can speak with the plunder server, we can now evaluate the changes

	// Fetch the plunderMachine instance.
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
		return r.reconcileMachineDelete(client, log, machine, plunderMachine, cluster, plunderCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileMachine(client, log, machine, plunderMachine, cluster, plunderCluster)
}

func (r *PlunderMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.PlunderMachine{}).
		Complete(r)
}

func (r *PlunderMachineReconciler) reconcileMachine(client *plunder.Client, log logr.Logger, machine *clusterv1.Machine, plunderMachine *infrav1.PlunderMachine, cluster *clusterv1.Cluster, plunderCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {
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

	// Make sure bootstrap data is available and populated.
	if machine.Spec.Bootstrap.Data == nil {
		log.Info("The Plunder Provider currently doesn't require bootstrap data")
	}

	if plunderMachine.Spec.MACAddress == nil {
		installMAC, err := client.FindMachine()
		if err != nil {
			r.Recorder.Eventf(plunderMachine, corev1.EventTypeWarning, "No Hardware found", "Plunder has no available hardware to provision")
			return ctrl.Result{}, err
		}
		plunderMachine.Spec.MACAddress = &installMAC
	}
	log.Info(fmt.Sprintf("Found Hardware %s", *plunderMachine.Spec.MACAddress))

	// 	//Check the role of the machine
	if util.IsControlPlaneMachine(machine) {
		log.Info(fmt.Sprintf("Provisioning Control plane node %s", machine.Name))
	} else {
		log.Info(fmt.Sprintf("Provisioning Worker node %s", machine.Name))
	}

	err := client.ProvisionMachine(plunderMachine.Name, *plunderMachine.Spec.MACAddress, *plunderMachine.Spec.IPAdress, *plunderMachine.Spec.DeploymentType)
	if err != nil {
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeWarning, "PlunderProvision", "Plunder failed to deploy")
		return ctrl.Result{}, err
	}
	r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", "Plunder has begun provisioning the Operating System")

	result, err := client.ProvisionMachineWait(*plunderMachine.Spec.IPAdress)

	// 	provisioningResult := fmt.Sprintf("Host has been succesfully provisioned OS in %s Seconds\n", time.Since(t).Round(time.Second))

	if result != nil {
		r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", *result)

		log.Info(*result)
	}
	providerID := fmt.Sprintf("plunder://%s", *plunderMachine.Spec.MACAddress)

	plunderMachine.Spec.ProviderID = &providerID
	// Mark the plunderMachine ready
	plunderMachine.Status.Ready = true
	// Set the object status
	plunderMachine.Status.MACAddress = *plunderMachine.Spec.MACAddress
	plunderMachine.Status.IPAdress = *plunderMachine.Spec.IPAdress

	return ctrl.Result{}, nil
}

func (r *PlunderMachineReconciler) reconcileMachineDelete(client *plunder.Client,
	logger logr.Logger,
	machine *clusterv1.Machine,
	plunderMachine *infrav1.PlunderMachine,
	cluster *clusterv1.Cluster,
	plunderCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {

	logger.Info(fmt.Sprintf("Deleting Machine %s", plunderMachine.Name))

	if plunderMachine.Spec.MACAddress == nil || plunderMachine.Spec.IPAdress == nil {
		logger.Info(fmt.Sprintf("Plunder failed to remove machine [%s] as it has no hardware address, it may need removing manually", plunderMachine.Name))
		plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
		return ctrl.Result{}, nil
	}

	err := client.DeleteMachine(*plunderMachine.Spec.IPAdress)
	// 	// If an error has been returned then handle the error gracefully and terminate
	if err != nil {
		// TODO - if this error occurs it's because the machine doesn't exist
		plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
		logger.Info(fmt.Sprintf("Plunder failed to remove machine [%s], it may need removing manually", plunderMachine.Name))
		return ctrl.Result{}, err
	}

	// Machine is deleted so remove the finalizer.
	plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
	return ctrl.Result{}, nil

}
