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

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/answer/internal/service/activity_type"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apache/answer/internal/service/eventqueue"
	"github.com/apache/answer/pkg/token"

	"github.com/apache/answer/internal/base/constant"
	questioncommon "github.com/apache/answer/internal/service/question_common"
	"github.com/apache/answer/internal/service/user_notification_config"

	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/base/translator"
	"github.com/apache/answer/internal/base/validator"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/activity"
	"github.com/apache/answer/internal/service/activity_common"
	"github.com/apache/answer/internal/service/activity_log"
	"github.com/apache/answer/internal/service/auth"
	"github.com/apache/answer/internal/service/export"
	"github.com/apache/answer/internal/service/file_record"
	"github.com/apache/answer/internal/service/role"
	"github.com/apache/answer/internal/service/siteinfo_common"
	usercommon "github.com/apache/answer/internal/service/user_common"
	"github.com/apache/answer/internal/service/user_external_login"
	"github.com/apache/answer/pkg/checker"
	"github.com/apache/answer/plugin"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
	"golang.org/x/crypto/bcrypt"
)

// UserService user service
type UserService struct {
	userCommonService             *usercommon.UserCommon
	userRepo                      usercommon.UserRepo
	userActivity                  activity.UserActiveActivityRepo
	activityRepo                  activity_common.ActivityRepo
	emailService                  *export.EmailService
	authService                   *auth.AuthService
	siteInfoService               siteinfo_common.SiteInfoCommonService
	userRoleService               *role.UserRoleRelService
	userExternalLoginService      *user_external_login.UserExternalLoginService
	userNotificationConfigRepo    user_notification_config.UserNotificationConfigRepo
	userNotificationConfigService *user_notification_config.UserNotificationConfigService
	questionService               *questioncommon.QuestionCommon
	eventQueueService             eventqueue.Service
	// [cd] the last-week rankings on the users page reuse the activity log aggregation
	activityLogService *activity_log.ActivityLogService
	fileRecordService  *file_record.FileRecordService
}

func NewUserService(userRepo usercommon.UserRepo,
	userActivity activity.UserActiveActivityRepo,
	activityRepo activity_common.ActivityRepo,
	emailService *export.EmailService,
	authService *auth.AuthService,
	siteInfoService siteinfo_common.SiteInfoCommonService,
	userRoleService *role.UserRoleRelService,
	userCommonService *usercommon.UserCommon,
	userExternalLoginService *user_external_login.UserExternalLoginService,
	userNotificationConfigRepo user_notification_config.UserNotificationConfigRepo,
	userNotificationConfigService *user_notification_config.UserNotificationConfigService,
	questionService *questioncommon.QuestionCommon,
	eventQueueService eventqueue.Service,
	activityLogService *activity_log.ActivityLogService,
	fileRecordService *file_record.FileRecordService,
) *UserService {
	return &UserService{
		userCommonService:             userCommonService,
		userRepo:                      userRepo,
		userActivity:                  userActivity,
		activityRepo:                  activityRepo,
		emailService:                  emailService,
		authService:                   authService,
		siteInfoService:               siteInfoService,
		userRoleService:               userRoleService,
		userExternalLoginService:      userExternalLoginService,
		userNotificationConfigRepo:    userNotificationConfigRepo,
		userNotificationConfigService: userNotificationConfigService,
		questionService:               questionService,
		eventQueueService:             eventQueueService,
		activityLogService:            activityLogService,
		fileRecordService:             fileRecordService,
	}
}

// GetUserInfoByUserID get user info by user id
func (us *UserService) GetUserInfoByUserID(ctx context.Context, token, userID string) (
	resp *schema.GetCurrentLoginUserInfoResp, err error) {
	userInfo, exist, err := us.userRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	if userInfo.Status == entity.UserStatusDeleted {
		return nil, errors.Unauthorized(reason.UnauthorizedError)
	}

	resp = &schema.GetCurrentLoginUserInfoResp{}
	resp.ConvertFromUserEntity(userInfo)
	resp.RoleID, err = us.userRoleService.GetUserRole(ctx, userInfo.ID)
	if err != nil {
		log.Error(err)
	}
	resp.Avatar = us.siteInfoService.FormatAvatar(ctx, userInfo.Avatar, userInfo.EMail, userInfo.Status)
	resp.AccessToken = token
	resp.HavePassword = len(userInfo.Pass) > 0
	return resp, nil
}

func (us *UserService) GetOtherUserInfoByUsername(ctx context.Context, req *schema.GetOtherUserInfoByUsernameReq) (
	resp *schema.GetOtherUserInfoByUsernameResp, err error) {
	userInfo, exist, err := us.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.NotFound(reason.UserNotFound)
	}
	resp = &schema.GetOtherUserInfoByUsernameResp{}
	resp.ConvertFromUserEntityWithLang(ctx, userInfo)
	resp.Avatar = us.siteInfoService.FormatAvatar(ctx, userInfo.Avatar, userInfo.EMail, userInfo.Status).GetURL()

	// Only the user himself and the administrator can see the hidden questions
	questionCount, err := us.questionService.GetPersonalUserQuestionCount(ctx, req.UserID, userInfo.ID, req.IsAdmin)
	if err != nil {
		return nil, err
	}
	resp.QuestionCount = int(questionCount)
	return resp, nil
}

