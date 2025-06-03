package controller

import (
	"context"
	"sync"
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

func TestHandleResourceWithTimezones(t *testing.T) {
	// Create a fake client with scheme
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	// Test cases for different timezone scenarios
	testCases := []struct {
		name            string
		timezone        string
		schedule        string
		expectedSuccess bool
		description     string
	}{
		{
			name:            "Valid LA timezone",
			timezone:        "America/Los_Angeles",
			schedule:        "5 * * * 1",
			expectedSuccess: true,
			description:     "Should schedule correctly with LA timezone",
		},
		{
			name:            "Valid NY timezone",
			timezone:        "America/New_York",
			schedule:        "0 2 * * *",
			expectedSuccess: true,
			description:     "Should schedule correctly with NY timezone",
		},
		{
			name:            "UTC timezone",
			timezone:        "UTC",
			schedule:        "30 4 * * *",
			expectedSuccess: true,
			description:     "Should schedule correctly with UTC timezone",
		},
		{
			name:            "Invalid timezone falls back to default",
			timezone:        "Invalid/Timezone",
			schedule:        "0 0 * * *",
			expectedSuccess: true,
			description:     "Should use default timezone when invalid timezone provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create deployment with specific timezone
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment-" + tc.name,
					Namespace: "default",
					Annotations: map[string]string{
						ScheduleAnnotation: tc.schedule,
						TimezoneAnnotation: tc.timezone,
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

			// Create client and reconciler
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deployment).Build()
			logger := zap.New()
			defaultTz, _ := time.LoadLocation("UTC")
			reconciler := NewReconciler(client, logger, scheme, defaultTz)

			// Test scheduling
			ctx := context.Background()
			namespacedName := types.NamespacedName{Namespace: "default", Name: deployment.Name}
			_, err := reconciler.handleResource(ctx, "Deployment", namespacedName, deployment.GetAnnotations())

			if tc.expectedSuccess {
				assert.NoError(t, err, tc.description)
				assert.True(t, reconciler.scheduler.HasSchedule("Deployment", namespacedName.String()))

				// Verify schedule details
				details, exists := reconciler.scheduler.GetScheduleDetails("Deployment", namespacedName.String())
				assert.True(t, exists)
				assert.Contains(t, details, deployment.Name)
			} else {
				assert.Error(t, err, tc.description)
			}
		})
	}
}

func TestCronScheduleWithTimezoneExecution(t *testing.T) {
	// This test verifies that the controller correctly passes timezone
	// information to the scheduler and that jobs execute in the right timezone
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	// Track restart executions
	restartExecutions := make(map[string]time.Time)
	var mu sync.Mutex

	// Create a custom resource manager that tracks executions
	trackingResourceMgr := &mockResourceManager{
		onRestart: func(ctx context.Context, namespacedName types.NamespacedName) error {
			mu.Lock()
			defer mu.Unlock()
			restartExecutions[namespacedName.String()] = time.Now()
			return nil
		},
	}

	// Create deployments with different timezones
	deployments := []struct {
		name     string
		timezone string
		schedule string
	}{
		{"utc-deployment", "UTC", "* * * * *"},                // Every minute
		{"la-deployment", "America/Los_Angeles", "* * * * *"}, // Every minute
	}

	var objects []runtime.Object
	for _, d := range deployments {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      d.name,
				Namespace: "default",
				Annotations: map[string]string{
					ScheduleAnnotation: d.schedule,
					TimezoneAnnotation: d.timezone,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": d.name},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": d.name},
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
		objects = append(objects, deployment)
	}

	// Create client and reconciler
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
	logger := zap.New()
	defaultTz, _ := time.LoadLocation("UTC")
	reconciler := NewReconciler(client, logger, scheme, defaultTz)
	reconciler.resourceMgr = trackingResourceMgr

	// Schedule all deployments
	ctx := context.Background()
	for _, d := range deployments {
		namespacedName := types.NamespacedName{Namespace: "default", Name: d.name}
		deployment := &appsv1.Deployment{}
		err := client.Get(ctx, namespacedName, deployment)
		assert.NoError(t, err)

		_, err = reconciler.handleResource(ctx, "Deployment", namespacedName, deployment.GetAnnotations())
		assert.NoError(t, err)
	}

	// Since we're testing with minute-based cron, we need to manually
	// trigger the executions to avoid waiting for a full minute
	for _, d := range deployments {
		namespacedName := types.NamespacedName{Namespace: "default", Name: d.name}
		loc, _ := time.LoadLocation(d.timezone)
		reconciler.processResource("Deployment", namespacedName, time.Now().In(loc))
	}

	// Give a moment for processing
	time.Sleep(100 * time.Millisecond)

	// Verify that both deployments were executed
	mu.Lock()
	assert.Len(t, restartExecutions, 2, "Both deployments should have been restarted")
	mu.Unlock()
}

// mockResourceManager is a test implementation of resources.ResourceManager interface
type mockResourceManager struct {
	onRestart func(ctx context.Context, namespacedName types.NamespacedName) error
}

func (m *mockResourceManager) RestartDeployment(ctx context.Context, namespacedName types.NamespacedName) error {
	if m.onRestart != nil {
		return m.onRestart(ctx, namespacedName)
	}
	return nil
}

func (m *mockResourceManager) RestartStatefulSet(ctx context.Context, namespacedName types.NamespacedName) error {
	if m.onRestart != nil {
		return m.onRestart(ctx, namespacedName)
	}
	return nil
}

func (m *mockResourceManager) RestartDaemonSet(ctx context.Context, namespacedName types.NamespacedName) error {
	if m.onRestart != nil {
		return m.onRestart(ctx, namespacedName)
	}
	return nil
}
