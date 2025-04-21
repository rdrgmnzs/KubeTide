package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestHandleResource(t *testing.T) {
	// Create a fake client with scheme
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	// Create a test deployment with annotations
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			Annotations: map[string]string{
				ScheduleAnnotation: "0 2 * * *",
				TimezoneAnnotation: "UTC",
			},
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

	// Create reconciler
	defaultTz, _ := time.LoadLocation("UTC")
	reconciler := NewReconciler(client, logger, scheme, defaultTz)

	// Test with schedule annotation
	ctx := context.Background()
	namespacedName := types.NamespacedName{Namespace: "default", Name: "test-deployment"}
	_, err := reconciler.handleResource(ctx, "Deployment", namespacedName, deployment.GetAnnotations())
	assert.NoError(t, err)
	assert.True(t, reconciler.scheduler.HasSchedule("Deployment", "default/test-deployment"))

	// Test without schedule annotation - need to get fresh copy from client
	deploymentNoSchedule := &appsv1.Deployment{}
	err = client.Get(ctx, namespacedName, deploymentNoSchedule)
	assert.NoError(t, err)

	// Remove annotations
	deploymentNoSchedule.Annotations = map[string]string{}
	err = client.Update(ctx, deploymentNoSchedule)
	assert.NoError(t, err)

	_, err = reconciler.handleResource(ctx, "Deployment", namespacedName, deploymentNoSchedule.GetAnnotations())
	assert.NoError(t, err)
	assert.False(t, reconciler.scheduler.HasSchedule("Deployment", "default/test-deployment"))

	// Test with invalid timezone - need to get fresh copy from client
	deploymentInvalidTz := &appsv1.Deployment{}
	err = client.Get(ctx, namespacedName, deploymentInvalidTz)
	assert.NoError(t, err)

	// Set invalid timezone
	deploymentInvalidTz.Annotations = map[string]string{
		ScheduleAnnotation: "0 2 * * *",
		TimezoneAnnotation: "InvalidTZ",
	}
	err = client.Update(ctx, deploymentInvalidTz)
	assert.NoError(t, err)

	_, err = reconciler.handleResource(ctx, "Deployment", namespacedName, deploymentInvalidTz.GetAnnotations())
	assert.NoError(t, err)
	assert.True(t, reconciler.scheduler.HasSchedule("Deployment", "default/test-deployment"))
}
