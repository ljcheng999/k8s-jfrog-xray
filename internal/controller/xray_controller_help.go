package controller

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	jfrogv1alpha1 "kubesource.toolbox/jfrogxray/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
)

func isStringEmpty(str string) bool {
	return len(str) == 0
}

// Function to add a condition to the XrayStatus
func addCondition(status *jfrogv1alpha1.XrayStatus, condType string, statusType metav1.ConditionStatus, reason, message string) {

	fmt.Println("status.Conditions: ", len(status.Conditions))

	// for i, existingCondition := range status.Conditions {
	// 	if existingCondition.Type == batchv1.JobComplete {

	// 	}

	// 	// if existingCondition.Type == condType {
	// 	// 	// Condition already exists, update it
	// 	// 	status.Conditions[i].Status = statusType
	// 	// 	status.Conditions[i].Reason = reason
	// 	// 	status.Conditions[i].Message = message
	// 	// 	status.Conditions[i].LastTransitionTime = metav1.Now()
	// 	// 	return
	// 	// }
	// }

	// // Condition does not exist, add it
	// condition := metav1.Condition{
	// 	Type:               condType,
	// 	Status:             statusType,
	// 	Reason:             reason,
	// 	Message:            message,
	// 	LastTransitionTime: metav1.Now(),
	// }
	// status.Conditions = append(status.Conditions, condition)
}

func checkXrayFinished(xray *jfrogv1alpha1.Xray) (bool, batchv1.JobConditionType) {
	for _, condition := range xray.Status.Conditions {
		if (condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed) && condition.Status == corev1.ConditionTrue {
			return true, condition.Type
		}
	}
	return false, ""
}

// A helper function to set and extract the job runtime from the annotation that we added during job creation.
func getRunTimeForXray(xray *jfrogv1alpha1.Xray) (*time.Time, error) {
	timeRaw := xray.Annotations[runTimeAnnotation]
	if len(timeRaw) == 0 {
		return nil, nil
	}

	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		return nil, err
	}
	return &timeParsed, nil
}

// Function to update the status of the Xray object
func (r *XrayReconciler) updateStatus(ctx context.Context, xray *jfrogv1alpha1.Xray) error {
	// Update the status of the Xray object
	if err := r.Status().Update(ctx, xray); err != nil {
		return err
	}

	return nil
}

// // A helper function to set and extract the job runtime from the annotation that we added during job creation.
// func getRunTimeForjob(job *batchv1.Job) (*time.Time, error) {
// 	timeRaw := job.Annotations[runTimeAnnotation]
// 	if len(timeRaw) == 0 {
// 		return nil, nil
// 	}

// 	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &timeParsed, nil
// }

// checks whether the pvc is already created and if not, it will create it in the right namespace.
func (r *XrayReconciler) addPodIfNotExists(ctx context.Context, xray *jfrogv1alpha1.Xray) error {
	logger := logger.FromContext(ctx)
	logger.Info("addPodIfNotExists Run!")

	pod := &corev1.Pod{}
	podName := customXrayPodKey + "-pod"

	if err := r.Get(ctx, client.ObjectKey{Namespace: xray.ObjectMeta.Namespace, Name: podName}, pod); err == nil {
		// Pod exists, we are done here!
		return nil
	}

	// // Pod does not exist, create it
	desiredPod := generateDesiredPod(xray, podName)
	logger.Info("Pod is creating", "Pod", podName)

	if err := controllerutil.SetControllerReference(xray, desiredPod, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, desiredPod); err != nil {
		return err
	}
	// r.recoder.Event(xray, corev1.EventTypeNormal, "SuccessfulCreate", "Created pod: "+podName)
	logger.Info("Pod created", "Pod", podName)
	return nil
}

func generateDesiredPod(xray *jfrogv1alpha1.Xray, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: xray.ObjectMeta.Namespace,
			Labels: map[string]string{
				"app": customXrayLabelKey + "-pod",
			},
			// CreationTimestamp: metav1.Time{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    xray.ObjectMeta.Name,
					Image:   xray.Spec.Template.Spec.Containers[0].Image,
					Command: xray.Spec.Template.Spec.Containers[0].Command,
				},
			},
			RestartPolicy: xray.Spec.Template.Spec.RestartPolicy,
		},
	}
}
