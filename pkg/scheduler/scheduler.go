package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/types"
)

type ProcessResourceFunc func(resourceType string, namespacedName types.NamespacedName, scheduledTime time.Time)

type scheduledResource struct {
	cronID         cron.EntryID
	resourceType   string
	namespacedName types.NamespacedName
}

type Scheduler struct {
	cron        *cron.Cron
	resources   map[string]scheduledResource
	mutex       sync.RWMutex
	processFunc ProcessResourceFunc
	logger      logr.Logger
}

func NewScheduler(processFunc ProcessResourceFunc, logger logr.Logger) *Scheduler {
	// Create cron scheduler with seconds field enabled
	cronParser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	scheduler := &Scheduler{
		cron:        cron.New(cron.WithParser(cronParser), cron.WithLocation(time.UTC)),
		resources:   make(map[string]scheduledResource),
		processFunc: processFunc,
		logger:      logger.WithName("scheduler"),
	}

	scheduler.cron.Start()
	return scheduler
}

func (s *Scheduler) Schedule(
	schedule string,
	location *time.Location,
	resourceKey string,
	resourceType string,
	namespacedName types.NamespacedName,
) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a unique key that includes the resource type to prevent collisions
	uniqueKey := fmt.Sprintf("%s-%s", resourceType, resourceKey)

	// If already scheduled, remove previous schedule
	if existing, ok := s.resources[uniqueKey]; ok {
		s.cron.Remove(existing.cronID)
	}

	// Parse the schedule to validate it
	_, err := cron.ParseStandard(schedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule '%s': %w", schedule, err)
	}

	// Schedule the actual job with our cron instance
	cronID, err := s.cron.AddFunc(schedule, func() {
		scheduledTime := time.Now()
		s.logger.Info("Executing scheduled restart",
			"resourceKey", resourceKey,
			"resourceType", resourceType,
			"namespace", namespacedName.Namespace,
			"name", namespacedName.Name,
			"schedule", schedule,
			"timezone", location.String(),
		)
		s.processFunc(resourceType, namespacedName, scheduledTime)
	})

	if err != nil {
		return fmt.Errorf("failed to schedule cron job: %w", err)
	}

	s.resources[uniqueKey] = scheduledResource{
		cronID:         cronID,
		resourceType:   resourceType,
		namespacedName: namespacedName,
	}

	s.logger.Info("Resource scheduled",
		"resourceKey", uniqueKey,
		"resourceType", resourceType,
		"namespace", namespacedName.Namespace,
		"name", namespacedName.Name,
		"schedule", schedule,
		"timezone", location.String(),
	)

	return nil
}

func (s *Scheduler) Remove(resourceType string, resourceKey string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	uniqueKey := fmt.Sprintf("%s-%s", resourceType, resourceKey)

	if existing, ok := s.resources[uniqueKey]; ok {
		s.cron.Remove(existing.cronID)
		delete(s.resources, uniqueKey)

		s.logger.Info("Resource schedule removed",
			"resourceKey", uniqueKey,
			"resourceType", existing.resourceType,
			"namespace", existing.namespacedName.Namespace,
			"name", existing.namespacedName.Name,
		)
	}
}

func (s *Scheduler) HasSchedule(resourceType string, resourceKey string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	uniqueKey := fmt.Sprintf("%s-%s", resourceType, resourceKey)
	_, exists := s.resources[uniqueKey]
	return exists
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// GetScheduledResources returns a map of all currently scheduled resources for debugging
func (s *Scheduler) GetScheduledResources() map[string]string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]string)
	for key, resource := range s.resources {
		result[key] = fmt.Sprintf("ResourceType: %s, Namespace: %s, Name: %s, CronID: %d",
			resource.resourceType,
			resource.namespacedName.Namespace,
			resource.namespacedName.Name,
			resource.cronID)
	}
	return result
}

// GetScheduleDetails returns detailed information about a specific scheduled resource
func (s *Scheduler) GetScheduleDetails(resourceType string, resourceKey string) (string, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	uniqueKey := fmt.Sprintf("%s-%s", resourceType, resourceKey)
	if resource, exists := s.resources[uniqueKey]; exists {
		entry := s.cron.Entry(resource.cronID)
		return fmt.Sprintf(
			"ResourceType: %s\nNamespace: %s\nName: %s\nNext Run: %v\nPrev Run: %v",
			resource.resourceType,
			resource.namespacedName.Namespace,
			resource.namespacedName.Name,
			entry.Next,
			entry.Prev,
		), true
	}
	return "", false
}
