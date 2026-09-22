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

package user_prestige

import (
	"context"
	"errors"
	"testing"

	"github.com/apache/answer/internal/entity"
	"github.com/stretchr/testify/require"
)

const (
	userAnna = "10000000000000001"
	userEwa  = "10000000000000002"

	badgeKapitan   = "10090000001527725"
	badgeSierzant  = "10090000001906889"
	badgeWizytowka = "10090000000000006"
	badgeTrafiona  = "10090000000000014"
	badgeZolta     = "10090000001178695"
	badgeUkryta    = "10090000000000099"
)

type fakeGroupRepo struct {
	groups []*entity.BadgeGroup
	err    error
	calls  int
}

func (f *fakeGroupRepo) ListGroups(ctx context.Context) ([]*entity.BadgeGroup, error) {
	f.calls++
	return f.groups, f.err
}

type fakeBadgeRepo struct {
	badges []*entity.Badge
	err    error
	calls  int
}

func (f *fakeBadgeRepo) ListActivated(ctx context.Context, page int, pageSize int) ([]*entity.Badge, int64, error) {
	f.calls++
	return f.badges, int64(len(f.badges)), f.err
}

type fakeAwardRepo struct {
	counts []*entity.UserBadgeEarnedCount
	err    error
	calls  int
	gotIDs []string
}

func (f *fakeAwardRepo) BatchUserEarnedCount(ctx context.Context, userIDs []string) (
	[]*entity.UserBadgeEarnedCount, error) {
	f.calls++
	f.gotIDs = userIDs
	return f.counts, f.err
}

func testBadges() []*entity.Badge {
	return []*entity.Badge{
		{ID: badgeSierzant, Name: "badge.aoa.sierzant.name", Icon: "/b/sierzant.svg", BadgeGroupID: 2,
			Level: entity.BadgeLevelSilver, Param: `{"amount":"15"}`},
		{ID: badgeKapitan, Name: "badge.aoa.kapitan.name", Icon: "/b/kapitan.svg", BadgeGroupID: 2,
			Level: entity.BadgeLevelGold, Param: `{"amount":"80"}`},
		{ID: badgeWizytowka, Name: "badge.aoa.wizytowka.name", Icon: "/b/wizytowka.svg", BadgeGroupID: 1,
			Level: entity.BadgeLevelBronze},
		{ID: badgeTrafiona, Name: "badge.aoa.trafiona-odpowiedz.name", Icon: "/b/trafiona.svg", BadgeGroupID: 3,
			Level: entity.BadgeLevelBronze, Param: `{"amount":"3"}`},
		{ID: badgeZolta, Name: "badge.aoa.zolta-kartka.name", Icon: "/b/zolta.svg", BadgeGroupID: 4,
			Level: entity.BadgeLevelBronze},
	}
}

func testGroups() []*entity.BadgeGroup {
	return []*entity.BadgeGroup{
		{ID: "1", Name: "badge.aoa_groups.start.name"},
		{ID: "2", Name: "badge.aoa_groups.stopnie.name"},
		{ID: "3", Name: "badge.aoa_groups.jakosc.name"},
		{ID: "4", Name: "badge.aoa_groups.rodo.name"},
	}
}

func newTestService(counts []*entity.UserBadgeEarnedCount) (
	*UserPrestigeService, *fakeBadgeRepo, *fakeGroupRepo, *fakeAwardRepo) {
	badgeRepo := &fakeBadgeRepo{badges: testBadges()}
	groupRepo := &fakeGroupRepo{groups: testGroups()}
	awardRepo := &fakeAwardRepo{counts: counts}
	return NewUserPrestigeService(badgeRepo, groupRepo, awardRepo), badgeRepo, groupRepo, awardRepo
}

func TestBatchGet_HighestRankWinsAndIsExcludedFromBadgeCount(t *testing.T) {
	svc, _, _, _ := newTestService([]*entity.UserBadgeEarnedCount{
		{UserID: userAnna, BadgeID: badgeSierzant, EarnedCount: 1},
		{UserID: userAnna, BadgeID: badgeKapitan, EarnedCount: 1},
		{UserID: userAnna, BadgeID: badgeWizytowka, EarnedCount: 1},
		{UserID: userAnna, BadgeID: badgeTrafiona, EarnedCount: 3},
		{UserID: userAnna, BadgeID: badgeZolta, EarnedCount: 2},
		{UserID: userAnna, BadgeID: badgeUkryta, EarnedCount: 5},
	})

	got, err := svc.BatchGet(context.TODO(), []string{userAnna})
	require.NoError(t, err)
	require.Len(t, got, 1)

	p := got[userAnna]
	require.NotNil(t, p)
	require.Equal(t, badgeKapitan, p.RankBadgeID)
	require.Equal(t, "badge.aoa.kapitan.name", p.RankBadgeName)
	require.Equal(t, "/b/kapitan.svg", p.RankBadgeIcon)
	require.Equal(t, 3, p.RankBadgeLevel)
	// wizytówka (1) + trafiona odpowiedź (3); stopnie, żółta kartka i odznaka spoza bazy nie liczą się
	require.Equal(t, 4, p.BadgeCount)
	require.Equal(t, 2, p.YellowCards)
}

