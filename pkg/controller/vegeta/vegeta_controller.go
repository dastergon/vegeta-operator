package vegeta

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	vegetav1alpha1 "github.com/dastergon/vegeta-operator/pkg/apis/vegeta/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	vegetaContainerImage = "peterevans/vegeta"
	containerName        = "vegeta"
	mountPath            = "/report"
)

var log = logf.Log.WithName("controller_vegeta")

// Add creates a new Vegeta Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVegeta{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vegeta-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Vegeta
	err = c.Watch(&source.Kind{Type: &vegetav1alpha1.Vegeta{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Vegeta
	err = c.Watch(&source.Kind{Type: &vegetav1alpha1.Vegeta{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &vegetav1alpha1.Vegeta{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileVegeta implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVegeta{}

// ReconcileVegeta reconciles a Vegeta object
type ReconcileVegeta struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Vegeta object and makes changes based on the state read
// and what is in the Vegeta.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVegeta) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Vegeta")

	// Fetch the Vegeta instance
	instance := &vegetav1alpha1.Vegeta{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Job object.
	job := newJobForCR(instance)

	// Set Vegeta instance as the owner and controller.
	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the Job already exists.
	found := &batchv1.Job{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
		err = r.client.Create(context.TODO(), job)
		if err != nil {
			return reconcile.Result{}, err
		}

		for {
			err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				reqLogger.Error(err, "Failed to retrieve object from Kubernetes", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
				return reconcile.Result{}, err
			}

			if found.Status.Succeeded == 1 {
				for _, c := range found.Status.Conditions {
					if c.Type == "Complete" && c.Status == "True" {
						reqLogger.Info("Job completed", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
					}
				}
				break
			}
		}

		// Job created successfully - don't requeue.
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Job already exists - don't requeue.
	reqLogger.Info("Skip reconcile: Job already exists", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
	return reconcile.Result{}, nil
}

// newJobForCR returns a Job with the same name/namespace as the CR.
func newJobForCR(cr *vegetav1alpha1.Vegeta) *batchv1.Job {
	labels := map[string]string{
		"app": cr.Name,
	}

	envVars := []corev1.EnvVar{}
	if cr.Spec.BlobStorage != nil && cr.Spec.BlobStorage.Env != nil {
		envVars = cr.Spec.BlobStorage.Env
	}

	resources := corev1.ResourceRequirements{}
	if Resources(cr.Spec.Resources) != nil {
		resources = cr.Spec.Resources
	}

	command := assembleCommand(cr.Spec)

	if hasBlobStorage(cr.Spec) {
		rcloneCmd := assembleRclone(cr.Spec)
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cr.Name + "-job",
				Namespace: cr.Namespace,
				Labels:    labels,
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name:      containerName,
								Image:     vegetaContainerImage,
								Command:   []string{"/bin/sh"},
								Args:      []string{"-c", strings.Join(command, " ")},
								Resources: resources,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "vegeta-report",
										MountPath: mountPath,
									},
								},
								WorkingDir: mountPath,
							},
						},
						Containers: []corev1.Container{
							{
								Name:      "export-vegeta-report-to-cloud",
								Image:     "rclone/rclone",
								Command:   []string{"/bin/sh"},
								Args:      []string{"-c", strings.Join(rcloneCmd, " ")},
								Env:       envVars,
								Resources: resources,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "vegeta-report",
										MountPath: mountPath,
									},
								},
								WorkingDir: mountPath,
							},
						},
						RestartPolicy: "Never",
						Volumes: []corev1.Volume{
							{
								Name: "vegeta-report",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
						},
					},
				},
			},
		}
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-job",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      containerName,
							Image:     vegetaContainerImage,
							Command:   []string{"/bin/sh"},
							Args:      []string{"-c", strings.Join(command, " ")},
							Resources: resources,
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}
}

// assembleRclone returns the actual command to transfer local report to a blob storage system.
func assembleRclone(spec vegetav1alpha1.VegetaSpec) []string {
	report := ""
	if spec.Attack.Output != "" {
		report = spec.Attack.Output
	} else if spec.Attack.Report != nil && (spec.Attack.Report.Output != "" && spec.Attack.Report.Output != "stdout") {
		report = spec.Attack.Report.Output
	} else {
		return []string{}
	}

	command := []string{"rclone", "--config /dev/null", "copy", report}
	switch spec.BlobStorage.Provider {
	case "aws":
		command = append(command, "--s3-env-auth", "s3:"+spec.BlobStorage.Name)
	default:
		return []string{}
	}
	return command
}

// hasBlobStorage checks if all conditions are met to store to remote blob storage.
func hasBlobStorage(spec vegetav1alpha1.VegetaSpec) bool {
	return spec.BlobStorage != nil && (spec.Attack.Report != nil) && (spec.Attack.Report.Output != "" && spec.Attack.Report.Output != "stdout")
}

func Resources(resources corev1.ResourceRequirements) *corev1.ResourceRequirements {
	return &resources
}

func assembleCommand(spec vegetav1alpha1.VegetaSpec) []string {
	command := []string{}

	if spec.Target != "" {
		command = append(command, "echo", `"GET "`+spec.Target+`"`, "|")
	}

	command = append(command, "vegeta", "attack")

	if spec.Attack != nil {
		if spec.Attack.Body != "" {
			command = append(command, "-body", spec.Attack.Body)
		}

		if spec.Attack.Cert != "" {
			command = append(command, "-cert", spec.Attack.Cert)
		}

		if spec.Attack.Chunked {
			command = append(command, "-chunked")
		}

		if spec.Attack.Connections > 0 {
			command = append(command, "-connections", strconv.Itoa(spec.Attack.Connections))
		}

		if spec.Attack.Duration != "" {
			command = append(command, "-duration", spec.Attack.Duration)
		}

		if spec.Attack.H2C {
			command = append(command, "-h2c")
		}

		if spec.Attack.Header != "" {
			command = append(command, "-header", spec.Attack.Header)
		}

		if spec.Attack.HTTP2 {
			command = append(command, "-http2")
		}

		if spec.Attack.Insecure {
			command = append(command, "-insecure")
		}

		if spec.Attack.KeepAlive {
			command = append(command, "-keepalive")
		}

		if spec.Attack.Key != "" {
			command = append(command, "-key", spec.Attack.Key)
		}

		if spec.Attack.LAddr != "" {
			command = append(command, "-laddr", spec.Attack.LAddr)
		}

		if spec.Attack.Lazy {
			command = append(command, "-lazy")
		}

		if spec.Attack.MaxBody != 0 {
			command = append(command, "-max-body", fmt.Sprint(spec.Attack.MaxBody))
		}

		if spec.Attack.MaxWorkers != 0 {
			command = append(command, "-max-workers", fmt.Sprint(spec.Attack.MaxWorkers))
		}

		if spec.Attack.Name != "" {
			command = append(command, "-name", spec.Attack.Name)
		}

		if spec.Attack.Output != "" {
			command = append(command, "-output", spec.Attack.Output)
		}

		if spec.Attack.ProxyHeader != "" {
			command = append(command, "-proxy-header", spec.Attack.ProxyHeader)
		}

		if spec.Attack.Rate != "" {
			command = append(command, "-rate", spec.Attack.Rate)
		}

		if spec.Attack.Redirects != 0 {
			command = append(command, "-redirects", strconv.Itoa(spec.Attack.Redirects))
		}

		if spec.Attack.Resolvers != "" {
			command = append(command, "-resolvers", spec.Attack.Resolvers)
		}

		if spec.Attack.RootCerts != "" {
			command = append(command, "-root-certs", spec.Attack.RootCerts)
		}

		if spec.Attack.Targets != "" {
			command = append(command, "-targets", spec.Attack.Targets)
		}

		if spec.Attack.Timeout != "" {
			command = append(command, "-timeout", spec.Attack.Timeout)
		}

		if spec.Attack.UnixSocket != "" {
			command = append(command, "-unix-socket", spec.Attack.UnixSocket)
		}

		if spec.Attack.Workers > 0 {
			command = append(command, "-workers", fmt.Sprint(spec.Attack.Workers))
		}

		if spec.Attack.Output == "" {
			if spec.Attack.Report != nil {
				command = append(command, "|", "vegeta", "report")
				if spec.Attack.Report.Buckets != "" {
					command = append(command, "-buckets", spec.Attack.Report.Buckets)
				}

				if spec.Attack.Report.Every != "" {
					command = append(command, "-every", spec.Attack.Report.Every)
				}

				if spec.Attack.Report.Output != "" {
					command = append(command, "-output", spec.Attack.Report.Output)
				}
				if spec.Attack.Report.Type != "" {
					command = append(command, "-type", spec.Attack.Report.Type)
				}
			}

		}
	}

	return command
}
