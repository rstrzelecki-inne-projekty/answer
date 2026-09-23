/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package cron

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/translator"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/activity_log"
	"github.com/apache/answer/internal/service/admin_message"
	"github.com/apache/answer/internal/service/content"
	"github.com/apache/answer/internal/service/file_record"
	"github.com/apache/answer/internal/service/service_config"
	"github.com/apache/answer/internal/service/siteinfo_common"
	"github.com/apache/answer/internal/service/user_admin"
	"github.com/apache/answer/pkg/converter"
	"github.com/robfig/cron/v3"
	"github.com/segmentfault/pacman/i18n"
	"github.com/segmentfault/pacman/log"
)

// ScheduledTaskManager scheduled task manager
type ScheduledTaskManager struct {
	siteInfoService   siteinfo_common.SiteInfoCommonService
	questionService   *content.QuestionService
	fileRecordService *file_record.FileRecordService
	userAdminService  *user_admin.UserAdminService
	serviceConfig     *service_config.ServiceConfig
	activityLog       *activity_log.ActivityLogService
	adminMessage      *admin_message.AdminMessageService
}

// NewScheduledTaskManager new scheduled task manager
func NewScheduledTaskManager(
	siteInfoService siteinfo_common.SiteInfoCommonService,
	questionService *content.QuestionService,
	fileRecordService *file_record.FileRecordService,
	userAdminService *user_admin.UserAdminService,
	serviceConfig *service_config.ServiceConfig,
	activityLog *activity_log.ActivityLogService,
	adminMessage *admin_message.AdminMessageService,
) *ScheduledTaskManager {
	manager := &ScheduledTaskManager{
		activityLog:       activityLog,
		adminMessage:      adminMessage,
		siteInfoService:   siteInfoService,
		questionService:   questionService,
		fileRecordService: fileRecordService,
		userAdminService:  userAdminService,
		serviceConfig:     serviceConfig,
	}
	return manager
}

func (s *ScheduledTaskManager) Run() {
	log.Infof("cron job manager start")

	s.questionService.SitemapCron(context.Background())
	c := cron.New()
	_, err := c.AddFunc("0 */1 * * *", func() {
		ctx := context.Background()
		log.Infof("sitemap cron execution")
		s.questionService.SitemapCron(ctx)
	})
	if err != nil {
		log.Error(err)
	}

	_, err = c.AddFunc("0 */1 * * *", func() {
		ctx := context.Background()
		log.Infof("refresh hottest cron execution")
		s.questionService.RefreshHottestCron(ctx)
	})
	if err != nil {
		log.Error(err)
	}

	// [cd] activity log: page views are the bulk of the table → keep ACTIVITY_LOG_PAGEVIEW_DAYS days (default 90, 0 = forever)
	pageViewDays := 90
	if v := os.Getenv("ACTIVITY_LOG_PAGEVIEW_DAYS"); v != "" {
		pageViewDays = converter.StringToInt(v)
	}
	_, err = c.AddFunc("30 3 * * *", func() {
		s.activityLog.CleanupPageViews(context.Background(), pageViewDays)
	})
	if err != nil {
		log.Error(err)
	}

	// [cd] remind the person who asked to mark the solution, once per thread
	_, err = c.AddFunc("0 9 * * *", func() {
		log.Infof("unsolved question reminder cron execution")
		s.remindUnsolvedQuestions(context.Background())
	})
	if err != nil {
		log.Error(err)
	}

	// Check for expired user suspensions every 10 minutes
	_, err = c.AddFunc("*/10 * * * *", func() {
		ctx := context.Background()
		log.Infof("checking expired user suspensions")
		err := s.userAdminService.CheckAndUnsuspendExpiredUsers(ctx)
		if err != nil {
			log.Errorf("failed to check expired user suspensions: %v", err)
		}
	})
	if err != nil {
		log.Error(err)
	}

	if s.serviceConfig.CleanUpUploads {
		log.Infof("clean up uploads cron enabled")

		conf := s.serviceConfig
		_, err = c.AddFunc(fmt.Sprintf("0 */%d * * *", conf.CleanOrphanUploadsPeriodHours), func() {
			log.Infof("clean orphan upload files cron execution")
			s.fileRecordService.CleanOrphanUploadFiles(context.Background())
		})
		if err != nil {
			log.Error(err)
		}

		_, err = c.AddFunc(fmt.Sprintf("0 0 */%d * *", conf.PurgeDeletedFilesPeriodDays), func() {
			log.Infof("purge deleted files cron execution")
			s.fileRecordService.PurgeDeletedFiles(context.Background())
		})
		if err != nil {
			log.Error(err)
		}
	}
	c.Start()
}

// [cd] contest support: a thread with answers and no solution marked earns nobody any points, and
// people simply do not know the button is there. A week after the question was asked its author
// gets one message asking to mark the answer that helped. The reminder is logged, so it is sent
// once per thread even though the job runs every day.
const (
	unsolvedReminderDays   = 7
	unsolvedReminderLimit  = 200
	unsolvedReminderAction = "question.solution_reminder"
	unsolvedReminderWindow = 365 * 24 * time.Hour
)

func (s *ScheduledTaskManager) remindUnsolvedQuestions(ctx context.Context) {
	if s.adminMessage == nil || s.questionService == nil {
		return
	}
	items, err := s.questionService.ListUnsolvedForReminder(ctx, unsolvedReminderDays, unsolvedReminderLimit)
	if err != nil {
		log.Error(err)
		return
	}
	if len(items) == 0 {
		return
	}
	reminded := map[string]bool{}
	if s.activityLog != nil {
		if done, err := s.activityLog.ObjectsWithAction(ctx, unsolvedReminderAction,
			time.Now().Add(-unsolvedReminderWindow)); err == nil {
			reminded = done
		} else {
			log.Error(err)
			return // without the log the reminder would go out again every day
		}
	}
	lang := i18n.DefaultLanguage
	if siteInterface, err := s.siteInfoService.GetSiteInterface(ctx); err == nil && siteInterface != nil {
		lang = i18n.Language(siteInterface.Language)
	}
	title := translator.Tr(lang, "backend.reminder.solution.title")
	bodyTpl := translator.Tr(lang, "backend.reminder.solution.body")
	sent := 0
	for _, item := range items {
		if reminded[item.QuestionID] {
			continue
		}
		body := fmt.Sprintf(bodyTpl, item.Title)
		if err := s.adminMessage.SendSystemMessage(ctx, item.UserID, title, body, item.QuestionID); err != nil {
			log.Error(err)
			continue
		}
		s.activityLog.Log(ctx, &entity.ActivityLog{UserID: "0", Action: unsolvedReminderAction,
			ObjectType: constant.QuestionObjectType, ObjectID: item.QuestionID,
			QuestionID: item.QuestionID, TargetUserID: item.UserID})
		sent++
	}
	if sent > 0 {
		log.Infof("unsolved question reminder: %d sent", sent)
	}
}
