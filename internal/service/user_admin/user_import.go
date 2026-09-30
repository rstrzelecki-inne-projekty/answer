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

package user_admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"io"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/activity_log"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
	"golang.org/x/crypto/bcrypt"
)

// maxImportUsersRows keeps one import (and the invitations sent after it) within a few minutes
const maxImportUsersRows = 2000

var (
	importEmailHeaders = map[string]bool{"email": true, "e-mail": true, "mail": true, "adres e-mail": true,
		"adres email": true, "e_mail": true}
	importNameHeaders = map[string]bool{"name": true, "display_name": true, "display name": true, "nazwa": true,
		"imię i nazwisko": true, "imie i nazwisko": true, "nazwisko i imię": true, "nazwisko i imie": true,
		"full name": true, "użytkownik": true, "uzytkownik": true}
	importFirstNameHeaders = map[string]bool{"imię": true, "imie": true, "first name": true, "first_name": true}
	importLastNameHeaders  = map[string]bool{"nazwisko": true, "last name": true, "last_name": true}
)

// parseUserImport reads a CSV/TSV pasted or uploaded by the admin. The delimiter is guessed from the first line
// (tab, semicolon — Excel with the Polish locale — or comma). A header row is optional: with a header the columns
// are found by name (e-mail, name, or first name + last name), without it the e-mail is the cell with "@" and the
// name is made of the remaining cells. Rows are validated here; the database checks come later.
func parseUserImport(content string) (rows []*schema.ImportUsersRow, delimiter string) {
	content = strings.TrimPrefix(content, "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	firstLine, _, _ := strings.Cut(strings.TrimLeft(content, "\n"), "\n")
	comma := ','
	switch {
	case strings.Contains(firstLine, "\t"):
		comma = '\t'
	case strings.Count(firstLine, ";") > strings.Count(firstLine, ","):
		comma = ';'
	}
	delimiter = map[rune]string{'\t': "tab", ';': "semicolon", ',': "comma"}[comma]

	r := csv.NewReader(strings.NewReader(content))
	r.Comma = comma
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	emailCol, nameCol, firstCol, lastCol := -1, -1, -1, -1
	headerSeen := false
	seen := make(map[string]int)
	for {
		record, err := r.Read()
		if err != nil {
			if err != io.EOF {
				line, _ := r.FieldPos(0)
				rows = append(rows, &schema.ImportUsersRow{Line: line, Status: schema.ImportUserStatusInvalid,
					Message: "csv_error: " + err.Error()})
			}
			break
		}
		line, _ := r.FieldPos(0)
		cells := make([]string, len(record))
		empty := true
		for i, c := range record {
			cells[i] = strings.TrimSpace(c)
			if cells[i] != "" {
				empty = false
			}
		}
		if empty {
			continue
		}

		if !headerSeen {
			headerSeen = true
			if !strings.Contains(strings.Join(cells, " "), "@") {
				for i, c := range cells {
					h := strings.ToLower(c)
					switch {
					case importEmailHeaders[h]:
						emailCol = i
					case importNameHeaders[h]:
						nameCol = i
					case importFirstNameHeaders[h]:
						firstCol = i
					case importLastNameHeaders[h]:
						lastCol = i
					}
				}
				continue
			}
		}

		row := &schema.ImportUsersRow{Line: line}
		rows = append(rows, row)
		if len(rows) > maxImportUsersRows {
			row.Status, row.Message = schema.ImportUserStatusInvalid, "too_many_rows"
			continue
		}
		cell := func(i int) string {
			if i >= 0 && i < len(cells) {
				return cells[i]
			}
			return ""
		}
		if emailCol >= 0 {
			row.Email = cell(emailCol)
			switch {
			case nameCol >= 0:
				row.DisplayName = cell(nameCol)
			case firstCol >= 0 || lastCol >= 0:
				row.DisplayName = strings.TrimSpace(cell(firstCol) + " " + cell(lastCol))
			}
		} else {
			var name []string
			for _, c := range cells {
				if row.Email == "" && strings.Contains(c, "@") {
					row.Email = c
				} else if c != "" {
					name = append(name, c)
				}
			}
			row.DisplayName = strings.Join(name, " ")
		}
		row.Email = strings.ToLower(strings.Trim(row.Email, " <>"))
		row.DisplayName = strings.Join(strings.Fields(row.DisplayName), " ")

		addr, err := mail.ParseAddress(row.Email)
		switch {
		case row.Email == "":
			row.Status, row.Message = schema.ImportUserStatusInvalid, "missing_email"
		case err != nil || addr.Address != row.Email || len(row.Email) > 500:
			row.Status, row.Message = schema.ImportUserStatusInvalid, "invalid_email"
		case utf8.RuneCountInString(row.DisplayName) < 2 || utf8.RuneCountInString(row.DisplayName) > 30:
			row.Status, row.Message = schema.ImportUserStatusInvalid, "invalid_name"
		case seen[row.Email] > 0:
			row.Status, row.DuplicateOf = schema.ImportUserStatusDuplicate, seen[row.Email]
		default:
			seen[row.Email] = row.Line
			row.Status = schema.ImportUserStatusNew
		}
	}
	return rows, delimiter
}

// randomPassword a password nobody knows: the person sets their own through the invitation link
func randomPassword() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ImportUsers creates password accounts from a CSV/TSV (people without the Google login). Existing e-mails are left
// untouched; invitations are sent separately (user/invite) so the admin panel can show progress and retry.
func (us *UserAdminService) ImportUsers(ctx context.Context, req *schema.ImportUsersReq) (
	resp *schema.ImportUsersResp, err error) {
	rows, delimiter := parseUserImport(req.Content)
	if len(rows) == 0 {
		return nil, errors.BadRequest(reason.RequestFormatError)
	}
	resp = &schema.ImportUsersResp{Delimiter: delimiter, Rows: rows,
		Statuses: make(map[string]int), Domains: make(map[string]int)}

	for _, row := range rows {
		if row.Status == schema.ImportUserStatusNew {
			existing, has, err := us.userRepo.GetUserInfoByEmail(ctx, row.Email)
			if err != nil {
				return nil, err
			}
			if has {
				row.Status, row.UserID = schema.ImportUserStatusExists, existing.ID
				if !existing.LastLoginDate.IsZero() && existing.LastLoginDate.Unix() > 0 {
					row.LastLoginAt = existing.LastLoginDate.Unix()
				}
			} else if !req.DryRun {
				us.importOne(ctx, req.LoginUserID, row)
			}
		}
		resp.Statuses[row.Status]++
		if _, domain, ok := strings.Cut(row.Email, "@"); ok && row.Status != schema.ImportUserStatusInvalid {
			resp.Domains[domain]++
		}
	}
	return resp, nil
}

func (us *UserAdminService) importOne(ctx context.Context, loginUserID string, row *schema.ImportUsersRow) {
	fail := func(err error) {
		log.Errorf("import user %s: %v", row.Email, err)
		row.Status, row.Message = schema.ImportUserStatusError, err.Error()
	}
	pass, err := randomPassword()
	if err != nil {
		fail(err)
		return
	}
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		fail(err)
		return
	}
	userInfo := &entity.User{
		EMail:       row.Email,
		DisplayName: row.DisplayName,
		Pass:        string(hashPwd),
		MailStatus:  entity.EmailStatusAvailable,
		Status:      entity.UserStatusAvailable,
		Rank:        1,
	}
	if userInfo.Username, err = us.userCommonService.MakeUsername(ctx, row.DisplayName); err != nil {
		fail(err)
		return
	}
	if err = us.userRepo.AddUser(ctx, userInfo); err != nil {
		fail(err)
		return
	}
	row.Status, row.UserID = schema.ImportUserStatusCreated, userInfo.ID
	us.activityLogService.LogDetail(ctx, &entity.ActivityLog{UserID: loginUserID, Action: activity_log.ActionUserCreate,
		ObjectType: constant.UserObjectType, ObjectID: userInfo.ID, TargetUserID: userInfo.ID},
		activity_log.Detail{"email": row.Email, "source": "import"})
}
