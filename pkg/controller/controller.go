package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rdrgmnzs/kubetide/pkg/resources"
	"github.com/rdrgmnzs/kubetide/pkg/scheduler"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ScheduleAnnotation = "kubetide.io/schedule"
	TimezoneAnnotation = "kubetide.io/timezone"
)

var (
	restartTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubetide_restart_total",
			Help: "Total number of restarts performed by KubeTide",
		},
		[]string{"resource_type", "namespace", "name"},
	)

	errorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubetide_error_total",
			Help: "Total number of errors encountered by KubeTide",
		},
		[]string{"resource_type", "namespace", "name"},
	)

	scheduleDrift = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubetide_schedule_drift_seconds",
			Help:    "Time drift between scheduled and actual restart time",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600},
		},
		[]string{"resource_type", "namespace", "name"},
	)

	processingLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubetide_resource_processing_latency_seconds",
			Help:    "Time taken to process a resource operation",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"resource_type", "namespace", "name", "operation"},
	)
)

func init() {
	prometheus.MustRegister(restartTotal)
	prometheus.MustRegister(errorTotal)
	prometheus.MustRegister(scheduleDrift)
	prometheus.MustRegister(processingLatency)
}

type Reconciler struct {
	client.Client
	logger      logr.Logger
	scheme      *runtime.Scheme
	scheduler   *scheduler.Scheduler
	defaultTz   *time.Location
	resourceMgr *resources.Manager
}

func NewReconciler(client client.Client, logger logr.Logger, scheme *runtime.Scheme, defaultTz *time.Location) *Reconciler {
	r := &Reconciler{
		Client:    client,
		logger:    logger,
		scheme:    scheme,
		defaultTz: defaultTz,
	}
	r.resourceMgr = resources.NewManager(client, logger)
	r.scheduler = scheduler.NewScheduler(r.processResource, logger)
	return r
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("kubetide-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch Deployments
	if err := c.Watch(
		source.Kind(mgr.GetCache(), &appsv1.Deployment{}),
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return err
	}

	// Watch StatefulSets
	if err := c.Watch(
		source.Kind(mgr.GetCache(), &appsv1.StatefulSet{}),
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return err
	}

	// Watch DaemonSets
	if err := c.Watch(
		source.Kind(mgr.GetCache(), &appsv1.DaemonSet{}),
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	startTime := time.Now()

	log.Info("Reconciling resource", "namespacedName", req.NamespacedName)

	// Try to get resource as a Deployment
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, req.NamespacedName, deployment)
	if err == nil {
		resourceType := "Deployment"
		r.recordLatency(resourceType, req.Namespace, req.Name, "reconcile", startTime)
		return r.handleResource(ctx, resourceType, req.NamespacedName, deployment.GetAnnotations())
	}

	// Try to get resource as a StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err = r.Get(ctx, req.NamespacedName, statefulSet)
	if err == nil {
		resourceType := "StatefulSet"
		r.recordLatency(resourceType, req.Namespace, req.Name, "reconcile", startTime)
		return r.handleResource(ctx, resourceType, req.NamespacedName, statefulSet.GetAnnotations())
	}

	// Try to get resource as a DaemonSet
	daemonSet := &appsv1.DaemonSet{}
	err = r.Get(ctx, req.NamespacedName, daemonSet)
	if err == nil {
		resourceType := "DaemonSet"
		r.recordLatency(resourceType, req.Namespace, req.Name, "reconcile", startTime)
		return r.handleResource(ctx, resourceType, req.NamespacedName, daemonSet.GetAnnotations())
	}

	// If we got here, the resource is not found
	log.Error(err, "error retrieving resource", "namespacedName", req.NamespacedName)
	return ctrl.Result{}, client.IgnoreNotFound(err)
}

func (r *Reconciler) handleResource(ctx context.Context, resourceType string, namespacedName types.NamespacedName, annotations map[string]string) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	schedule, found := annotations[ScheduleAnnotation]
	if !found {
		// No schedule annotation, nothing to do
		log.Info("Resource has no schedule annotation, ignoring",
			"resourceType", resourceType,
			"namespace", namespacedName.Namespace,
			"name", namespacedName.Name)
		r.scheduler.Remove(resourceType, namespacedName.String())
		return ctrl.Result{}, nil
	}

	// Get timezone from annotation or use default
	tzName, found := annotations[TimezoneAnnotation]
	var location *time.Location
	var err error

	if found {
		location, err = time.LoadLocation(tzName)
		if err != nil {
			log.Error(err, "invalid timezone in annotation, using default",
				"timezone", tzName,
				"resourceType", resourceType,
				"namespace", namespacedName.Namespace,
				"name", namespacedName.Name)
			errorTotal.WithLabelValues(resourceType, namespacedName.Namespace, namespacedName.Name).Inc()
			location = r.defaultTz
		}
	} else {
		location = r.defaultTz
	}

	// Schedule the resource for restarts
	err = r.scheduler.Schedule(schedule, location, namespacedName.String(), resourceType, namespacedName)
	if err != nil {
		log.Error(err, "failed to schedule resource",
			"resourceType", resourceType,
			"namespace", namespacedName.Namespace,
			"name", namespacedName.Name)
		errorTotal.WithLabelValues(resourceType, namespacedName.Namespace, namespacedName.Name).Inc()
		return ctrl.Result{}, err
	}

	log.Info("Successfully scheduled resource",
		"schedule", schedule,
		"timezone", location.String(),
		"resourceType", resourceType,
		"namespace", namespacedName.Namespace,
		"name", namespacedName.Name)

	return ctrl.Result{}, nil
}

func (r *Reconciler) processResource(resourceType string, namespacedName types.NamespacedName, scheduledTime time.Time) {
	ctx := context.Background()
	log := r.logger.WithValues(
		"resourceType", resourceType,
		"namespace", namespacedName.Namespace,
		"name", namespacedName.Name,
	)

	startTime := time.Now()
	drift := startTime.Sub(scheduledTime).Seconds()
	scheduleDrift.WithLabelValues(resourceType, namespacedName.Namespace, namespacedName.Name).Observe(drift)

	log.Info("Processing scheduled restart",
		"scheduledTime", scheduledTime,
		"actualTime", startTime,
		"driftSeconds", drift,
	)

	var err error
	switch resourceType {
	case "Deployment":
		err = r.resourceMgr.RestartDeployment(ctx, namespacedName)
	case "StatefulSet":
		err = r.resourceMgr.RestartStatefulSet(ctx, namespacedName)
	case "DaemonSet":
		err = r.resourceMgr.RestartDaemonSet(ctx, namespacedName)
	default:
		log.Error(fmt.Errorf("unknown resource type: %s", resourceType), "Unknown resource type")
		return
	}

	if err != nil {
		log.Error(err, "Failed to restart resource")
		errorTotal.WithLabelValues(resourceType, namespacedName.Namespace, namespacedName.Name).Inc()
		return
	}

	log.Info("Successfully restarted resource")
	restartTotal.WithLabelValues(resourceType, namespacedName.Namespace, namespacedName.Name).Inc()
	r.recordLatency(resourceType, namespacedName.Namespace, namespacedName.Name, "restart", startTime)
}

func (r *Reconciler) recordLatency(resourceType, namespace, name, operation string, startTime time.Time) {
	latency := time.Since(startTime).Seconds()
	processingLatency.WithLabelValues(resourceType, namespace, name, operation).Observe(latency)
}
