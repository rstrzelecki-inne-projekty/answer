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

package badge

// [cd] Tests for the manual badge award/revoke admin actions (fork patch #3, AA-34).
// Repositories are hand-written fakes: unused interface methods are left to the embedded nil interface.
// IDs must be >= 10^16: anything smaller is treated as a short ID by uid.DeShortID and decoded.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/mock"
	"github.com/apache/answer/internal/service/noticequeue"
	usercommon "github.com/apache/answer/internal/service/user_common"
	"github.com/apache/answer/pkg/uid"
	pacmanErrors "github.com/segmentfault/pacman/errors"
	"go.uber.org/mock/gomock"
)

type fakeBadgeRepo struct {
	BadgeRepo
	badges map[string]*entity.Badge
}

func (f *fakeBadgeRepo) GetByID(_ context.Context, id string) (*entity.Badge, bool, error) {
	b, ok := f.badges[id]
	return b, ok, nil
}

type fakeBadgeAwardRepo struct {
	BadgeAwardRepo
	awarded  map[string]bool // badgeID|userID|awardKey
	awards   []*entity.BadgeAward
	revokeOK bool
	revoked  []string // "userID|badgeID|awardKey"
}

func (f *fakeBadgeAwardRepo) CheckIsAward(_ context.Context, badgeID, userID, awardKey string, single int8) (bool, error) {
	if single == entity.BadgeSingleAward {
		for k := range f.awarded {
			if len(k) > len(badgeID+"|"+userID) && k[:len(badgeID+"|"+userID)] == badgeID+"|"+userID {
				return true, nil
			}
		}
		return false, nil
	}
	return f.awarded[badgeID+"|"+userID+"|"+awardKey], nil
}

func (f *fakeBadgeAwardRepo) AwardBadgeForUser(_ context.Context, a *entity.BadgeAward) error {
	a.ID = "award-1"
	f.awards = append(f.awards, a)
	return nil
}

func (f *fakeBadgeAwardRepo) RevokeBadgeAward(_ context.Context, userID, badgeID, awardKey string) (bool, error) {
	f.revoked = append(f.revoked, userID+"|"+badgeID+"|"+awardKey)
	return f.revokeOK, nil
}

type fakeUserRepo struct {
	usercommon.UserRepo
	users map[string]*entity.User
}

func (f *fakeUserRepo) GetByUsername(_ context.Context, username string) (*entity.User, bool, error) {
	u, ok := f.users[username]
	if !ok {
		return &entity.User{}, false, nil
	}
	return u, true, nil
}

func newTestService(t *testing.T, badgeRepo *fakeBadgeRepo, awardRepo *fakeBadgeAwardRepo) *BadgeAwardService {
	t.Helper()
	ctrl := gomock.NewController(t)
	siteInfo := mock.NewMockSiteInfoCommonService(ctrl)
	siteInfo.EXPECT().FormatAvatar(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&schema.AvatarInfo{}).AnyTimes()
	userCommon := usercommon.NewUserCommon(&fakeUserRepo{users: map[string]*entity.User{
		"alice": {ID: "u-alice", Username: "alice"},
	}}, nil, nil, siteInfo)
	return NewBadgeAwardService(awardRepo, badgeRepo, userCommon, nil, noticequeue.NewService())
}

func reasonOf(err error) string {
	var e *pacmanErrors.Error
	if errors.As(err, &e) {
		return e.Reason
	}
	return ""
}

