/*
Copyright 2021.

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
	"reflect"

	//	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchjobsv1alpha1 "github.com/mcastelino/jobqueue-operator/api/v1alpha1"
)

// JobQueueReconciler reconciles a JobQueue object
type JobQueueReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batchjobs.example.com,resources=jobqueues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batchjobs.example.com,resources=jobqueues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batchjobs.example.com,resources=jobqueues/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the JobQueue object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *JobQueueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("jobqueue", req.NamespacedName)

	jobQueue := &batchjobsv1alpha1.JobQueue{}
	err := r.Get(ctx, req.NamespacedName, jobQueue)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("jobQueue resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Error Reading the jobqueue")
		return ctrl.Result{}, err
	}

	//work := jobQueue.Spec.Work
	size := jobQueue.Spec.Size
	workSize := jobQueue.Spec.WorkSize

	if workSize < size {
		jobQueue.Status.Status = "Red"
		if err := r.Status().Update(ctx, jobQueue); err != nil {
			log.Error(err, "Failed to update status")
		}
		return ctrl.Result{}, fmt.Errorf("WorkSize (%d) < Size (%d)", workSize, size)
	}

	// Check if the deployment already exists, if not create a new one
	found := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: jobQueue.Name, Namespace: jobQueue.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForJobQueue(jobQueue)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	if *found.Spec.Parallelism != size || *found.Spec.Completions != workSize {
		found.Spec.Parallelism = &size
		found.Spec.Completions = &workSize //Note: This cannot be changed. We should check and bail
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		log.Info("Updated Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	// Update the JobQueue status with the pod names
	// List the pods for this JobQueue's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(jobQueue.Namespace),
		client.MatchingLabels(labelsForJobQueue(jobQueue.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "jobQueue.Namespace", jobQueue.Namespace, "jobQueue.Name", jobQueue.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Ensure the deployment size is the same as the spec
	if *found.Spec.Parallelism != size {
		found.Spec.Parallelism = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, jobQueue.Status.Nodes) {
		jobQueue.Status.Nodes = podNames
		err := r.Status().Update(ctx, jobQueue)
		if err != nil {
			log.Error(err, "Failed to update Memcached status")
			return ctrl.Result{}, err
		}
	}

	jobQueue.Status.ActiveJobs = size
	jobQueue.Status.WorkQueueSize = workSize
	jobQueue.Status.Status = "Green"
	if err := r.Status().Update(ctx, jobQueue); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobQueueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchjobsv1alpha1.JobQueue{}).
		Complete(r)
}

// deploymentForJobQueue returns a jobqueue Deployment object
func (r *JobQueueReconciler) deploymentForJobQueue(m *batchjobsv1alpha1.JobQueue) *batchv1.Job {
	ls := labelsForJobQueue(m.Name)
	replicas := m.Spec.Size
	items := m.Spec.WorkSize

	dep := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: batchv1.JobSpec{
			Parallelism: &replicas,
			Completions: &items,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Image:   "busybox",
						Name:    "batchjob",
						Command: []string{"sleep", "3600"},
					}},
				},
			},
		},
	}
	// Set JobQueue instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForJobQueue returns the labels for selecting the resources
// belonging to the given jobqueue CR name.
func labelsForJobQueue(name string) map[string]string {
	return map[string]string{"app": "jobqueue", "jobqueue_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
