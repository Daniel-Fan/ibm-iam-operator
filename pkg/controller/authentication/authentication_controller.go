//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package authentication

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	operatorv1alpha1 "github.com/IBM/ibm-iam-operator/pkg/apis/operator/v1alpha1"
	pkgCommon "github.com/IBM/ibm-iam-operator/pkg/common"
	common "github.com/IBM/ibm-iam-operator/pkg/controller/common"
	certmgr "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_authentication")
var fullAccess int32 = 0777
var trueVar bool = true
var falseVar bool = false
var seconds60 int64 = 60
var partialAccess int32 = 420
var authServicePort int32 = 9443
var identityProviderPort int32 = 4300
var identityManagerPort int32 = 4500
var serviceAccountName string = "ibm-iam-operand-restricted"

var cpu10 = resource.NewMilliQuantity(10, resource.DecimalSI)            // 10m
var cpu50 = resource.NewMilliQuantity(50, resource.DecimalSI)            // 50m
var cpu100 = resource.NewMilliQuantity(100, resource.DecimalSI)          // 100m
var cpu1000 = resource.NewMilliQuantity(1000, resource.DecimalSI)        // 1000m
var memory32 = resource.NewQuantity(100*1024*1024, resource.BinarySI)    // 32Mi
var memory128 = resource.NewQuantity(128*1024*1024, resource.BinarySI)   // 128Mi
var memory150 = resource.NewQuantity(150*1024*1024, resource.BinarySI)   // 128Mi
var memory350 = resource.NewQuantity(350*1024*1024, resource.BinarySI)   // 350Mi
var memory1024 = resource.NewQuantity(1024*1024*1024, resource.BinarySI) // 1024Mi

