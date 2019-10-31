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

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/plunder-app/cluster-api-provider-plunder/api/v1alpha1"
)

// PlunderMachineReconciler reconciles a PlunderMachine object
type PlunderMachineReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=plundermachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=plundermachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events;secrets,verbs=get;list;watch;create;update;patch

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

func (r *PlunderMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.PlunderMachine{}).
		Complete(r)
}

func (r *PlunderMachineReconciler) reconcileMachine(log logr.Logger, machine *clusterv1.Machine, inceptionMachine *infrav1.PlunderMachine, cluster *clusterv1.Cluster, inceptionCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {
	log.Info("Reconciling Machine")
	// If the DockerMachine doesn't have finalizer, add it.
	if !util.Contains(inceptionMachine.Finalizers, infrav1.MachineFinalizer) {
		inceptionMachine.Finalizers = append(inceptionMachine.Finalizers, infrav1.MachineFinalizer)
	}

	// Immeditaly give it the details it needs
	//	providerID := "inception:////inception"

	// if the machine is already provisioned, return
	if inceptionMachine.Spec.ProviderID != nil {
		inceptionMachine.Status.Ready = true

		return ctrl.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machine.Spec.Bootstrap.Data == nil {

		log.Info("The Plunder Provider currently doesn't require bootstrap data")
		//return ctrl.Result{}, nil
	}

	//Check the role of the machine
	//role := constants.WorkerNodeRoleValue
	if util.IsControlPlaneMachine(machine) {
		log.Info(fmt.Sprintf("Provisioning %s", machine.Name))
		//role = constants.ControlPlaneNodeRoleValue
	}

	// TODO - Attempt to create the machine

	// // if the machine is a control plane added, update the load balancer configuration
	// if util.IsControlPlaneMachine(machine) {}

	// DEPLOY THE MACHINE
	clusterDeploy(nil)

	providerID := fmt.Sprintf("inception:////%s", "test")

	inceptionMachine.Spec.ProviderID = &providerID
	// Mark the inceptionMachine ready
	inceptionMachine.Status.Ready = true

	return ctrl.Result{}, nil

}

func (r *PlunderMachineReconciler) reconcileMachineDelete(logger logr.Logger, machine *clusterv1.Machine, inceptionMachine *infrav1.PlunderMachine, cluster *clusterv1.Cluster, inceptionCluster *infrav1.PlunderCluster) (_ ctrl.Result, reterr error) {
	logger.Info("Deleting Machine")
	// Machine is deleted so remove the finalizer.
	inceptionMachine.Finalizers = util.Filter(inceptionMachine.Finalizers, infrav1.MachineFinalizer)
	return ctrl.Result{}, nil

}
