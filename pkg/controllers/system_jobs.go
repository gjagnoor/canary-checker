package controllers

import (
	"reflect"
	"time"

	v1 "github.com/flanksource/canary-checker/api/v1"
	"github.com/flanksource/canary-checker/pkg"
	"github.com/flanksource/canary-checker/pkg/db"
	"github.com/flanksource/canary-checker/pkg/topology"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/kommons"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/types"
)

var SystemScheduler = cron.New()
var SystemFuncSchedule = cron.New()

type SystemJob struct {
	*kommons.Client
	v1.SystemTemplate
}

func (job SystemJob) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{Name: job.SystemTemplate.Name, Namespace: job.SystemTemplate.Namespace}
}

func (job SystemJob) Run() {
	opts := topology.TopologyRunOptions{
		Client:    job.Client,
		Depth:     10,
		Namespace: job.Namespace,
	}
	systems := topology.Run(opts, job.SystemTemplate)
	for _, system := range systems {
		system.Name = job.SystemTemplate.Name
		system.Namespace = job.SystemTemplate.Namespace
		system.Labels = job.SystemTemplate.Labels
		system.SystemTemplateID = uuid.MustParse(*job.SystemTemplate.Status.PersistedID)
		err := db.PersistSystem(system)
		if err != nil {
			logger.Errorf("error persisting the system: %v", err)
		}
	}
}

func ScheduleSystemFunc(schedule string, fn func()) (interface{}, error) {
	return SystemFuncSchedule.AddFunc(schedule, fn)
}

func SyncSystemsJobs() {
	logger.Infof("Syncing systemTemplate jobs")
	seenEntryIds := map[cron.EntryID]bool{}
	if Kommons == nil {
		var err error
		Kommons, err = pkg.NewKommonsClient()
		if err != nil {
			logger.Warnf("Failed to get kommons client, features that read kubernetes config will fail: %v", err)
		}
	}

	systemTemplates, err := db.GetAllSystemTemplates()
	if err != nil {
		logger.Errorf("Failed to get systemTemplate: %v", err)
		return
	}

	for _, systemTemplate := range systemTemplates {
		entry := findSystemTemplateCronEntry(systemTemplate)
		if entry != nil {
			job := entry.Job.(SystemJob)
			if !reflect.DeepEqual(job.SystemTemplate.Spec, systemTemplate.Spec) {
				logger.Infof("Rescheduling %s/%s systemTemplate with updated specs", systemTemplate.Namespace, systemTemplate.Name)
				SystemScheduler.Remove(entry.ID)
			} else {
				seenEntryIds[entry.ID] = true
				job.SystemTemplate = systemTemplate
				(*entry).Job = job
				continue
			}
		}
		job := SystemJob{
			Client:         Kommons,
			SystemTemplate: systemTemplate,
		}
		if systemTemplate.Spec.GetSchedule() == "@never" {
			continue
		}
		schedule := systemTemplate.Spec.GetSchedule()
		entryID, err := SystemScheduler.AddJob(schedule, job)
		if err != nil {
			logger.Errorf("Failed to schedule systemTemplate %s/%s: %v", systemTemplate.Namespace, systemTemplate.Name, err)
			continue
		} else {
			logger.Infof("Scheduling %s/%s to %s", systemTemplate.Namespace, systemTemplate.Name, systemTemplate.Spec.GetSchedule())
			seenEntryIds[entryID] = true
		}
		entry = findSystemTemplateCronEntry(systemTemplate)
		if entry != nil && time.Until(entry.Next) < 1*time.Hour {
			// run all regular canaries on startup
			job = entry.Job.(SystemJob)
			go job.Run()
		}
	}
	for _, entry := range SystemScheduler.Entries() {
		if !seenEntryIds[entry.ID] {
			logger.Infof("Removing  %s", entry.Job.(SystemJob).SystemTemplate)
			SystemScheduler.Remove(entry.ID)
		}
	}
}

func findSystemTemplateCronEntry(systemTemplate v1.SystemTemplate) *cron.Entry {
	for _, entry := range SystemScheduler.Entries() {
		if entry.Job.(SystemJob).GetPersistedID() == systemTemplate.GetPersistedID() {
			return &entry
		}
	}
	return nil
}