var rule = `^([a-z0-9]){32,}$`
var wlpClientID = common.GenerateRandomString(rule)
var wlpClientSecret = common.GenerateRandomString(rule)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Authentication Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAuthentication{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("authentication-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Authentication
	err = c.Watch(&source.Kind{Type: &operatorv1alpha1.Authentication{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Certificate and requeue the owner Authentication
	err = c.Watch(&source.Kind{Type: &certmgr.Certificate{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.Authentication{},
	})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Certificate and requeue the owner Authentication
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.Authentication{},
	})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Certificate and requeue the owner Authentication
	err = c.Watch(&source.Kind{Type: &batchv1.Job{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.Authentication{},
	})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Service and requeue the owner Authentication
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.Authentication{},
	})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource ConfigMap and requeue the owner Authentication
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.Authentication{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Ingress and requeue the owner Authentication
	err = c.Watch(&source.Kind{Type: &net.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.Authentication{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Deployment and requeue the owner Authentication
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.Authentication{},
	})
	if err != nil {
		return err
	}

	clusterType, err := pkgCommon.GetClusterType(context.Background(), pkgCommon.GlobalConfigMapName)
	if err != nil {
		return err
	}
	if clusterType == pkgCommon.OpenShift {
		err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &operatorv1alpha1.Authentication{},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// blank assignment to verify that ReconcileAuthentication implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAuthentication{}

// ReconcileAuthentication reconciles a Authentication object
type ReconcileAuthentication struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client      client.Client
	scheme      *runtime.Scheme
	Mutex       sync.Mutex
	clusterType pkgCommon.ClusterType
}

// RunningOnOpenShiftCluster returns whether the Operator is running on an OpenShift cluster
func (r *ReconcileAuthentication) RunningOnOpenShiftCluster() bool {
	return r.clusterType == pkgCommon.OpenShift
}

// RunningOnCNCFCluster returns whether the Operator is running on a CNCF cluster
func (r *ReconcileAuthentication) RunningOnCNCFCluster() bool {
	return r.clusterType == pkgCommon.CNCF
}

// RunningOnUnknownCluster returns whether the Operator is running on an unknown cluster type
func (r *ReconcileAuthentication) RunningOnUnknownCluster() bool {
	return r.clusterType == pkgCommon.Unknown
}

func (r *ReconcileAuthentication) addFinalizer(ctx context.Context, finalizerName string, instance *operatorv1alpha1.Authentication) (err error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if !containsString(instance.ObjectMeta.Finalizers, finalizerName) {
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizerName)
		err = r.client.Update(ctx, instance)
	}
	return
}

// removeFinalizer removes the provided finalizer from the Authentication instance.
func (r *ReconcileAuthentication) removeFinalizer(ctx context.Context, finalizerName string, instance *operatorv1alpha1.Authentication) (err error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if containsString(instance.ObjectMeta.Finalizers, finalizerName) {
		instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizerName)
		err = r.client.Update(ctx, instance)
		if err != nil {
			return fmt.Errorf("error updating the CR to remove the finalizer: %w", err)
		}
	}
	return
}

// needsAuditServiceDummyDataReset compares the state in an Authentication's .spec.auditService and returns whether it
// needs to be overwritten with dummy data.
func needsAuditServiceDummyDataReset(a *operatorv1alpha1.Authentication) bool {
	return a.Spec.AuditService.ImageName != operatorv1alpha1.AuditServiceIgnoreString ||
		a.Spec.AuditService.ImageRegistry != operatorv1alpha1.AuditServiceIgnoreString ||
		a.Spec.AuditService.ImageTag != operatorv1alpha1.AuditServiceIgnoreString ||
		a.Spec.AuditService.SyslogTlsPath != "" ||
		a.Spec.AuditService.Resources != nil
}

// Reconcile reads that state of the cluster for a Authentication object and makes changes based on the state read
// and what is in the Authentication.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileAuthentication) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, err error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	needToRequeue := false

	reconcileCtx := logf.IntoContext(ctx, reqLogger)
	// Set default result
	result = reconcile.Result{}
	// Set Requeue to true if requeue is needed at end of reconcile loop
	defer func() {
		if needToRequeue {
			result.Requeue = true
		}
	}()

	// Determine the type of cluster that the Operator is installed on
	if r.RunningOnUnknownCluster() {
		r.clusterType, err = pkgCommon.GetClusterType(reconcileCtx, pkgCommon.GlobalConfigMapName)
		if err != nil {
			reqLogger.Error(err, "Failed to determine cluster platform")
			return
		} else {
			reqLogger.Info("Set cluster type", "clusterType", r.clusterType)
		}
	}

	reqLogger.Info("Reconciling Authentication")

	// Fetch the Authentication instance
	instance := &operatorv1alpha1.Authentication{}
	err = r.client.Get(reconcileCtx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Set err to nil to signal the error has been handled
			err = nil
		}
		// Return without requeueing
		return
	}

	// Credit: kubebuilder book
	finalizerName := "authentication.operator.ibm.com"
	// Determine if the Authentication CR  is going to be deleted
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Object not being deleted, but add our finalizer so we know to remove this object later when it is going to be deleted
		beforeFinalizerCount := len(instance.GetFinalizers())
		err = r.addFinalizer(reconcileCtx, finalizerName, instance)
		if err != nil {
			return
		}
		afterFinalizerCount := len(instance.GetFinalizers())
		if afterFinalizerCount > beforeFinalizerCount {
			needToRequeue = true
			return
		}
	} else {
		// Object scheduled to be deleted
		err = r.removeFinalizer(reconcileCtx, finalizerName, instance)
		return
	}

	// Be sure to update status before returning if Authentication is found (but only if the Authentication hasn't
	// already been updated, e.g. finalizer update
	defer func() {
		reqLogger.Info("Update status before finishing loop.")
		if reflect.DeepEqual(instance.Status, operatorv1alpha1.AuthenticationStatus{}) {
			instance.Status = operatorv1alpha1.AuthenticationStatus{
				Nodes: []string{},
			}
		}
		currentServiceStatus := r.getCurrentServiceStatus(ctx, r.client, instance)
		if !reflect.DeepEqual(currentServiceStatus, instance.Status.Service) {
			instance.Status.Service = currentServiceStatus
			reqLogger.Info("Current status does not reflect current state; updating")
		}
		err = r.client.Status().Update(ctx, instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update status; trying again")
			currentInstance := &operatorv1alpha1.Authentication{}
			r.client.Get(ctx, request.NamespacedName, currentInstance)
			currentInstance.Status.Service = currentServiceStatus
			err = r.client.Status().Update(ctx, currentInstance)
			if err != nil {
				reqLogger.Error(err, "Retry failed; returning error")
				return
			}
		} else {
			reqLogger.Info("Updated status")
		}
	}()

	// Check if this Certificate already exists and create it if it doesn't
	reqLogger.Info("Creating ibm-iam-operand-restricted serviceaccount")
	currentSA := &corev1.ServiceAccount{}
	err = r.createSA(instance, currentSA, &needToRequeue)
	if err != nil {
		return
	}
	// create operand role and role-binding
	r.createRole(instance)
	r.createRoleBinding(instance)

	// Check if this Certificate already exists and create it if it doesn't
	currentCertificate := &certmgr.Certificate{}
	err = r.handleCertificate(instance, currentCertificate, &needToRequeue)
	if err != nil {
		return
	}

	// Check if this Service already exists and create it if it doesn't
	currentService := &corev1.Service{}
	err = r.handleService(instance, currentService, &needToRequeue)
	if err != nil {
		return
	}

	// Check if this Secret already exists and create it if it doesn't
	currentSecret := &corev1.Secret{}
	err = r.handleSecret(instance, wlpClientID, wlpClientSecret, currentSecret, &needToRequeue)
	if err != nil {
		return
	}

	//Check if this ConfigMap already exists and create it if it doesn't
	currentConfigMap := &corev1.ConfigMap{}
	err = r.handleConfigMap(instance, wlpClientID, wlpClientSecret, currentConfigMap, &needToRequeue)
	if err != nil {
		return
	}

	// Check if this Job already exists and create it if it doesn't
	currentJob := &batchv1.Job{}
	err = r.handleJob(instance, currentJob, &needToRequeue)
	if err != nil {
		return
	}
	// create clusterrole and clusterrolebinding
	r.createClusterRole(instance)
	r.createClusterRoleBinding(instance)

	r.ReconcileRemoveIngresses(ctx, instance, &needToRequeue)
	// updates redirecturi annotations to serviceaccount
	r.handleServiceAccount(instance, &needToRequeue)

	if r.RunningOnOpenShiftCluster() {
		err = r.handleRoutes(ctx, instance, &needToRequeue)
		if err != nil && !errors.IsNotFound(err) {
			return
		}
	}

	// Check if this Deployment already exists and create it if it doesn't
	currentDeployment := &appsv1.Deployment{}
	currentProviderDeployment := &appsv1.Deployment{}
	currentManagerDeployment := &appsv1.Deployment{}
	err = r.handleDeployment(instance, currentDeployment, currentProviderDeployment, currentManagerDeployment, &needToRequeue)
	if err != nil {
		return
	}

	if needsAuditServiceDummyDataReset(instance) {
		instance.SetRequiredDummyData()
		err = r.client.Update(ctx, instance)
		if err != nil {
			return
		}
	}

	return
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
