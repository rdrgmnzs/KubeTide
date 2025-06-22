package scheduler

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestCronScheduleDebug(t *testing.T) {
	schedule := "5 * * * *" // At minute 5 of every hour on Monday
	location, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)

	// Parse with standard parser
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, err := parser.Parse(schedule)
	require.NoError(t, err)

	// Get current time in LA timezone
	now := time.Now().In(location)
	t.Logf("Current time in LA: %s (weekday: %s)", now.Format(time.RFC3339), now.Weekday())

	// Calculate next 5 runs
	next := now
	for i := 0; i < 5; i++ {
		next = sched.Next(next)
		t.Logf("Next run %d: %s (weekday: %s)", i+1, next.Format(time.RFC3339), next.Weekday())
	}
}

func TestSchedulerWithCurrentTime(t *testing.T) {
	logger := zap.New()

	executed := false
	var executedTime time.Time

	processFunc := func(resourceType string, namespacedName types.NamespacedName, scheduledTime time.Time) {
		executed = true
		executedTime = scheduledTime
		t.Logf("Cron executed at: %s", scheduledTime.Format(time.RFC3339))
	}

	scheduler := NewScheduler(processFunc, logger)
	defer scheduler.Stop()

	// Test with LA timezone and Monday schedule
	location, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)

	namespacedName := types.NamespacedName{Namespace: "test", Name: "monday-test"}
	resourceKey := "test/monday-test"
	resourceType := "Deployment"

	// Use a schedule that should run soon (every minute on Monday)
	schedule := "* * * * 1"

	err = scheduler.Schedule(schedule, location, resourceKey, resourceType, namespacedName)
	require.NoError(t, err)

	// Check the schedule details
	details, exists := scheduler.GetScheduleDetails(resourceType, resourceKey)
	assert.True(t, exists)
	t.Log("Schedule details:", details)

	// If it's Monday, wait for execution
	now := time.Now().In(location)
	if now.Weekday() == time.Monday {
		t.Log("It's Monday, waiting for cron execution...")
		time.Sleep(65 * time.Second) // Wait just over a minute

		if executed {
			t.Logf("Cron successfully executed at: %s", executedTime.Format(time.RFC3339))
		} else {
			t.Error("Cron did not execute within timeout on Monday")
		}
	} else {
		t.Logf("It's not Monday (it's %s), skipping execution wait", now.Weekday())
	}
}

func TestMondayScheduleDebug(t *testing.T) {
	// Debug the specific Monday schedule issue
	schedule := "5 * * * 1" // At minute 5 of every hour on Monday
	location, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)

	// Create a cron instance
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	c := cron.New(cron.WithParser(parser), cron.WithLocation(location))

	// Add the schedule
	executed := false
	entryID, err := c.AddFunc(schedule, func() {
		executed = true
		t.Logf("CRON EXECUTED at %s", time.Now().In(location).Format(time.RFC3339))
	})
	require.NoError(t, err)

	// Start the cron
	c.Start()
	defer c.Stop()

	// Get entry details
	entry := c.Entry(entryID)
	now := time.Now().In(location)

	t.Logf("Current time: %s (weekday: %s)", now.Format(time.RFC3339), now.Weekday())
	t.Logf("Next scheduled: %s (weekday: %s)", entry.Next.Format(time.RFC3339), entry.Next.Weekday())
	t.Logf("Time until next: %s", time.Until(entry.Next))

	// If next run is within 2 minutes and it's Monday, wait for it
	if time.Until(entry.Next) < 2*time.Minute && now.Weekday() == time.Monday {
		t.Log("Next run is soon, waiting...")
		time.Sleep(time.Until(entry.Next) + 5*time.Second)

		if executed {
			t.Log("SUCCESS: Cron executed as expected")
		} else {
			t.Error("FAILED: Cron did not execute")
		}
	}
}
