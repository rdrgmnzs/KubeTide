package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestScheduler(t *testing.T) {
	// Create test logger
	logger := zap.New()

	// Create a test process function
	processFunc := func(resourceType string, namespacedName types.NamespacedName, scheduledTime time.Time) {
		// Test logic here
	}

	// Create scheduler
	scheduler := NewScheduler(processFunc, logger)
	defer scheduler.Stop()

	// Test schedule
	location, _ := time.LoadLocation("UTC")
	namespacedName := types.NamespacedName{Namespace: "default", Name: "test-deployment"}
	resourceKey := "default/test-deployment"
	resourceType := "Deployment"

	// Schedule with valid cron expression
	err := scheduler.Schedule("0 0 * * *", location, resourceKey, resourceType, namespacedName)
	assert.NoError(t, err)
	assert.True(t, scheduler.HasSchedule(resourceType, resourceKey))

	// Schedule with invalid cron expression
	err = scheduler.Schedule("invalid", location, resourceKey, resourceType, namespacedName)
	assert.Error(t, err)

	// Remove schedule
	scheduler.Remove(resourceType, resourceKey)
	assert.False(t, scheduler.HasSchedule(resourceType, resourceKey))
}
