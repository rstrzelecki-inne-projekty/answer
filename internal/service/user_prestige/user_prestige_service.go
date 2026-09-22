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

// Package user_prestige [cd] collects the badges shown next to a user's avatar:
// the rank insignia, the number of other badges and the yellow cards.
package user_prestige

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/translator"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
)

const (
	// name of the badge group holding the ranks; the column stores an i18n key
	defaultRankGroupName = "badge.aoa_groups.stopnie.name"
	// name of the yellow card badge; the column stores an i18n key
	defaultYellowCardBadgeName = "badge.aoa.zolta-kartka.name"
	// badge definitions change only on deployment, award counts stay fresh
	badgeMetaTTL = 10 * time.Minute
	// there are ~50 badge definitions, one page is enough
	badgeMetaPageSize = 1000
)

// BadgeMetaRepo reads the badge definitions
type BadgeMetaRepo interface {
	ListActivated(ctx context.Context, page int, pageSize int) (badges []*entity.Badge, total int64, err error)
}

// BadgeGroupMetaRepo reads the badge groups
type BadgeGroupMetaRepo interface {
	ListGroups(ctx context.Context) (groups []*entity.BadgeGroup, err error)
}

// AwardCountRepo counts the badges awarded to a set of users
type AwardCountRepo interface {
	BatchUserEarnedCount(ctx context.Context, userIDs []string) (counts []*entity.UserBadgeEarnedCount, err error)
}

// badgeMeta is what the prestige card needs to know about one badge
type badgeMeta struct {
	name         string
	icon         string
	level        int
	rankAmount   int64
	isRank       bool
	isYellowCard bool
}

type UserPrestigeService struct {
	badgeRepo      BadgeMetaRepo
	groupRepo      BadgeGroupMetaRepo
	awardRepo      AwardCountRepo
	rankGroupName  string
	yellowCardName string

	mu     sync.RWMutex
	meta   map[string]*badgeMeta
	metaAt time.Time
}

func NewUserPrestigeService(
	badgeRepo BadgeMetaRepo,
	groupRepo BadgeGroupMetaRepo,
	awardRepo AwardCountRepo,
) *UserPrestigeService {
	return &UserPrestigeService{
		badgeRepo:      badgeRepo,
		groupRepo:      groupRepo,
		awardRepo:      awardRepo,
		rankGroupName:  envOr("PRESTIGE_RANK_GROUP_NAME", defaultRankGroupName),
		yellowCardName: envOr("PRESTIGE_YELLOW_CARD_BADGE_NAME", defaultYellowCardBadgeName),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// BatchGet returns the prestige card for every user that earned at least one badge.
// Users without any award are absent from the map, so the UI renders them as before.
func (s *UserPrestigeService) BatchGet(ctx context.Context, userIDs []string) (
	map[string]*schema.UserPrestige, error) {
	result := make(map[string]*schema.UserPrestige)

	wanted := make([]string, 0, len(userIDs))
	seen := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		if id == "" || id == "0" || seen[id] {
			continue
		}
		seen[id] = true
		wanted = append(wanted, id)
	}
	if len(wanted) == 0 {
		return result, nil
	}

	counts, err := s.awardRepo.BatchUserEarnedCount(ctx, wanted)
	if err != nil {
		return result, err
	}
	if len(counts) == 0 {
		return result, nil
	}

	meta, err := s.badgeMeta(ctx)
	if err != nil {
		return result, err
	}

	// the rank badge with the highest threshold wins; ties go to the higher level, then the higher id
	best := make(map[string]*badgeMeta)
	bestID := make(map[string]string)
	for _, c := range counts {
		m, ok := meta[c.BadgeID]
		if !ok {
			continue
		}
		p := result[c.UserID]
		if p == nil {
			p = &schema.UserPrestige{}
			result[c.UserID] = p
		}
		switch {
		case m.isYellowCard:
			p.YellowCards += int(c.EarnedCount)
		case m.isRank:
			if cur := best[c.UserID]; cur == nil || betterRank(m, c.BadgeID, cur, bestID[c.UserID]) {
				best[c.UserID] = m
				bestID[c.UserID] = c.BadgeID
			}
		default:
			p.BadgeCount += int(c.EarnedCount)
		}
	}

	for userID, m := range best {
		p := result[userID]
		p.RankBadgeID = bestID[userID]
		// badge names are stored as i18n keys, the same way the badge pages translate them
		p.RankBadgeName = translator.Tr(handler.GetLangByCtx(ctx), m.name)
		p.RankBadgeIcon = m.icon
		p.RankBadgeLevel = m.level
		p.RankBadgeAmount = int(m.rankAmount)
	}
	return result, nil
}

func betterRank(candidate *badgeMeta, candidateID string, current *badgeMeta, currentID string) bool {
	if candidate.rankAmount != current.rankAmount {
		return candidate.rankAmount > current.rankAmount
	}
	if candidate.level != current.level {
		return candidate.level > current.level
	}
	return candidateID > currentID
}

// badgeMeta reads the badge definitions, cached for badgeMetaTTL
func (s *UserPrestigeService) badgeMeta(ctx context.Context) (map[string]*badgeMeta, error) {
	s.mu.RLock()
	if s.meta != nil && time.Since(s.metaAt) < badgeMetaTTL {
		cached := s.meta
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	groups, err := s.groupRepo.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	// zero means the community has no rank group, then every badge counts as an ordinary one
	var rankGroupID int64
	for _, g := range groups {
		if g.Name != s.rankGroupName {
			continue
		}
		if id, convErr := strconv.ParseInt(g.ID, 10, 64); convErr == nil {
			rankGroupID = id
		}
		break
	}

	badges, _, err := s.badgeRepo.ListActivated(ctx, 1, badgeMetaPageSize)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]*badgeMeta, len(badges))
	for _, b := range badges {
		meta[b.ID] = &badgeMeta{
			name:         b.Name,
			icon:         b.Icon,
			level:        int(b.Level),
			rankAmount:   b.GetIntParam("amount"),
			isRank:       rankGroupID != 0 && b.BadgeGroupID == rankGroupID,
			isYellowCard: b.Name == s.yellowCardName,
		}
	}

	s.mu.Lock()
	s.meta, s.metaAt = meta, time.Now()
	s.mu.Unlock()
	return meta, nil
}