// EmailLogin email login
func (us *UserService) EmailLogin(ctx context.Context, req *schema.UserEmailLoginReq) (resp *schema.UserLoginResp, err error) {
	siteLogin, err := us.siteInfoService.GetSiteLogin(ctx)
	if err != nil {
		return nil, err
	}
	if !siteLogin.AllowPasswordLogin {
		return nil, errors.BadRequest(reason.NotAllowedLoginViaPassword)
	}
	userInfo, exist, err := us.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if !exist || userInfo.Status == entity.UserStatusDeleted {
		return nil, errors.BadRequest(reason.EmailOrPasswordWrong)
	}
	if !us.verifyPassword(ctx, req.Pass, userInfo.Pass) {
		return nil, errors.BadRequest(reason.EmailOrPasswordWrong)
	}
	ok, externalID, err := us.userExternalLoginService.CheckUserStatusInUserCenter(ctx, userInfo.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.BadRequest(reason.EmailOrPasswordWrong)
	}

	err = us.userRepo.UpdateLastLoginDate(ctx, userInfo.ID)
	if err != nil {
		log.Errorf("update last login data failed, err: %v", err)
	}

	roleID, err := us.userRoleService.GetUserRole(ctx, userInfo.ID)
	if err != nil {
		log.Error(err)
	}

	resp = &schema.UserLoginResp{}
	resp.ConvertFromUserEntity(userInfo)
	resp.Avatar = us.siteInfoService.FormatAvatar(ctx, userInfo.Avatar, userInfo.EMail, userInfo.Status).GetURL()
	userCacheInfo := &entity.UserCacheInfo{
		UserID:      userInfo.ID,
		EmailStatus: userInfo.MailStatus,
		UserStatus:  userInfo.Status,
		RoleID:      roleID,
		ExternalID:  externalID,
	}
	resp.AccessToken, resp.VisitToken, err = us.authService.SetUserCacheInfo(ctx, userCacheInfo)
	if err != nil {
		return nil, err
	}
	resp.RoleID = userCacheInfo.RoleID
	if resp.RoleID == role.RoleAdminID {
		err = us.authService.SetAdminUserCacheInfo(ctx, resp.AccessToken, userCacheInfo)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// RetrievePassWord .
func (us *UserService) RetrievePassWord(ctx context.Context, req *schema.UserRetrievePassWordRequest) error {
	userInfo, has, err := us.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}

	// send email
	data := &schema.EmailCodeContent{
		Email:  req.Email,
		UserID: userInfo.ID,
	}
	code := token.GenerateToken()
	verifyEmailURL := fmt.Sprintf("%s/users/password-reset?code=%s", us.getSiteUrl(ctx), code)
	title, body, err := us.emailService.PassResetTemplate(ctx, verifyEmailURL)
	if err != nil {
		return err
	}
	go us.emailService.SendAndSaveCode(ctx, userInfo.ID, req.Email, title, body, code, data.ToJSONString())
	return nil
}

// UpdatePasswordWhenForgot update user password when user forgot password
func (us *UserService) UpdatePasswordWhenForgot(ctx context.Context, req *schema.UserRePassWordRequest) (err error) {
	data := &schema.EmailCodeContent{}
	err = data.FromJSONString(req.Content)
	if err != nil {
		return errors.BadRequest(reason.EmailVerifyURLExpired)
	}

	userInfo, exist, err := us.userRepo.GetByEmail(ctx, data.Email)
	if err != nil {
		return err
	}
	if !exist {
		return errors.BadRequest(reason.UserNotFound)
	}
	enpass, err := us.encryptPassword(ctx, req.Pass)
	if err != nil {
		return err
	}
	err = us.userRepo.UpdatePass(ctx, userInfo.ID, enpass)
	if err != nil {
		return err
	}
	// When the user changes the password, all the current user's tokens are invalid.
	us.authService.RemoveUserAllTokens(ctx, userInfo.ID)
	return nil
}

func (us *UserService) UserModifyPassWordVerification(ctx context.Context, req *schema.UserModifyPasswordReq) (bool, error) {
	userInfo, has, err := us.userRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return false, err
	}
	if !has {
		return false, errors.BadRequest(reason.UserNotFound)
	}
	isPass := us.verifyPassword(ctx, req.OldPass, userInfo.Pass)
	if !isPass {
		return false, nil
	}

	return true, nil
}

// UserModifyPassword user modify password
func (us *UserService) UserModifyPassword(ctx context.Context, req *schema.UserModifyPasswordReq) error {
	enpass, err := us.encryptPassword(ctx, req.Pass)
	if err != nil {
		return err
	}
	userInfo, exist, err := us.userRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return err
	}
	if !exist {
		return errors.BadRequest(reason.UserNotFound)
	}

	isPass := us.verifyPassword(ctx, req.OldPass, userInfo.Pass)
	if !isPass {
		return errors.BadRequest(reason.OldPasswordVerificationFailed)
	}
	err = us.userRepo.UpdatePass(ctx, userInfo.ID, enpass)
	if err != nil {
		return err
	}

	us.authService.RemoveTokensExceptCurrentUser(ctx, userInfo.ID, req.AccessToken)
	return nil
}

// UpdateInfo update user info
func (us *UserService) UpdateInfo(ctx context.Context, req *schema.UpdateInfoRequest) (
	errFields []*validator.FormErrorField, err error) {
	if len(req.Username) > 0 {
		if checker.IsInvalidUsername(req.Username) {
			return append(errFields, &validator.FormErrorField{
				ErrorField: "username",
				ErrorMsg:   reason.UsernameInvalid,
			}), errors.BadRequest(reason.UsernameInvalid)
		}
		// admin can use reserved username
		if !req.IsAdmin && checker.IsReservedUsername(req.Username) {
			return append(errFields, &validator.FormErrorField{
				ErrorField: "username",
				ErrorMsg:   reason.UsernameInvalid,
			}), errors.BadRequest(reason.UsernameInvalid)
		} else if req.IsAdmin && checker.IsUsersIgnorePath(req.Username) {
			return append(errFields, &validator.FormErrorField{
				ErrorField: "username",
				ErrorMsg:   reason.UsernameInvalid,
			}), errors.BadRequest(reason.UsernameInvalid)
		}

		userInfo, exist, err := us.userRepo.GetByUsername(ctx, req.Username)
		if err != nil {
			return nil, err
		}
		if exist && userInfo.ID != req.UserID {
			return append(errFields, &validator.FormErrorField{
				ErrorField: "username",
				ErrorMsg:   reason.UsernameDuplicate,
			}), errors.BadRequest(reason.UsernameDuplicate)
		}
	}

	oldUserInfo, exist, err := us.userRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	errFields, err = us.validateAvatarInfo(ctx, req.UserID, oldUserInfo.Avatar, req.Avatar)
	if err != nil {
		return errFields, err
	}

	cond := us.formatUserInfoForUpdateInfo(oldUserInfo, req)

	us.cleanUpRemovedAvatar(ctx, req.UserID, oldUserInfo.Avatar, cond.Avatar)

	err = us.userRepo.UpdateInfo(ctx, cond)
	if err != nil {
		return nil, err
	}
	us.eventQueueService.Send(ctx, schema.NewEvent(constant.EventUserUpdate, req.UserID))
	return nil, err
}

func (us *UserService) validateAvatarInfo(
	ctx context.Context,
	userID string,
	oldAvatarJSON string,
	newAvatar schema.AvatarInfo,
) (errFields []*validator.FormErrorField, err error) {
	if newAvatar.Type != constant.AvatarTypeCustom {
		return nil, nil
	}
	if len(newAvatar.Custom) == 0 {
		return append(errFields, &validator.FormErrorField{
			ErrorField: "avatar",
			ErrorMsg:   reason.UserSetAvatar,
		}), errors.BadRequest(reason.UserSetAvatar)
	}

	var oldAvatar schema.AvatarInfo
	_ = json.Unmarshal([]byte(oldAvatarJSON), &oldAvatar)
	if oldAvatar.Type == constant.AvatarTypeCustom && oldAvatar.Custom == newAvatar.Custom {
		return nil, nil
	}

	fileRecord, err := us.fileRecordService.GetFileRecordByURL(ctx, newAvatar.Custom)
	if err != nil {
		return nil, err
	}
	if fileRecord == nil || fileRecord.UserID != userID || fileRecord.Source != string(plugin.UserAvatar) {
		return append(errFields, &validator.FormErrorField{
			ErrorField: "avatar",
			ErrorMsg:   reason.UserSetAvatar,
		}), errors.BadRequest(reason.UserSetAvatar)
	}
	return nil, nil
}

