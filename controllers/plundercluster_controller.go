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
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/plunder-app/cluster-api-plunder/api/v1alpha1"
)

// PlunderClusterReconciler reconciles a PlunderCluster object
type PlunderClusterReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=plunderclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=plunderclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *PlunderClusterReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx := context.Background()
	log := r.Log.WithValues("plundercluster", req.NamespacedName)

	// Plunder Cluster Logic begins here

	// Fetch the PlunderCluster instance
	plunderCluster := &infrav1.PlunderCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, plunderCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Cluster-API Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, plunderCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Waiting for Cluster Controller to set OwnerRef on Plunder Cluster")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(plunderCluster, r)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to Patch the plunderCluster object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, plunderCluster); err != nil {
			log.Error(err, "failed to patch infrav1Cluster")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// Handle deleted clusters
	if !plunderCluster.DeletionTimestamp.IsZero() {
		return r.reconcileClusterDelete(log, plunderCluster)
	}

	return r.reconcileCluster(log, cluster, plunderCluster)
}

func (r *PlunderClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.PlunderCluster{}).
		Complete(r)
}

func (r *PlunderClusterReconciler) reconcileCluster(logger logr.Logger, cluster *clusterv1.Cluster, plunderCluster *infrav1.PlunderCluster) (ctrl.Result, error) {
	logger.Info("Reconciling Cluster")

	if !util.Contains(plunderCluster.Finalizers, infrav1.ClusterFinalizer) {
		plunderCluster.Finalizers = append(plunderCluster.Finalizers, infrav1.ClusterFinalizer)
	}

	// With plunder at the moment there is nothing that can be done throught he server component

	// plunderCluster.Status.APIEndpoints = []infrav1.APIEndpoint{
	// 	{
	// 		Host: "192.168.0.1",
	// 		Port: 6443,
	// 	},
	// }
	// Deploy a new cluster
	//clusterDeploy(plunderCluster)

	plunderCluster.Status.Ready = true
	return ctrl.Result{}, nil

}

func (r *PlunderClusterReconciler) reconcileClusterDelete(logger logr.Logger, plunderCluster *infrav1.PlunderCluster) (ctrl.Result, error) {
	logger.Info("Deleting Cluster")
	plunderCluster.Finalizers = util.Filter(plunderCluster.Finalizers, infrav1.ClusterFinalizer)

	return ctrl.Result{}, nil
}

func clusterDeploy(i *infrav1.PlunderCluster) {
	// This will simulate the deployment
	time.Sleep(time.Duration(5) * time.Second)

}