func TestBatchGet_UserWithoutRankKeepsEmptyRankFields(t *testing.T) {
	svc, _, _, _ := newTestService([]*entity.UserBadgeEarnedCount{
		{UserID: userEwa, BadgeID: badgeWizytowka, EarnedCount: 1},
	})

	got, err := svc.BatchGet(context.TODO(), []string{userEwa})
	require.NoError(t, err)

	p := got[userEwa]
	require.NotNil(t, p)
	require.Empty(t, p.RankBadgeID)
	require.Empty(t, p.RankBadgeName)
	require.Equal(t, 0, p.RankBadgeLevel)
	require.Equal(t, 1, p.BadgeCount)
	require.Equal(t, 0, p.YellowCards)
}

func TestBatchGet_UserWithoutAnyAwardIsAbsentFromResult(t *testing.T) {
	svc, _, _, _ := newTestService(nil)

	got, err := svc.BatchGet(context.TODO(), []string{userAnna, userEwa})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestBatchGet_EmptyInputDoesNotTouchRepositories(t *testing.T) {
	svc, badgeRepo, groupRepo, awardRepo := newTestService(nil)

	got, err := svc.BatchGet(context.TODO(), []string{"", "0"})
	require.NoError(t, err)
	require.Empty(t, got)
	require.Zero(t, badgeRepo.calls)
	require.Zero(t, groupRepo.calls)
	require.Zero(t, awardRepo.calls)
}

func TestBatchGet_BadgeMetadataIsCachedBetweenCalls(t *testing.T) {
	svc, badgeRepo, groupRepo, awardRepo := newTestService([]*entity.UserBadgeEarnedCount{
		{UserID: userAnna, BadgeID: badgeWizytowka, EarnedCount: 1},
	})

	_, err := svc.BatchGet(context.TODO(), []string{userAnna})
	require.NoError(t, err)
	_, err = svc.BatchGet(context.TODO(), []string{userAnna})
	require.NoError(t, err)

	require.Equal(t, 1, badgeRepo.calls)
	require.Equal(t, 1, groupRepo.calls)
	require.Equal(t, 2, awardRepo.calls, "liczniki przyznań mają być świeże przy każdym wywołaniu")
}

func TestBatchGet_RepositoryErrorIsReturned(t *testing.T) {
	badgeRepo := &fakeBadgeRepo{badges: testBadges()}
	groupRepo := &fakeGroupRepo{groups: testGroups()}
	awardRepo := &fakeAwardRepo{err: errors.New("baza padła")}
	svc := NewUserPrestigeService(badgeRepo, groupRepo, awardRepo)

	got, err := svc.BatchGet(context.TODO(), []string{userAnna})
	require.Error(t, err)
	require.Empty(t, got)
}

func TestBatchGet_MissingRankGroupDegradesToCountsOnly(t *testing.T) {
	badgeRepo := &fakeBadgeRepo{badges: testBadges()}
	groupRepo := &fakeGroupRepo{groups: []*entity.BadgeGroup{{ID: "1", Name: "badge.group.other"}}}
	awardRepo := &fakeAwardRepo{counts: []*entity.UserBadgeEarnedCount{
		{UserID: userAnna, BadgeID: badgeKapitan, EarnedCount: 1},
		{UserID: userAnna, BadgeID: badgeWizytowka, EarnedCount: 1},
	}}
	svc := NewUserPrestigeService(badgeRepo, groupRepo, awardRepo)

	got, err := svc.BatchGet(context.TODO(), []string{userAnna})
	require.NoError(t, err)

	p := got[userAnna]
	require.NotNil(t, p)
	require.Empty(t, p.RankBadgeID, "bez grupy stopni nie ma czego pokazać jako pagon")
	require.Equal(t, 2, p.BadgeCount, "odznaki stopni wliczają się wtedy do zwykłego licznika")
}