func (us *UserService) cleanUpRemovedAvatar(
	ctx context.Context,
	updatingUserID string,
	oldAvatarJSON string,
	newAvatarJSON string,
) {
	if oldAvatarJSON == newAvatarJSON {
		return
	}

	var oldAvatar, newAvatar schema.AvatarInfo

	_ = json.Unmarshal([]byte(oldAvatarJSON), &oldAvatar)
	_ = json.Unmarshal([]byte(newAvatarJSON), &newAvatar)

	if len(oldAvatar.Custom) == 0 {
		return
	}

	// clean up if old is custom and it's either removed or replaced
	if oldAvatar.Custom != newAvatar.Custom {
		fileRecord, err := us.fileRecordService.GetFileRecordByURL(ctx, oldAvatar.Custom)
		if err != nil {
			log.Error(err)
			return
		}
		if fileRecord == nil {
			log.Warn("no file record found for old avatar url:", oldAvatar.Custom)
			return
		}
		if fileRecord.UserID != updatingUserID || fileRecord.Source != string(plugin.UserAvatar) {
			log.Warnf(
				"refuse to clean avatar url %q: file record owner/source mismatch (owner=%s source=%s updating_user=%s)",
				oldAvatar.Custom, fileRecord.UserID, fileRecord.Source, updatingUserID,
			)
			return
		}
		if err := us.fileRecordService.DeleteAndMoveFileRecord(ctx, fileRecord); err != nil {
			log.Error(err)
		}
	}
}

func (us *UserService) formatUserInfoForUpdateInfo(
	oldUserInfo *entity.User, req *schema.UpdateInfoRequest) *entity.User {
	avatar, _ := json.Marshal(req.Avatar)

	userInfo := &entity.User{}
	userInfo.DisplayName = oldUserInfo.DisplayName
	userInfo.Username = oldUserInfo.Username
	userInfo.Avatar = oldUserInfo.Avatar
	userInfo.Bio = oldUserInfo.Bio
	userInfo.BioHTML = oldUserInfo.BioHTML
	userInfo.Website = oldUserInfo.Website
	userInfo.Location = oldUserInfo.Location
	userInfo.ID = req.UserID

	if len(req.DisplayName) > 0 {
		userInfo.DisplayName = req.DisplayName
	}
	if len(req.Username) > 0 {
		userInfo.Username = req.Username
	}
	if len(avatar) > 0 {
		userInfo.Avatar = string(avatar)
	}
	userInfo.Bio = req.Bio
	userInfo.BioHTML = req.BioHTML
	userInfo.Website = req.Website
	userInfo.Location = req.Location
	return userInfo
}

// UserUpdateInterface update user interface
func (us *UserService) UserUpdateInterface(ctx context.Context, req *schema.UpdateUserInterfaceRequest) (err error) {
	return us.userRepo.UpdateUserInterface(ctx, req.UserId, req.Language, req.ColorScheme)
}

// UserRegisterByEmail user register
func (us *UserService) UserRegisterByEmail(ctx context.Context, registerUserInfo *schema.UserRegisterReq) (
	resp *schema.UserLoginResp, errFields []*validator.FormErrorField, err error,
) {
	_, has, err := us.userRepo.GetByEmail(ctx, registerUserInfo.Email)
	if err != nil {
		return nil, nil, err
	}
	if has {
		errFields = append(errFields, &validator.FormErrorField{
			ErrorField: "e_mail",
			ErrorMsg:   reason.EmailDuplicate,
		})
		return nil, errFields, errors.BadRequest(reason.EmailDuplicate)
	}

	userInfo := &entity.User{}
	userInfo.EMail = registerUserInfo.Email
	userInfo.DisplayName = registerUserInfo.Name
	userInfo.Pass, err = us.encryptPassword(ctx, registerUserInfo.Pass)
	if err != nil {
		return nil, nil, err
	}
	userInfo.Username, err = us.userCommonService.MakeUsername(ctx, registerUserInfo.Name)
	if err != nil {
		errFields = append(errFields, &validator.FormErrorField{
			ErrorField: "name",
			ErrorMsg:   reason.UsernameInvalid,
		})
		return nil, errFields, err
	}
	userInfo.IPInfo = registerUserInfo.IP
	userInfo.MailStatus = entity.EmailStatusToBeVerified
	userInfo.Status = entity.UserStatusAvailable
	userInfo.LastLoginDate = time.Now()
	err = us.userRepo.AddUser(ctx, userInfo)
	if err != nil {
		return nil, nil, err
	}
	if err := us.userNotificationConfigService.SetDefaultUserNotificationConfig(ctx, []string{userInfo.ID}); err != nil {
		log.Errorf("set default user notification config failed, err: %v", err)
	}

	err = applyRegistrationVerification(userInfo, registerUserInfo.RequireEmailVerification, registrationVerificationActions{
		sendActivationEmail: func() error {
			return us.sendRegistrationActivationEmail(ctx, userInfo)
		},
		activateUser: func() error {
			return us.userActivity.UserActive(ctx, userInfo.ID)
		},
		markEmailAvailable: func() error {
			return us.userRepo.UpdateEmailStatus(ctx, userInfo.ID, entity.EmailStatusAvailable)
		},
	})
	if err != nil {
		return nil, nil, err
	}

	roleID, err := us.userRoleService.GetUserRole(ctx, userInfo.ID)
	if err != nil {
		log.Error(err)
	}

	// return user info and token
	resp = &schema.UserLoginResp{}
	resp.ConvertFromUserEntity(userInfo)
	resp.Avatar = us.siteInfoService.FormatAvatar(ctx, userInfo.Avatar, userInfo.EMail, userInfo.Status).GetURL()
	userCacheInfo := &entity.UserCacheInfo{
		UserID:      userInfo.ID,
		EmailStatus: userInfo.MailStatus,
		UserStatus:  userInfo.Status,
		RoleID:      roleID,
	}
	resp.AccessToken, resp.VisitToken, err = us.authService.SetUserCacheInfo(ctx, userCacheInfo)
	if err != nil {
		return nil, nil, err
	}
	resp.RoleID = userCacheInfo.RoleID
	if resp.RoleID == role.RoleAdminID {
		err = us.authService.SetAdminUserCacheInfo(ctx, resp.AccessToken, &entity.UserCacheInfo{UserID: userInfo.ID})
		if err != nil {
			return nil, nil, err
		}
	}
	return resp, nil, nil
}

type registrationVerificationActions struct {
	sendActivationEmail func() error
	activateUser        func() error
	markEmailAvailable  func() error
}

func applyRegistrationVerification(
	userInfo *entity.User, requireEmailVerification bool, actions registrationVerificationActions,
) error {
	userInfo.MailStatus = entity.EmailStatusToBeVerified
	if requireEmailVerification {
		return actions.sendActivationEmail()
	}

	if err := actions.activateUser(); err != nil {
		log.Errorf("activate user during registration failed, fallback to email verification, err: %v", err)
		return actions.sendActivationEmail()
	}
	if err := actions.markEmailAvailable(); err != nil {
		log.Errorf("mark email available during registration failed, fallback to email verification, err: %v", err)
		return actions.sendActivationEmail()
	}
	userInfo.MailStatus = entity.EmailStatusAvailable
	return nil
}

