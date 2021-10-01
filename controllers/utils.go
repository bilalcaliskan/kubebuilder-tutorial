package controllers

import (
	"fmt"
	batch "github.com/bilalcaliskan/kubebuilder-tutorial/api/v1"
	"github.com/robfig/cron"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// We consider a job “finished” if it has a “Complete” or “Failed” condition marked as true. Status conditions allow us
// to add extensible status information to our objects that other humans and controllers can examine to check things
// like completion and health.
func isJobFinished(job *kbatch.Job) (bool, kbatch.JobConditionType) {
	for _, c := range job.Status.Conditions {
		if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
			return true, c.Type
		}
	}

	return false, ""
}

// We’ll use a helper to extract the scheduled time from the annotation that we added during job creation.
func getScheduledTimeForJob(job *kbatch.Job) (*time.Time, error) {
	timeRaw := job.Annotations[scheduledTimeAnnotation]
	if len(timeRaw) == 0 {
		return nil, nil
	}

	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		return nil, err
	}

	return &timeParsed, nil
}

// getNextSchedule calculates the next scheduled time using our helpful cron library
func getNextSchedule(cronJob *batch.CronJob, now time.Time) (lastMissed time.Time, next time.Time, err error) {
	sched, err := cron.ParseStandard(cronJob.Spec.Schedule)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("Unparseable schedule %q: %v", cronJob.Spec.Schedule, err)
	}

	// for optimization purposes, cheat a bit and start from our last observed run time
	// we could reconstitute this here, but there's not much point, since we've
	// just updated it.
	var earliestTime time.Time
	if cronJob.Status.LastScheduleTime != nil {
		earliestTime = cronJob.Status.LastScheduleTime.Time
	} else {
		earliestTime = cronJob.ObjectMeta.CreationTimestamp.Time
	}

	if cronJob.Spec.StartingDeadlineSeconds != nil {
		// controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*cronJob.Spec.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}
	if earliestTime.After(now) {
		return time.Time{}, sched.Next(now), nil
	}

	starts := 0
	for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		lastMissed = t
		// An object might miss several starts. For example, if
		// controller gets wedged on Friday at 5:01pm when everyone has
		// gone home, and someone comes in on Tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly scheduledJob, should
		// all start running with no further intervention (if the scheduledJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		starts++
		if starts > 100 {
			// We can't get the most recent times so just return an empty slice
			return time.Time{}, time.Time{}, fmt.Errorf("Too many missed start times (> 100). Set or decrease " +
				".spec.startingDeadlineSeconds or check clock skew.")
		}
	}
	return lastMissed, sched.Next(now), nil
}

// constructJobForCronJob gets the batch.CronJob and returns a kbatch.Job
func constructJobForCronJob(cronJob *batch.CronJob, scheduledTime time.Time, scheme *runtime.Scheme) (*kbatch.Job, error) {
	// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
	name := fmt.Sprintf("%s-%d", cronJob.Name, scheduledTime.Unix())

	job := &kbatch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   cronJob.Namespace,
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}
	for k, v := range cronJob.Spec.JobTemplate.Annotations {
		job.Annotations[k] = v
	}
	job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
	for k, v := range cronJob.Spec.JobTemplate.Labels {
		job.Labels[k] = v
	}
	if err := ctrl.SetControllerReference(cronJob, job, scheme); err != nil {
		return nil, err
	}

	return job, nil
}