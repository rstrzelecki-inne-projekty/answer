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

package repo_test

// [cd] 27: a single award badge can be awarded again by hand (each award has its own key),
// while badge rules keep awarding it only once.

import (
	"context"
	"testing"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/repo/badge_award"
	"github.com/apache/answer/internal/repo/unique"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_badgeAwardRepo_AwardBadgeForUser_repeatable(t *testing.T) {
	ctx := context.TODO()
	repo := badge_award.NewBadgeAwardRepo(testDataSource, unique.NewUniqueIDRepo(testDataSource))
	single := &entity.Badge{}
	exist, err := testDataSource.DB.Context(ctx).Where("single = ?", entity.BadgeSingleAward).Get(single)
	require.NoError(t, err)
	require.True(t, exist, "seed data has no single award badge")
	const userID = "10010000000009901"
	newAward := func(key string) *entity.BadgeAward {
		return &entity.BadgeAward{UserID: userID, BadgeID: single.ID, AwardKey: key, BadgeGroupID: single.BadgeGroupID,
			IsBadgeDeleted: entity.IsBadgeNotDeleted}
	}
	t.Cleanup(func() { _, _ = testDataSource.DB.Where("user_id = ?", userID).Delete(&entity.BadgeAward{}) })

	// badge rule: once per user
	require.NoError(t, repo.AwardBadgeForUser(ctx, newAward("10030000000000001"), entity.BadgeSingleAward))
	assert.Error(t, repo.AwardBadgeForUser(ctx, newAward("10030000000000002"), entity.BadgeSingleAward))

	// manual awards: checked by key, so another key is another award
	require.NoError(t, repo.AwardBadgeForUser(ctx, newAward("admin-1"), entity.BadgeMultiAward))
	require.NoError(t, repo.AwardBadgeForUser(ctx, newAward("admin-2"), entity.BadgeMultiAward))
	assert.Error(t, repo.AwardBadgeForUser(ctx, newAward("admin-2"), entity.BadgeMultiAward))
	assert.Equal(t, int64(3), repo.CountByUserIdAndBadgeId(ctx, userID, single.ID))
}