func (us *UserService) sendRegistrationActivationEmail(ctx context.Context, userInfo *entity.User) error {
	data := &schema.EmailCodeContent{
		Email:  userInfo.EMail,
		UserID: userInfo.ID,
	}
	code := token.GenerateToken()
	verifyEmailURL := fmt.Sprintf("%s/users/account-activation?code=%s", us.getSiteUrl(ctx), code)
	title, body, err := us.emailService.RegisterTemplate(ctx, verifyEmailURL)
	if err != nil {
		return err
	}
	go us.emailService.SendAndSaveCode(ctx, userInfo.ID, userInfo.EMail, title, body, code, data.ToJSONString())
	return nil
}

func (us *UserService) UserVerifyEmailSend(ctx context.Context, userID string) error {
	userInfo, has, err := us.userRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if !has {
		return errors.BadRequest(reason.UserNotFound)
	}

	data := &schema.EmailCodeContent{
		Email:  userInfo.EMail,
		UserID: userInfo.ID,
	}
	code := token.GenerateToken()
	verifyEmailURL := fmt.Sprintf("%s/users/account-activation?code=%s", us.getSiteUrl(ctx), code)
	title, body, err := us.emailService.RegisterTemplate(ctx, verifyEmailURL)
	if err != nil {
		return err
	}
	go us.emailService.SendAndSaveCode(ctx, userInfo.ID, userInfo.EMail, title, body, code, data.ToJSONString())
	return nil
}

func (us *UserService) UserVerifyEmail(ctx context.Context, req *schema.UserVerifyEmailReq) (resp *schema.UserLoginResp, err error) {
	data := &schema.EmailCodeContent{}
	err = data.FromJSONString(req.Content)
	if err != nil {
		return nil, errors.BadRequest(reason.EmailVerifyURLExpired)
	}

	userInfo, has, err := us.userRepo.GetByEmail(ctx, data.Email)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	if userInfo.MailStatus == entity.EmailStatusToBeVerified {
		userInfo.MailStatus = entity.EmailStatusAvailable
		err = us.userRepo.UpdateEmailStatus(ctx, userInfo.ID, userInfo.MailStatus)
		if err != nil {
			return nil, err
		}
	}
	if err = us.userActivity.UserActive(ctx, userInfo.ID); err != nil {
		log.Error(err)
		return nil, err
	}

	// In the case of three-party login, the associated users are bound
	if len(data.BindingKey) > 0 {
		err = us.userExternalLoginService.ExternalLoginBindingUser(ctx, data.BindingKey, userInfo)
		if err != nil {
			return nil, err
		}
	}

	accessToken, userCacheInfo, err := us.userCommonService.CacheLoginUserInfo(
		ctx, userInfo.ID, userInfo.MailStatus, userInfo.Status, "")
	if err != nil {
		return nil, err
	}

	resp = &schema.UserLoginResp{}
	resp.ConvertFromUserEntity(userInfo)
	resp.Avatar = us.siteInfoService.FormatAvatar(ctx, userInfo.Avatar, userInfo.EMail, userInfo.Status).GetURL()
	resp.AccessToken = accessToken
	// User verified email will update user email status. So user status cache should be updated.
	if err = us.authService.SetUserStatus(ctx, userCacheInfo); err != nil {
		return nil, err
	}
	return resp, nil
}

// verifyPassword
// Compare whether the password is correct
func (us *UserService) verifyPassword(_ context.Context, loginPass, userPass string) bool {
	if len(loginPass) == 0 && len(userPass) == 0 {
		return true
	}
	err := bcrypt.CompareHashAndPassword([]byte(userPass), []byte(loginPass))
	return err == nil
}

// encryptPassword
// The password does irreversible encryption.
func (us *UserService) encryptPassword(_ context.Context, pass string) (string, error) {
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	// This encrypted string can be saved to the database and can be used as password matching verification
	return string(hashPwd), err
}

// UserChangeEmailSendCode user change email verification
func (us *UserService) UserChangeEmailSendCode(ctx context.Context, req *schema.UserChangeEmailSendCodeReq) (
	resp []*validator.FormErrorField, err error) {
	userInfo, exist, err := us.userRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.BadRequest(reason.UserNotFound)
	}

	// If user's email already verified, then must verify password first.
	if userInfo.MailStatus == entity.EmailStatusAvailable && !us.verifyPassword(ctx, req.Pass, userInfo.Pass) {
		resp = append(resp, &validator.FormErrorField{
			ErrorField: "pass",
			ErrorMsg:   translator.Tr(handler.GetLangByCtx(ctx), reason.OldPasswordVerificationFailed),
		})
		return resp, errors.BadRequest(reason.OldPasswordVerificationFailed)
	}

	_, exist, err = us.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exist {
		resp = append([]*validator.FormErrorField{}, &validator.FormErrorField{
			ErrorField: "e_mail",
			ErrorMsg:   translator.Tr(handler.GetLangByCtx(ctx), reason.EmailDuplicate),
		})
		return resp, errors.BadRequest(reason.EmailDuplicate)
	}

	data := &schema.EmailCodeContent{
		Email:  req.Email,
		UserID: req.UserID,
	}
	code := token.GenerateToken()
	var title, body string
	verifyEmailURL := fmt.Sprintf("%s/users/confirm-new-email?code=%s", us.getSiteUrl(ctx), code)
	if userInfo.MailStatus == entity.EmailStatusToBeVerified {
		title, body, err = us.emailService.RegisterTemplate(ctx, verifyEmailURL)
	} else {
		title, body, err = us.emailService.ChangeEmailTemplate(ctx, verifyEmailURL)
	}
	if err != nil {
		return nil, err
	}
	log.Infof("send email confirmation %s", verifyEmailURL)

	go us.emailService.SendAndSaveCode(ctx, userInfo.ID, req.Email, title, body, code, data.ToJSONString())
	return nil, nil
}

