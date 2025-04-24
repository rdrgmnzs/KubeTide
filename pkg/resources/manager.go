package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager struct {
	client client.Client
	logger logr.Logger
}

func NewManager(client client.Client, logger logr.Logger) *Manager {
	return &Manager{
		client: client,
		logger: logger.WithName("resource-manager"),
	}
}

func (m *Manager) RestartDeployment(ctx context.Context, namespacedName types.NamespacedName) error {
	m.logger.Info("Restarting deployment",
		"namespace", namespacedName.Namespace,
		"name", namespacedName.Name,
	)

	// Get the deployment
	deployment := &appsv1.Deployment{}
	if err := m.client.Get(ctx, namespacedName, deployment); err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Add restart annotation
	return m.patchRestartAnnotation(ctx, deployment)
}

func (m *Manager) RestartStatefulSet(ctx context.Context, namespacedName types.NamespacedName) error {
	m.logger.Info("Restarting statefulset",
		"namespace", namespacedName.Namespace,
		"name", namespacedName.Name,
	)

	// Get the statefulset
	statefulset := &appsv1.StatefulSet{}
	if err := m.client.Get(ctx, namespacedName, statefulset); err != nil {
		return fmt.Errorf("failed to get statefulset: %w", err)
	}

	// Add restart annotation
	return m.patchRestartAnnotation(ctx, statefulset)
}

func (m *Manager) RestartDaemonSet(ctx context.Context, namespacedName types.NamespacedName) error {
	m.logger.Info("Restarting daemonset",
		"namespace", namespacedName.Namespace,
		"name", namespacedName.Name,
	)

	// Get the daemonset
	daemonset := &appsv1.DaemonSet{}
	if err := m.client.Get(ctx, namespacedName, daemonset); err != nil {
		return fmt.Errorf("failed to get daemonset: %w", err)
	}

	// Add restart annotation
	return m.patchRestartAnnotation(ctx, daemonset)
}

func (m *Manager) patchRestartAnnotation(ctx context.Context, obj client.Object) error {
	// Create a patch with the kubectl.kubernetes.io/restartedAt annotation
	patch := client.MergeFrom(obj.DeepCopyObject().(client.Object))

	// Get current timestamp
	currentTime := time.Now().Format(time.RFC3339)

	// Handle different resource types
	switch resource := obj.(type) {
	case *appsv1.Deployment:
		// Initialize pod template annotations if nil
		if resource.Spec.Template.Annotations == nil {
			resource.Spec.Template.Annotations = make(map[string]string)
		}
		// Add the restart annotation to pod template
		resource.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = currentTime

	case *appsv1.StatefulSet:
		// Initialize pod template annotations if nil
		if resource.Spec.Template.Annotations == nil {
			resource.Spec.Template.Annotations = make(map[string]string)
		}
		// Add the restart annotation to pod template
		resource.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = currentTime

	case *appsv1.DaemonSet:
		// Initialize pod template annotations if nil
		if resource.Spec.Template.Annotations == nil {
			resource.Spec.Template.Annotations = make(map[string]string)
		}
		// Add the restart annotation to pod template
		resource.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = currentTime

	default:
		return fmt.Errorf("unsupported resource type: %T", obj)
	}

	// Apply the patch
	if err := m.client.Patch(ctx, obj, patch); err != nil {
		return fmt.Errorf("failed to patch resource with restart annotation: %w", err)
	}

	m.logger.Info("Successfully patched resource with restart annotation",
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)

	return nil
}
