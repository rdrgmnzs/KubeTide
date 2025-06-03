package resources

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
)

// ResourceManager defines the interface for managing Kubernetes resources
type ResourceManager interface {
	RestartDeployment(ctx context.Context, namespacedName types.NamespacedName) error
	RestartStatefulSet(ctx context.Context, namespacedName types.NamespacedName) error
	RestartDaemonSet(ctx context.Context, namespacedName types.NamespacedName) error
}