// UserChangeEmailVerify user change email verify code
func (us *UserService) UserChangeEmailVerify(ctx context.Context, content string) (resp *schema.UserLoginResp, err error) {
	data := &schema.EmailCodeContent{}
	err = data.FromJSONString(content)
	if err != nil {
		return nil, errors.BadRequest(reason.EmailVerifyURLExpired)
	}

	_, exist, err := us.userRepo.GetByEmail(ctx, data.Email)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, errors.BadRequest(reason.EmailDuplicate)
	}

	userInfo, exist, err := us.userRepo.GetByUserID(ctx, data.UserID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	err = us.userRepo.UpdateEmail(ctx, data.UserID, data.Email)
	if err != nil {
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	err = us.userRepo.UpdateEmailStatus(ctx, data.UserID, entity.EmailStatusAvailable)
	if err != nil {
		return nil, err
	}
	// if email status is to be verified, active user as well
	if userInfo.MailStatus == entity.EmailStatusToBeVerified {
		if err = us.userActivity.UserActive(ctx, userInfo.ID); err != nil {
			log.Error(err)
			return nil, err
		}
	}

	roleID, err := us.userRoleService.GetUserRole(ctx, userInfo.ID)
	if err != nil {
		log.Error(err)
	}

	resp = &schema.UserLoginResp{}
	resp.ConvertFromUserEntity(userInfo)
	resp.Avatar = us.siteInfoService.FormatAvatar(ctx, userInfo.Avatar, userInfo.EMail, userInfo.Status).GetURL()
	userCacheInfo := &entity.UserCacheInfo{
		UserID:      userInfo.ID,
		EmailStatus: entity.EmailStatusAvailable,
		UserStatus:  userInfo.Status,
		RoleID:      roleID,
	}
	resp.AccessToken, resp.VisitToken, err = us.authService.SetUserCacheInfo(ctx, userCacheInfo)
	if err != nil {
		return nil, err
	}
	// User verified email will update user email status. So user status cache should be updated.
	if err = us.authService.SetUserStatus(ctx, userCacheInfo); err != nil {
		return nil, err
	}
	resp.RoleID = userCacheInfo.RoleID
	if resp.RoleID == role.RoleAdminID {
		err = us.authService.SetAdminUserCacheInfo(ctx, resp.AccessToken, &entity.UserCacheInfo{UserID: userInfo.ID})
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// getSiteUrl get site url
func (us *UserService) getSiteUrl(ctx context.Context) string {
	siteGeneral, err := us.siteInfoService.GetSiteGeneral(ctx)
	if err != nil {
		log.Errorf("get site general failed: %s", err)
		return ""
	}
	return siteGeneral.SiteUrl
}

// UserRanking get user ranking
func (us *UserService) UserRanking(ctx context.Context) (resp *schema.UserRankingResp, err error) {
	limit := 20
	endTime := time.Now()
	// [cd] both rankings cover the whole history, not just the last week
	startTime := time.Unix(0, 0)
	userIDs, userIDExist := make([]string, 0), make(map[string]bool, 0)

	// [cd] most reputation users of all time, straight from user.rank
	rankStat, rankStatUserIDs, err := us.getTopRankUsers(ctx, limit, userIDExist)
	if err != nil {
		return nil, err
	}
	userIDs = append(userIDs, rankStatUserIDs...)

	// get most vote users
	voteStat, voteStatUserIDs, err := us.getActivityUserVoteStat(ctx, startTime, endTime, limit, userIDExist)
	if err != nil {
		return nil, err
	}
	userIDs = append(userIDs, voteStatUserIDs...)

	// get all staff members
	userRoleRels, staffUserIDs, err := us.getStaff(ctx, userIDExist)
	if err != nil {
		return nil, err
	}
	userIDs = append(userIDs, staffUserIDs...)

	// get user information
	userInfoMapping, err := us.getUserInfoMapping(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	resp = us.warpStatRankingResp(userInfoMapping, rankStat, voteStat, userRoleRels)
	us.fillRankingPrestige(ctx, userInfoMapping, resp)
	us.fillActivityRankings(ctx, resp, limit)
	us.fillContestRanking(ctx, resp)
	return resp, nil
}

// fillActivityRankings [cd] the two sections the dashboard also shows: who did the most
// (questions + answers + comments) and who read the most, over the last seven days.
// These decorations are optional: a failure leaves the sections empty, never breaks the page.
func (us *UserService) fillActivityRankings(ctx context.Context, resp *schema.UserRankingResp, limit int) {
	if us.activityLogService == nil {
		return
	}
	from := time.Now().AddDate(0, 0, -7).Unix()
	for _, section := range []struct {
		sort string
		fill *[]*schema.UserRankingSimpleInfo
	}{
		{sort: "activity", fill: &resp.MostActiveUsers},
		{sort: "views", fill: &resp.MostViewingUsers},
	} {
		rows, err := us.activityLogService.TopUsers(ctx, &schema.ActivityLogTopUsersReq{
			From: from, Limit: limit, Sort: section.sort,
		})
		if err != nil {
			log.Errorf("get top users (%s) failed: %v", section.sort, err)
			continue
		}
		list := make([]*schema.UserRankingSimpleInfo, 0, len(rows))
		for _, row := range rows {
			if row.User == nil || row.User.Username == "" || excludedUsername(row.User.Username) {
				continue
			}
			list = append(list, &schema.UserRankingSimpleInfo{
				Username:      row.User.Username,
				DisplayName:   row.User.DisplayName,
				Avatar:        row.User.Avatar,
				Rank:          row.User.Rank,
				ActivityCount: int(row.Total),
				ViewCount:     int(row.Views),
				Prestige:      row.User.Prestige,
			})
		}
		*section.fill = list
	}
}

// rankingExcludedUsernames [cd] accounts that post on behalf of the system are kept out of the
// public rankings; override with RANKING_EXCLUDED_USERNAMES (comma separated usernames).
var rankingExcludedUsernames = func() map[string]bool {
	raw := os.Getenv("RANKING_EXCLUDED_USERNAMES")
	if raw == "" {
		raw = "baza-wiedzy"
	}
	excluded := make(map[string]bool)
	for _, name := range strings.Split(raw, ",") {
		if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
			excluded[name] = true
		}
	}
	return excluded
}()

func excludedFromRanking(userInfo *entity.User) bool {
	return userInfo == nil || excludedUsername(userInfo.Username)
}

func excludedUsername(username string) bool {
	return rankingExcludedUsernames[strings.ToLower(username)]
}

// getTopRankUsers [cd] users with the highest reputation of all time
func (us *UserService) getTopRankUsers(ctx context.Context, limit int, userIDExist map[string]bool) (
	rankStat []*entity.ActivityUserRankStat, userIDs []string, err error) {
	rankStat = make([]*entity.ActivityUserRankStat, 0)
	userList, err := us.userRepo.ListTopByRank(ctx, limit)
	if err != nil {
		return nil, nil, err
	}
	for _, user := range userList {
		if userIDExist[user.ID] || excludedFromRanking(user) {
			continue
		}
		rankStat = append(rankStat, &entity.ActivityUserRankStat{UserID: user.ID, Rank: user.Rank})
		userIDs = append(userIDs, user.ID)
		userIDExist[user.ID] = true
	}
	return rankStat, userIDs, nil
}

// fillRankingPrestige [cd] adds the prestige card to every ranking entry and puts the highest
// ranks on top of the reputation list (reputation first, rank insignia breaks the tie).
func (us *UserService) fillRankingPrestige(ctx context.Context, userInfoMapping map[string]*entity.User,
	resp *schema.UserRankingResp) {
	if resp == nil {
		return
	}
	userIDs := make([]string, 0, len(userInfoMapping))
	usernameToID := make(map[string]string, len(userInfoMapping))
	for id, user := range userInfoMapping {
		if user == nil {
			continue
		}
		userIDs = append(userIDs, id)
		usernameToID[user.Username] = id
	}
	prestigeMap := us.userCommonService.BatchPrestige(ctx, userIDs)
	if len(prestigeMap) == 0 {
		return
	}
	decorate := func(list []*schema.UserRankingSimpleInfo) {
		for _, item := range list {
			if prestige, ok := prestigeMap[usernameToID[item.Username]]; ok {
				item.Prestige = prestige
			}
		}
	}
	decorate(resp.UsersWithTheMostReputation)
	decorate(resp.UsersWithTheMostVote)
	decorate(resp.Staffs)

	sort.SliceStable(resp.UsersWithTheMostReputation, func(i, j int) bool {
		return higherRanking(resp.UsersWithTheMostReputation[i], resp.UsersWithTheMostReputation[j])
	})
}

// higherRanking [cd] puts the highest insignia on top, reputation breaks the tie
func higherRanking(a, b *schema.UserRankingSimpleInfo) bool {
	if rankAmountOf(a) != rankAmountOf(b) {
		return rankAmountOf(a) > rankAmountOf(b)
	}
	return a.Rank > b.Rank
}

func rankAmountOf(info *schema.UserRankingSimpleInfo) int {
	if info == nil || info.Prestige == nil {
		return 0
	}
	return info.Prestige.RankBadgeAmount
}

// GetUserStaff get user staff
func (us *UserService) GetUserStaff(ctx context.Context, req *schema.GetUserStaffReq) (
	resp []*schema.GetUserStaffResp, err error) {
	userList, err := us.userRepo.SearchUserListByName(ctx, req.Username, req.PageSize, true)
	if err != nil {
		return nil, err
	}
	avatarMapping := us.siteInfoService.FormatListAvatar(ctx, userList)
	for _, u := range userList {
		resp = append(resp, &schema.GetUserStaffResp{
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Avatar:      avatarMapping[u.ID].GetURL(),
		})
	}
	return resp, nil
}

// UserUnsubscribeNotification user unsubscribe email notification
func (us *UserService) UserUnsubscribeNotification(
	ctx context.Context, req *schema.UserUnsubscribeNotificationReq) (err error) {
	data := &schema.EmailCodeContent{}
	err = data.FromJSONString(req.Content)
	if err != nil || len(data.UserID) == 0 {
		return errors.BadRequest(reason.EmailVerifyURLExpired)
	}

	for _, source := range data.NotificationSources {
		notificationConfig, exist, err := us.userNotificationConfigRepo.GetByUserIDAndSource(
			ctx, data.UserID, source)
		if err != nil {
			return err
		}
		if !exist {
			continue
		}
		channels := schema.NewNotificationChannelsFormJson(notificationConfig.Channels)
		// unsubscribe email notification
		for _, channel := range channels {
			if channel.Key == constant.EmailChannel {
				channel.Enable = false
			}
		}
		notificationConfig.Channels = channels.ToJsonString()
		if err = us.userNotificationConfigRepo.Save(ctx, notificationConfig); err != nil {
			return err
		}
	}
	return nil
}

func (us *UserService) getActivityUserRankStat(ctx context.Context, startTime, endTime time.Time, limit int,
	userIDExist map[string]bool) (rankStat []*entity.ActivityUserRankStat, userIDs []string, err error) {
	if plugin.RankAgentEnabled() {
		return make([]*entity.ActivityUserRankStat, 0), make([]string, 0), nil
	}
	rankStat, err = us.activityRepo.GetUsersWhoHasGainedTheMostReputation(ctx, startTime, endTime, limit)
	if err != nil {
		return nil, nil, err
	}
	for _, stat := range rankStat {
		if stat.Rank <= 0 {
			continue
		}
		if userIDExist[stat.UserID] {
			continue
		}
		userIDs = append(userIDs, stat.UserID)
		userIDExist[stat.UserID] = true
	}
	return rankStat, userIDs, nil
}

func (us *UserService) getActivityUserVoteStat(ctx context.Context, startTime, endTime time.Time, limit int,
	userIDExist map[string]bool) (voteStat []*entity.ActivityUserVoteStat, userIDs []string, err error) {
	if plugin.RankAgentEnabled() {
		return make([]*entity.ActivityUserVoteStat, 0), make([]string, 0), nil
	}
	voteStat, err = us.activityRepo.GetUsersWhoHasVoteMost(ctx, startTime, endTime, limit)
	if err != nil {
		return nil, nil, err
	}
	for _, stat := range voteStat {
		if stat.VoteCount <= 0 {
			continue
		}
		if userIDExist[stat.UserID] {
			continue
		}
		userIDs = append(userIDs, stat.UserID)
		userIDExist[stat.UserID] = true
	}
	return voteStat, userIDs, nil
}

func (us *UserService) getStaff(ctx context.Context, userIDExist map[string]bool) (
	userRoleRels []*entity.UserRoleRel, userIDs []string, err error) {
	userRoleRels, err = us.userRoleService.GetUserByRoleID(ctx, []int{role.RoleAdminID, role.RoleModeratorID})
	if err != nil {
		return nil, nil, err
	}
	for _, rel := range userRoleRels {
		if userIDExist[rel.UserID] {
			continue
		}
		userIDs = append(userIDs, rel.UserID)
		userIDExist[rel.UserID] = true
	}
	return userRoleRels, userIDs, nil
}

func (us *UserService) getUserInfoMapping(ctx context.Context, userIDs []string) (
	userInfoMapping map[string]*entity.User, err error) {
	userInfoMapping = make(map[string]*entity.User, 0)
	if len(userIDs) == 0 {
		return userInfoMapping, nil
	}
	userInfoList, err := us.userRepo.BatchGetByID(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	avatarMapping := us.siteInfoService.FormatListAvatar(ctx, userInfoList)
	for _, user := range userInfoList {
		user.Avatar = avatarMapping[user.ID].GetURL()
		userInfoMapping[user.ID] = user
	}
	return userInfoMapping, nil
}

func (us *UserService) SearchUserListByName(ctx context.Context, req *schema.GetOtherUserInfoByUsernameReq) (
	resp []*schema.UserBasicInfo, err error) {
	resp = make([]*schema.UserBasicInfo, 0)
	if len(req.Username) == 0 {
		return resp, nil
	}
	userList, err := us.userRepo.SearchUserListByName(ctx, req.Username, 5, false)
	if err != nil {
		return resp, err
	}
	avatarMapping := us.siteInfoService.FormatListAvatar(ctx, userList)
	for _, u := range userList {
		if req.UserID == u.ID {
			continue
		}
		basicInfo := us.userCommonService.FormatUserBasicInfo(ctx, u)
		basicInfo.Avatar = avatarMapping[u.ID].GetURL()
		resp = append(resp, basicInfo)
	}
	return resp, nil
}

func (us *UserService) warpStatRankingResp(
	userInfoMapping map[string]*entity.User,
	rankStat []*entity.ActivityUserRankStat,
	voteStat []*entity.ActivityUserVoteStat,
	userRoleRels []*entity.UserRoleRel) (resp *schema.UserRankingResp) {
	resp = &schema.UserRankingResp{
		UsersWithTheMostReputation: make([]*schema.UserRankingSimpleInfo, 0),
		UsersWithTheMostVote:       make([]*schema.UserRankingSimpleInfo, 0),
		Staffs:                     make([]*schema.UserRankingSimpleInfo, 0),
	}
	for _, stat := range rankStat {
		if stat.Rank <= 0 {
			continue
		}
		if userInfo := userInfoMapping[stat.UserID]; userInfo != nil && userInfo.Status != entity.UserStatusDeleted &&
			!excludedFromRanking(userInfo) {
			resp.UsersWithTheMostReputation = append(resp.UsersWithTheMostReputation, &schema.UserRankingSimpleInfo{
				Username:    userInfo.Username,
				Rank:        stat.Rank,
				DisplayName: userInfo.DisplayName,
				Avatar:      userInfo.Avatar,
			})
		}
	}
	for _, stat := range voteStat {
		if stat.VoteCount <= 0 {
			continue
		}
		if userInfo := userInfoMapping[stat.UserID]; userInfo != nil && userInfo.Status != entity.UserStatusDeleted &&
			!excludedFromRanking(userInfo) {
			resp.UsersWithTheMostVote = append(resp.UsersWithTheMostVote, &schema.UserRankingSimpleInfo{
				Username:    userInfo.Username,
				VoteCount:   stat.VoteCount,
				DisplayName: userInfo.DisplayName,
				Avatar:      userInfo.Avatar,
			})
		}
	}
	for _, rel := range userRoleRels {
		if userInfo := userInfoMapping[rel.UserID]; userInfo != nil && userInfo.Status != entity.UserStatusDeleted &&
			!excludedFromRanking(userInfo) {
			resp.Staffs = append(resp.Staffs, &schema.UserRankingSimpleInfo{
				Username:    userInfo.Username,
				Rank:        userInfo.Rank,
				DisplayName: userInfo.DisplayName,
				Avatar:      userInfo.Avatar,
			})
		}
	}
	return resp
}

// Help points (PP) of the contest, §3 of the contest rules. Only the rules that can be counted
// without a moderator's judgement are here; the ones that need one (a substantive answer without
// votes, a substantive comment, the knowledge base bonus) are added by the jury, not by the portal.
const (
	contestPPSolution        = 10.0
	contestPPUpvotedAnswer   = 5.0
	contestPPUpvotedQuestion = 4.0
	// §3: at most 15 PP from a single answer
	contestPPMaxPerAnswer = 15.0
	// §3: an answer scores for an upvote only when the voter has at least this reputation
	contestMinVoterRank = 100
	// §3.4: points for questions may not exceed 40% of the total, so at most 2/3 of the answer points
	contestQuestionShareOfAnswers = 2.0 / 3.0
	// §4 ranks places 1-20
	contestListLimit = 20
)

// contestExcludedUsernames [cd] §2.2 keeps the service accounts out of the contest; portal staff
// (admins and moderators) is excluded by role. Override with CONTEST_EXCLUDED_USERNAMES.
var contestExcludedUsernames = func() map[string]bool {
	raw := os.Getenv("CONTEST_EXCLUDED_USERNAMES")
	if raw == "" {
		raw = "baza-wiedzy,asystent-ai"
	}
	excluded := make(map[string]bool)
	for _, name := range strings.Split(raw, ",") {
		if name = strings.ToLower(strings.TrimSpace(name)); name != "" {
			excluded[name] = true
		}
	}
	return excluded
}()

// contestQuarterStart the first day of the calendar quarter the moment belongs to
func contestQuarterStart(now time.Time) time.Time {
	month := (int(now.Month())-1)/3*3 + 1
	return time.Date(now.Year(), time.Month(month), 1, 0, 0, 0, 0, now.Location())
}

// contestAnswerPoints §3: 10 PP for a solution, 5 PP for an answer with at least one qualifying
// upvote (once, whatever the number of votes), half rate for answering your own question (§3.5),
// at most 15 PP from a single answer.
func contestAnswerPoints(accepted, upvoted, ownQuestion bool) float64 {
	points := 0.0
	if accepted {
		points += contestPPSolution
	}
	if upvoted {
		points += contestPPUpvotedAnswer
	}
	if points == 0 {
		return 0
	}
	if ownQuestion {
		points /= 2
	}
	if points > contestPPMaxPerAnswer {
		points = contestPPMaxPerAnswer
	}
	return points
}

// contestTotalPoints §3.4: points for questions may not exceed 40% of the total, the surplus is lost
func contestTotalPoints(answerPoints, questionPoints float64) float64 {
	if limit := answerPoints * contestQuestionShareOfAnswers; questionPoints > limit {
		questionPoints = limit
	}
	return math.Round((answerPoints+questionPoints)*10) / 10
}

// contestScore one participant's running total// contestScore one participant's running total
type contestScore struct {
	answerPoints   float64
	questionPoints float64
	solved         int
	firstScoredAt  time.Time
}

// fillContestRanking [cd] the quarterly contest ranking (§3, §4). Everything it counts comes from
// the activity log, so un-accepting an answer or cancelling a vote takes the points away as well.
func (us *UserService) fillContestRanking(ctx context.Context, resp *schema.UserRankingResp) {
	now := time.Now()
	acts, err := us.contestActivities(ctx, contestQuarterStart(now), now)
	if err != nil {
		log.Errorf("get contest activities failed: %v", err)
		return
	}
	if len(acts.accepted) == 0 && len(acts.answerUpvotes) == 0 && len(acts.questionUpvotes) == 0 {
		return
	}

	// answers accepted as the solution, upvoted answers (qualified voters only), upvoted questions
	accepted := make(map[string]time.Time)           // answer id → when it was accepted
	answerVoters := make(map[string]map[string]bool) // answer id → voter ids
	questionUpvotes := make(map[string]time.Time)    // question id → first upvote
	questionAuthor := make(map[string]string)        // question id → author
	voterIDs := make(map[string]bool)

	for _, a := range acts.accepted {
		if _, seen := accepted[a.ObjectID]; !seen {
			accepted[a.ObjectID] = a.CreatedAt
		}
	}
	for _, a := range acts.answerUpvotes {
		voter := strconv.FormatInt(a.TriggerUserID, 10)
		if voter == "0" || voter == a.UserID {
			continue // §7: no self voting, and an unknown voter cannot be verified
		}
		if answerVoters[a.ObjectID] == nil {
			answerVoters[a.ObjectID] = make(map[string]bool)
		}
		answerVoters[a.ObjectID][voter] = true
		voterIDs[voter] = true
	}
	for _, a := range acts.questionUpvotes {
		if _, seen := questionUpvotes[a.ObjectID]; !seen {
			questionUpvotes[a.ObjectID] = a.CreatedAt
			questionAuthor[a.ObjectID] = a.UserID
		}
	}

	qualifiedVoters := us.contestQualifiedVoters(ctx, voterIDs)

	answerIDs := make([]string, 0, len(accepted)+len(answerVoters))
	for id := range accepted {
		answerIDs = append(answerIDs, id)
	}
	for id := range answerVoters {
		if _, seen := accepted[id]; !seen {
			answerIDs = append(answerIDs, id)
		}
	}
	authorship, err := us.questionService.AnswerAuthorship(ctx, answerIDs)
	if err != nil {
		log.Errorf("get answer authorship failed: %v", err)
		return
	}

	scores := make(map[string]*contestScore)
	score := func(userID string, at time.Time) *contestScore {
		s := scores[userID]
		if s == nil {
			s = &contestScore{firstScoredAt: at}
			scores[userID] = s
		}
		if at.Before(s.firstScoredAt) {
			s.firstScoredAt = at
		}
		return s
	}

	for answerID, author := range authorship {
		at, isAccepted := accepted[answerID]
		upvoted := false
		for voter := range answerVoters[answerID] {
			if qualifiedVoters[voter] {
				upvoted = true
				break
			}
		}
		points := contestAnswerPoints(isAccepted, upvoted,
			author.QuestionUserID != "" && author.QuestionUserID == author.AnswerUserID)
		if points == 0 {
			continue
		}
		when := now
		if isAccepted {
			when = at
		}
		s := score(author.AnswerUserID, when)
		s.answerPoints += points
		if _, ok := accepted[answerID]; ok {
			s.solved++
		}
	}

	for questionID, at := range questionUpvotes {
		author := questionAuthor[questionID]
		if author == "" || author == "0" {
			continue
		}
		s := score(author, at)
		s.questionPoints += contestPPUpvotedQuestion
	}

	resp.ContestRanking = us.contestRankingList(ctx, scores)
}

// contestActivityGroups the three activity types the contest can count
type contestActivityGroups struct {
	accepted        []*entity.Activity
	answerUpvotes   []*entity.Activity
	questionUpvotes []*entity.Activity
}

func (us *UserService) contestActivities(ctx context.Context, from, to time.Time) (*contestActivityGroups, error) {
	types := make(map[string]int, 3)
	for _, key := range []string{activity_type.AnswerAccepted, activity_type.AnswerVotedUp, activity_type.QuestionVotedUp} {
		id, err := us.activityRepo.GetActivityTypeByConfigKey(ctx, key)
		if err != nil {
			return nil, err
		}
		types[key] = id
	}
	ids := []int{types[activity_type.AnswerAccepted], types[activity_type.AnswerVotedUp],
		types[activity_type.QuestionVotedUp]}
	list, err := us.activityRepo.ListByTypesBetween(ctx, ids, from, to)
	if err != nil {
		return nil, err
	}
	groups := &contestActivityGroups{}
	for _, a := range list {
		switch a.ActivityType {
		case types[activity_type.AnswerAccepted]:
			groups.accepted = append(groups.accepted, a)
		case types[activity_type.AnswerVotedUp]:
			groups.answerUpvotes = append(groups.answerUpvotes, a)
		case types[activity_type.QuestionVotedUp]:
			groups.questionUpvotes = append(groups.questionUpvotes, a)
		}
	}
	return groups, nil
}

// contestQualifiedVoters §3: only a vote from a participant with at least 100 reputation scores.
// The reputation is read as it is now, the portal does not store what it was at the time of the vote.
func (us *UserService) contestQualifiedVoters(ctx context.Context, voterIDs map[string]bool) map[string]bool {
	qualified := make(map[string]bool, len(voterIDs))
	if len(voterIDs) == 0 {
		return qualified
	}
	ids := make([]string, 0, len(voterIDs))
	for id := range voterIDs {
		ids = append(ids, id)
	}
	users, err := us.userRepo.BatchGetByID(ctx, ids)
	if err != nil {
		log.Errorf("get contest voters failed: %v", err)
		return qualified
	}
	for _, u := range users {
		if u.Rank >= contestMinVoterRank {
			qualified[u.ID] = true
		}
	}
	return qualified
}

// contestRankingList turns the running totals into the published ranking: the question share is
// capped (§3.4), the excluded accounts and the portal staff drop out (§2.2), ties are broken by the
// number of solutions and then by who got there first (§4.8).
func (us *UserService) contestRankingList(ctx context.Context, scores map[string]*contestScore) []*schema.UserRankingSimpleInfo {
	list := make([]*schema.UserRankingSimpleInfo, 0, len(scores))
	if len(scores) == 0 {
		return list
	}
	staff := make(map[string]bool)
	if rels, err := us.userRoleService.GetUserByRoleID(ctx, []int{role.RoleAdminID, role.RoleModeratorID}); err == nil {
		for _, rel := range rels {
			staff[rel.UserID] = true
		}
	} else {
		log.Errorf("get contest staff failed: %v", err)
	}

	userIDs := make([]string, 0, len(scores))
	for id := range scores {
		if !staff[id] {
			userIDs = append(userIDs, id)
		}
	}
	userInfoMapping, err := us.getUserInfoMapping(ctx, userIDs)
	if err != nil {
		log.Errorf("get contest users failed: %v", err)
		return list
	}
	prestigeMap := us.userCommonService.BatchPrestige(ctx, userIDs)

	type ranked struct {
		info  *schema.UserRankingSimpleInfo
		score *contestScore
	}
	rankedList := make([]ranked, 0, len(userIDs))
	for _, id := range userIDs {
		user := userInfoMapping[id]
		if user == nil || user.Status != entity.UserStatusAvailable ||
			contestExcludedUsernames[strings.ToLower(user.Username)] {
			continue
		}
		s := scores[id]
		total := contestTotalPoints(s.answerPoints, s.questionPoints)
		if total <= 0 {
			continue
		}
		rankedList = append(rankedList, ranked{
			info: &schema.UserRankingSimpleInfo{
				Username:      user.Username,
				DisplayName:   user.DisplayName,
				Avatar:        user.Avatar,
				Rank:          user.Rank,
				ContestPoints: total,
				SolvedCount:   s.solved,
				Prestige:      prestigeMap[id],
			},
			score: s,
		})
	}
	sort.SliceStable(rankedList, func(i, j int) bool {
		a, b := rankedList[i], rankedList[j]
		if a.info.ContestPoints != b.info.ContestPoints {
			return a.info.ContestPoints > b.info.ContestPoints
		}
		if a.info.SolvedCount != b.info.SolvedCount {
			return a.info.SolvedCount > b.info.SolvedCount
		}
		return a.score.firstScoredAt.Before(b.score.firstScoredAt)
	})
	for i, r := range rankedList {
		if i >= contestListLimit {
			break
		}
		list = append(list, r.info)
	}
	return list
}
