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

package content

// [cd] Tests for vote activities on comments (fork patch #4, AA-35): comment votes produce the same
// vote_*/voted_* activity pairs as questions and answers, and missing config keys are skipped.

import (
	"context"
	"errors"
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/config"
)

type fakeConfigRepo struct {
	config.ConfigRepo
	byKey map[string]*entity.Config
}

func (f *fakeConfigRepo) GetConfigByKey(_ context.Context, key string) (*entity.Config, error) {
	if c, ok := f.byKey[key]; ok {
		return c, nil
	}
	return nil, errors.New("config not found: " + key)
}

func voteServiceWithConfig(cfg map[string]*entity.Config) *VoteService {
	return &VoteService{configService: config.NewConfigService(&fakeConfigRepo{byKey: cfg})}
}

var seededConfig = map[string]*entity.Config{
	"comment.vote_up":    {ID: 900, Key: "comment.vote_up", Value: "0"},
	"comment.vote_down":  {ID: 901, Key: "comment.vote_down", Value: "-1"},
	"comment.voted_up":   {ID: 902, Key: "comment.voted_up", Value: "5"},
	"comment.voted_down": {ID: 903, Key: "comment.voted_down", Value: "-2"},
}

func TestGetActivitiesCommentVotes(t *testing.T) {
	op := &schema.VoteOperationInfo{ObjectType: constant.CommentObjectType, ObjectCreatorUserID: "author", OperatingUserID: "voter"}

	t.Run("vote up: voter activity + author reputation", func(t *testing.T) {
		vs := voteServiceWithConfig(seededConfig)
		op.VoteUp, op.VoteDown = true, false
		acts := vs.getActivities(context.Background(), op)
		if len(acts) != 2 {
			t.Fatalf("want 2 activities, got %d", len(acts))
		}
		if acts[0].ActivityType != 900 || acts[0].ActivityUserID != "voter" || acts[0].TriggerUserID != "0" || acts[0].Rank != 0 {
			t.Errorf("vote_up activity = %+v", acts[0])
		}
		if acts[1].ActivityType != 902 || acts[1].ActivityUserID != "author" || acts[1].TriggerUserID != "voter" || acts[1].Rank != 5 {
			t.Errorf("voted_up activity = %+v", acts[1])
		}
	})

	t.Run("vote down: negative reputation for both", func(t *testing.T) {
		vs := voteServiceWithConfig(seededConfig)
		op.VoteUp, op.VoteDown = false, true
		acts := vs.getActivities(context.Background(), op)
		if len(acts) != 2 || acts[0].ActivityType != 901 || acts[0].Rank != -1 || acts[1].ActivityType != 903 || acts[1].Rank != -2 || acts[1].ActivityUserID != "author" {
			t.Fatalf("unexpected activities: %+v", acts)
		}
	})

	t.Run("upstream database without the new keys: only comment.vote_up remains", func(t *testing.T) {
		vs := voteServiceWithConfig(map[string]*entity.Config{"comment.vote_up": seededConfig["comment.vote_up"]})
		op.VoteUp, op.VoteDown = true, false
		if acts := vs.getActivities(context.Background(), op); len(acts) != 1 || acts[0].ActivityType != 900 {
			t.Fatalf("want only vote_up, got %+v", acts)
		}
		op.VoteUp, op.VoteDown = false, true
		if acts := vs.getActivities(context.Background(), op); len(acts) != 0 {
			t.Fatalf("down vote must be a no-op without config, got %+v", acts)
		}
	})
}
