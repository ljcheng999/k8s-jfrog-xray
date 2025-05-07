/*
Copyright 2025.

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

package controller

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logger "sigs.k8s.io/controller-runtime/pkg/log"

	jfrogv1alpha1 "kubesource.toolbox/jfrogxray/api/v1alpha1"
)

const (
	xrayOwnerKey       = ".metadata.xrayer"
	customXrayLabelKey = "jfrog-xray-ondemand"
	customXrayPodKey   = "jfrog-xray-ondemand"
	runTimeAnnotation  = "jfrog.kubesource.toolbox/creationTimestamp"
	podNamePrefix      = "jfrog-xray"
	apiGVStr           = "kubesource.toolbox/jfrogxray/api/v1alpha1"
)

// XrayReconciler reconciles a Xray object
type XrayReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	recoder record.EventRecorder // k8s recording events
}

// +kubebuilder:rbac:groups=jfrog.kubesource.toolbox,resources=xrays,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jfrog.kubesource.toolbox,resources=xrays/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jfrog.kubesource.toolbox,resources=xrays/finalizers,verbs=update
// +kubebuilder:rbac:groups=jfrog.kubesource.toolbox,resources=xrays/events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Xray object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *XrayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logger.FromContext(ctx)

	logger.Info("Reconciling Run!")

	// 1: Load the XrayJob by name
	xray := &jfrogv1alpha1.Xray{}
	if err := r.Get(ctx, req.NamespacedName, xray); err != nil {
		logger.Error(err, "Failed to get xray")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// xray.ObjectMeta.Labels[customXrayLabelKey] = customXrayLabelValuePrefix + xray.ResourceVersion

	// 2: List all active jobs within a namespace, and update the status
	childXrays := &jfrogv1alpha1.XrayList{}
	// labelSelector := labels.Set{customXrayLabelKey: req.Name}
	// logger.Info("before", "initial", labelSelector)

	// Declare a custom field key to faster lookup in local cache if the amount of xray keep increases
	if err := r.List(ctx, childXrays, client.InNamespace(req.Namespace), client.MatchingFields{xrayOwnerKey: req.Name}); err != nil {
		logger.Error(err, "unable to list child Xrays")
		return ctrl.Result{}, err
	}

	logger.Info("childXrays.Items", "hello", childXrays.Items)

	activeXrays := []*jfrogv1alpha1.Xray{}
	successfulXrays := []*jfrogv1alpha1.Xray{}
	failedXrays := []*jfrogv1alpha1.Xray{}
	mostRecentTime := &time.Time{} // find the last run so we can update the status

	// Call helper function to check if the any xray finished
	for i, xray := range childXrays.Items {
		_, finishedType := checkXrayFinished(&xray)
		logger.Info("here", "finishedType", finishedType)
		switch finishedType {
		// still going
		case "":
			activeXrays = append(activeXrays, &childXrays.Items[i])
		case batchv1.JobFailed:
			failedXrays = append(failedXrays, &childXrays.Items[i])
		case batchv1.JobComplete:
			successfulXrays = append(successfulXrays, &childXrays.Items[i])
		}

		// We'll store the launch time in an annotation, so we'll reconstitute that from the active jobs themselves.
		runTimeForXray, err := getRunTimeForXray(&xray)
		if err != nil {
			logger.Error(err, "unable to parse run time for child job", "job", &xray)
			continue
		}
		if runTimeForXray != nil {
			// Update the job annotation if it's before
			if mostRecentTime == nil || mostRecentTime.Before(*runTimeForXray) {
				mostRecentTime = runTimeForXray
			}
		}
	}

	if mostRecentTime != nil {
		xray.Status.StartTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		xray.Status.StartTime = nil
	}

	xray.Status.Active = int32(len(activeXrays))
	// for _, activeJob := range activeJobs {
	// 	jobRef, err := ref.GetReference(r.Scheme, activeJob)
	// 	if err != nil {
	// 		logger.Error(err, "unable to make reference to active job", "job", activeJob)
	// 		continue
	// 	}
	// 	xray.Status.Active += 1
	// }
	for _, successfulJob := range successfulXrays {
		jobRef, err := ref.GetReference(r.Scheme, successfulJob)
		if err != nil {
			logger.Error(err, "unable to make reference to successful job", "job", successfulJob)
			continue
		}
		// cronJob.Status.Active = append(cronJob.Status.Active, *jobRef)
		xray.Status.UncountedTerminatedPods.Succeeded = append(xray.Status.UncountedTerminatedPods.Succeeded, jobRef.UID)
	}
	for _, failedJob := range failedXrays {
		jobRef, err := ref.GetReference(r.Scheme, failedJob)
		if err != nil {
			logger.Error(err, "unable to make reference to failed job", "job", failedJob)
			continue
		}
		// cronJob.Status.Active = append(cronJob.Status.Active, *jobRef)
		xray.Status.UncountedTerminatedPods.Failed = append(xray.Status.UncountedTerminatedPods.Failed, jobRef.UID)
	}

	logger.V(1).Info("xray count", "active xrays", len(activeXrays), "successful xrays", len(successfulXrays), "failed xrays", len(failedXrays))

	// Business Logic start here
	// xray.Spec.Template.CreationTimestamp = &metav1.NewTime()

	// podsReady := []bool
	// for i, xrayContainer := xray.Spec.Template.Spec.Containers {
	// 	podReady[i] := false
	// }

	podReady := false
	// Add or update Pod
	if err := r.addPodIfNotExists(ctx, xray); err != nil {
		logger.Error(err, "Failed to add or update Deployment for Ghost")
		// addCondition(&xray.Status, "PodNotReady", metav1.ConditionFalse, "PodNotReady", "Failed to add or update Pod for Xray")
		// Record the Xray Event
		r.recoder.Event(xray, corev1.EventTypeNormal, "SuccessfulCreate", "Created pod: "+req.Name)
		return ctrl.Result{}, err
	} else {
		podReady = true
	}

	if podReady {
		logger.Info("End ready")
	}

	if err := r.updateStatus(ctx, xray); err != nil {
		logger.Error(err, "Failed to update Xray status")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling End!")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *XrayReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &jfrogv1alpha1.Xray{}, xrayOwnerKey, func(rawObj client.Object) []string {
		// grab the job object, extract the owner...
		xray := rawObj.(*jfrogv1alpha1.Xray)
		owner := metav1.GetControllerOf(xray)
		if owner == nil {
			return nil
		}
		// ...make sure it's a Xray...
		if owner.APIVersion != apiGVStr || owner.Kind != "Xray" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	r.recoder = mgr.GetEventRecorderFor("xray-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&jfrogv1alpha1.Xray{}).
		// Owns(&jfrogv1alpha1.Xray{}).
		Named("xray").
		Complete(r)
}