func TestAdminAward(t *testing.T) {
	badges := map[string]*entity.Badge{
		"10030000000000100": {ID: "10030000000000100", Name: "Pomocna dłoń", Status: entity.BadgeStatusActive, Single: entity.BadgeSingleAward, BadgeGroupID: 1},
		"10030000000000200": {ID: "10030000000000200", Name: "Miesiąc bez flag", Status: entity.BadgeStatusActive, Single: entity.BadgeMultiAward, BadgeGroupID: 1},
		"10030000000000300": {ID: "10030000000000300", Name: "Wyłączona", Status: entity.BadgeStatusInactive, Single: entity.BadgeSingleAward},
	}

	t.Run("unknown user", func(t *testing.T) {
		svc := newTestService(t, &fakeBadgeRepo{badges: badges}, &fakeBadgeAwardRepo{awarded: map[string]bool{}})
		err := svc.AdminAward(context.Background(), &schema.AdminAwardBadgeReq{BadgeID: "10030000000000100", Username: "nobody"})
		if reasonOf(err) != reason.UserNotFound {
			t.Fatalf("want %s, got %v", reason.UserNotFound, err)
		}
	})

	t.Run("inactive badge", func(t *testing.T) {
		svc := newTestService(t, &fakeBadgeRepo{badges: badges}, &fakeBadgeAwardRepo{awarded: map[string]bool{}})
		err := svc.AdminAward(context.Background(), &schema.AdminAwardBadgeReq{BadgeID: "10030000000000300", Username: "alice"})
		if reasonOf(err) != reason.BadgeObjectNotFound {
			t.Fatalf("want %s, got %v", reason.BadgeObjectNotFound, err)
		}
	})

	t.Run("single badge awarded once, short id accepted, key admin", func(t *testing.T) {
		awardRepo := &fakeBadgeAwardRepo{awarded: map[string]bool{}}
		svc := newTestService(t, &fakeBadgeRepo{badges: badges}, awardRepo)
		if err := svc.AdminAward(context.Background(), &schema.AdminAwardBadgeReq{BadgeID: uid.EnShortID("10030000000000100"), Username: "alice"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(awardRepo.awards) != 1 || awardRepo.awards[0].BadgeID != "10030000000000100" || awardRepo.awards[0].UserID != "u-alice" || awardRepo.awards[0].AwardKey != "admin" {
			t.Fatalf("unexpected award: %+v", awardRepo.awards)
		}
		// second time: explicit error instead of the silent no-op of Award()
		awardRepo.awarded["10030000000000100|u-alice|admin"] = true
		err := svc.AdminAward(context.Background(), &schema.AdminAwardBadgeReq{BadgeID: "10030000000000100", Username: "alice"})
		if reasonOf(err) != reason.BadgeAlreadyAwarded {
			t.Fatalf("want %s, got %v", reason.BadgeAlreadyAwarded, err)
		}
	})

	t.Run("multi-award badge defaults key to today, explicit key kept", func(t *testing.T) {
		awardRepo := &fakeBadgeAwardRepo{awarded: map[string]bool{}}
		svc := newTestService(t, &fakeBadgeRepo{badges: badges}, awardRepo)
		if err := svc.AdminAward(context.Background(), &schema.AdminAwardBadgeReq{BadgeID: "10030000000000200", Username: "alice"}); err != nil {
			t.Fatal(err)
		}
		if err := svc.AdminAward(context.Background(), &schema.AdminAwardBadgeReq{BadgeID: "10030000000000200", Username: "alice", AwardKey: "2026-09"}); err != nil {
			t.Fatal(err)
		}
		if got := awardRepo.awards[0].AwardKey; got != time.Now().Format("2006-01-02") {
			t.Fatalf("default award key = %q", got)
		}
		if got := awardRepo.awards[1].AwardKey; got != "2026-09" {
			t.Fatalf("explicit award key = %q", got)
		}
	})
}

func TestAdminRevoke(t *testing.T) {
	badges := map[string]*entity.Badge{"10030000000000100": {ID: "10030000000000100", Status: entity.BadgeStatusActive, Single: entity.BadgeSingleAward}}

	t.Run("nothing to revoke", func(t *testing.T) {
		svc := newTestService(t, &fakeBadgeRepo{badges: badges}, &fakeBadgeAwardRepo{revokeOK: false})
		err := svc.AdminRevoke(context.Background(), &schema.AdminRevokeBadgeReq{BadgeID: "10030000000000100", Username: "alice"})
		if reasonOf(err) != reason.BadgeAwardNotFound {
			t.Fatalf("want %s, got %v", reason.BadgeAwardNotFound, err)
		}
	})

	t.Run("revokes newest award by default, specific key when given", func(t *testing.T) {
		awardRepo := &fakeBadgeAwardRepo{revokeOK: true}
		svc := newTestService(t, &fakeBadgeRepo{badges: badges}, awardRepo)
		if err := svc.AdminRevoke(context.Background(), &schema.AdminRevokeBadgeReq{BadgeID: uid.EnShortID("10030000000000100"), Username: "alice"}); err != nil {
			t.Fatal(err)
		}
		if err := svc.AdminRevoke(context.Background(), &schema.AdminRevokeBadgeReq{BadgeID: "10030000000000100", Username: "alice", AwardKey: "2026-09"}); err != nil {
			t.Fatal(err)
		}
		if len(awardRepo.revoked) != 2 || awardRepo.revoked[0] != "u-alice|10030000000000100|" || awardRepo.revoked[1] != "u-alice|10030000000000100|2026-09" {
			t.Fatalf("unexpected revoke calls: %v", awardRepo.revoked)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		svc := newTestService(t, &fakeBadgeRepo{badges: badges}, &fakeBadgeAwardRepo{revokeOK: true})
		err := svc.AdminRevoke(context.Background(), &schema.AdminRevokeBadgeReq{BadgeID: "10030000000000100", Username: "nobody"})
		if reasonOf(err) != reason.UserNotFound {
			t.Fatalf("want %s, got %v", reason.UserNotFound, err)
		}
	})
}
