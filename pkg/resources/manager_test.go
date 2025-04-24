package resources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestRestartDeployment(t *testing.T) {
	// Create a fake client with scheme
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	// Create a test deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}

	// Create a fake client with the deployment
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deployment).Build()

	// Create a test logger
	logger := zap.New()

	// Create resource manager
	manager := NewManager(client, logger)

	// Test restart deployment
	ctx := context.Background()
	namespacedName := types.NamespacedName{Namespace: "default", Name: "test-deployment"}
	err := manager.RestartDeployment(ctx, namespacedName)
	assert.NoError(t, err)

	// Verify the restart annotation was added to the pod template
	updatedDeployment := &appsv1.Deployment{}
	err = client.Get(ctx, namespacedName, updatedDeployment)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeployment.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestRestartStatefulSet(t *testing.T) {
	// Create a fake client with scheme
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	// Create a test statefulset
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			ServiceName: "test-service",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}

	// Create a fake client with the statefulset
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(statefulset).Build()

	// Create a test logger
	logger := zap.New()

	// Create resource manager
	manager := NewManager(client, logger)

	// Test restart statefulset
	ctx := context.Background()
	namespacedName := types.NamespacedName{Namespace: "default", Name: "test-statefulset"}
	err := manager.RestartStatefulSet(ctx, namespacedName)
	assert.NoError(t, err)

	// Verify the restart annotation was added to the pod template
	updatedStatefulSet := &appsv1.StatefulSet{}
	err = client.Get(ctx, namespacedName, updatedStatefulSet)
	assert.NoError(t, err)
	assert.Contains(t, updatedStatefulSet.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestRestartDaemonSet(t *testing.T) {
	// Create a fake client with scheme
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	// Create a test daemonset
	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-daemonset",
			Namespace: "default",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}

	// Create a fake client with the daemonset
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(daemonset).Build()

	// Create a test logger
	logger := zap.New()

	// Create resource manager
	manager := NewManager(client, logger)

	// Test restart daemonset
	ctx := context.Background()
	namespacedName := types.NamespacedName{Namespace: "default", Name: "test-daemonset"}
	err := manager.RestartDaemonSet(ctx, namespacedName)
	assert.NoError(t, err)

	// Verify the restart annotation was added to the pod template
	updatedDaemonSet := &appsv1.DaemonSet{}
	err = client.Get(ctx, namespacedName, updatedDaemonSet)
	assert.NoError(t, err)
	assert.Contains(t, updatedDaemonSet.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}
