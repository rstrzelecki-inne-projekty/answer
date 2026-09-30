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

// Package kb_import [cd] imports ready question–answer pairs into the knowledge base from the admin panel:
// every pair becomes a question with an accepted answer posted by the knowledge-base account, tagged like the
// entries the kb-import service (answers/bot/kb_import.py) generates from documents.
package kb_import

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/activity_log"
	"github.com/apache/answer/internal/service/content"
	tagcommon "github.com/apache/answer/internal/service/tag_common"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
)

// ActionKBImport activity log action of one imported entry
const ActionKBImport = "kb.import"

const maxRows = 2000

// the same footer as the entries kb_import.py publishes from .tsv files
const footer = "\n\n---\n*Wpis bazy wiedzy z zestawu pytań i odpowiedzi: %s. Zgłoś błąd w komentarzu albo popraw wpis.*"

// KBImportService publishes question–answer pairs as the knowledge-base account
type KBImportService struct {
	data               *data.Data
	questionService    *content.QuestionService
	answerService      *content.AnswerService
	tagCommonService   *tagcommon.TagCommonService
	activityLogService *activity_log.ActivityLogService
}

// NewKBImportService new knowledge base import service
func NewKBImportService(
	data *data.Data,
	questionService *content.QuestionService,
	answerService *content.AnswerService,
	tagCommonService *tagcommon.TagCommonService,
	activityLogService *activity_log.ActivityLogService,
) *KBImportService {
	return &KBImportService{data: data, questionService: questionService, answerService: answerService,
		tagCommonService: tagCommonService, activityLogService: activityLogService}
}

// accountUsername the knowledge-base account (KB_IMPORT_USERNAME, "baza-wiedzy" by default)
func accountUsername() string {
	if v := strings.TrimSpace(os.Getenv("KB_IMPORT_USERNAME")); v != "" {
		return v
	}
	return "baza-wiedzy"
}

// baseTag the tag every entry gets (KB_IMPORT_TAG, "baza-wiedzy" by default)
func baseTag() string {
	if v := strings.TrimSpace(os.Getenv("KB_IMPORT_TAG")); v != "" {
		return strings.ToLower(v)
	}
	return "baza-wiedzy"
}

var (
	questionHeaders = map[string]bool{"pytanie": true, "pytania": true, "question": true}
	answerHeaders   = map[string]bool{"odpowiedź": true, "odpowiedz": true, "odpowiedzi": true, "answer": true}
	tagsHeaders     = map[string]bool{"tagi": true, "tags": true, "tag": true}
	tagSplit        = regexp.MustCompile(`[,;\s]+`)
)

// normalizeTitle titles are compared without case and repeated spaces
func normalizeTitle(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// parseRows reads a TSV (or a semicolon / comma CSV) with the columns question, answer and optional tags.
// A header row (PL/EN names) is optional and may reorder the columns; quoted multi-line cells work; a literal \n
// in the text is a line break. Mirrors parse_qa_tsv in answers/bot/kb_import.py.
func parseRows(content string) (rows []*schema.KBImportRow, delimiter string) {
	content = strings.TrimPrefix(content, "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	firstLine, _, _ := strings.Cut(strings.TrimLeft(content, "\n"), "\n")
	comma := '\t'
	switch {
	case strings.Contains(firstLine, "\t"):
	case strings.Count(firstLine, ";") > 0:
		comma = ';'
	case strings.Count(firstLine, ",") > 0:
		comma = ','
	}
	delimiter = map[rune]string{'\t': "tab", ';': "semicolon", ',': "comma"}[comma]

	r := csv.NewReader(strings.NewReader(content))
	r.Comma = comma
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	qCol, aCol, tCol := 0, 1, 2
	headerChecked := false
	seen := make(map[string]int)
	for {
		record, err := r.Read()
		if err != nil {
			if err != io.EOF {
				line, _ := r.FieldPos(0)
				rows = append(rows, &schema.KBImportRow{Line: line, Status: schema.KBImportStatusInvalid,
					Message: "csv_error: " + err.Error()})
			}
			break
		}
		line, _ := r.FieldPos(0)
		empty := true
		for _, c := range record {
			if strings.TrimSpace(c) != "" {
				empty = false
			}
		}
		if empty {
			continue
		}
		if !headerChecked {
			headerChecked = true
			q, a, t := -1, -1, -1
			for i, c := range record {
				h := strings.ToLower(strings.TrimSpace(c))
				switch {
				case questionHeaders[h]:
					q = i
				case answerHeaders[h]:
					a = i
				case tagsHeaders[h]:
					t = i
				}
			}
			if q >= 0 && a >= 0 {
				qCol, aCol, tCol = q, a, t
				continue
			}
		}
		cell := func(i int) string {
			if i >= 0 && i < len(record) {
				return strings.TrimSpace(record[i])
			}
			return ""
		}
		row := &schema.KBImportRow{
			Line:     line,
			Question: strings.Join(strings.Fields(cell(qCol)), " "),
			Answer:   strings.TrimSpace(strings.ReplaceAll(cell(aCol), `\n`, "\n")),
			Tags:     []string{},
		}
		for _, t := range tagSplit.Split(strings.ToLower(cell(tCol)), -1) {
			if t != "" {
				row.Tags = append(row.Tags, t)
			}
		}
		rows = append(rows, row)
		qLen := utf8.RuneCountInString(row.Question)
		key := normalizeTitle(row.Question)
		switch {
		case len(rows) > maxRows:
			row.Status, row.Message = schema.KBImportStatusInvalid, "too_many_rows"
		case row.Question == "" || row.Answer == "":
			row.Status, row.Message = schema.KBImportStatusInvalid, "missing_field"
		case qLen < 6 || qLen > 150:
			row.Status, row.Message = schema.KBImportStatusInvalid, "question_length"
		case utf8.RuneCountInString(row.Answer) < 6:
			row.Status, row.Message = schema.KBImportStatusInvalid, "answer_too_short"
		case seen[key] > 0:
			row.Status, row.DuplicateOf = schema.KBImportStatusDuplicate, seen[key]
		default:
			seen[key] = row.Line
			row.Status = schema.KBImportStatusNew
		}
	}
	return rows, delimiter
}

// account the knowledge-base user; nil when it does not exist (the panel shows how to create it)
func (s *KBImportService) account(ctx context.Context) (*entity.User, error) {
	user := &entity.User{}
	has, err := s.data.DB.Context(ctx).Where("username = ?", accountUsername()).
		Where("status != ?", entity.UserStatusDeleted).Get(user)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !has {
		return nil, nil
	}
	return user, nil
}

// existingTitles normalized titles of every question that is not deleted -> question id
func (s *KBImportService) existingTitles(ctx context.Context) (map[string]string, error) {
	questions := make([]*entity.Question, 0)
	err := s.data.DB.Context(ctx).Cols("id", "title").Where("status != ?", entity.QuestionStatusDeleted).Find(&questions)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	titles := make(map[string]string, len(questions))
	for _, q := range questions {
		titles[normalizeTitle(q.Title)] = q.ID
	}
	return titles, nil
}

// existingTags which of the slugs exist
func (s *KBImportService) existingTags(ctx context.Context, slugs []string) (map[string]bool, error) {
	found := make(map[string]bool)
	if len(slugs) == 0 {
		return found, nil
	}
	tags, err := s.tagCommonService.GetTagListByNames(ctx, append([]string{}, slugs...))
	if err != nil {
		return nil, err
	}
	for _, t := range tags {
		found[strings.ToLower(t.SlugName)] = true
	}
	return found, nil
}

// Check parses the rows and tells which would be published, which already exist and which tags are unknown
func (s *KBImportService) Check(ctx context.Context, req *schema.KBImportCheckReq) (*schema.KBImportCheckResp, error) {
	rows, delimiter := parseRows(req.Content)
	if len(rows) == 0 {
		return nil, errors.BadRequest(reason.RequestFormatError)
	}
	resp := &schema.KBImportCheckResp{Delimiter: delimiter, Rows: rows, Statuses: make(map[string]int),
		Username: accountUsername(), BaseTag: baseTag(), UnknownTags: []string{}}
	user, err := s.account(ctx)
	if err != nil {
		return nil, err
	}
	if user != nil {
		resp.AccountFound, resp.DisplayName = true, user.DisplayName
	}
	titles, err := s.existingTitles(ctx)
	if err != nil {
		return nil, err
	}
	allTags := []string{baseTag()}
	for _, row := range rows {
		allTags = append(allTags, row.Tags...)
	}
	known, err := s.existingTags(ctx, allTags)
	if err != nil {
		return nil, err
	}
	resp.BaseTagFound = known[baseTag()]
	unknown := make(map[string]bool)
	for _, row := range rows {
		if row.Status == schema.KBImportStatusNew {
			if id, ok := titles[normalizeTitle(row.Question)]; ok {
				row.Status, row.QuestionID = schema.KBImportStatusExists, id
			}
		}
		for _, t := range row.Tags {
			if !known[t] && !unknown[t] {
				unknown[t] = true
				resp.UnknownTags = append(resp.UnknownTags, t)
			}
		}
		resp.Statuses[row.Status]++
	}
	return resp, nil
}

// Publish one pair: question + answer by the knowledge-base account, the answer accepted. Unknown tags are
// skipped (new tags are created by the admin on purpose); a question with the same title is not created twice.
func (s *KBImportService) Publish(ctx context.Context, req *schema.KBImportRowReq) (*schema.KBImportRowResp, error) {
	user, err := s.account(ctx)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	title := strings.Join(strings.Fields(req.Question), " ")
	titles, err := s.existingTitles(ctx)
	if err != nil {
		return nil, err
	}
	if id, ok := titles[normalizeTitle(title)]; ok {
		return &schema.KBImportRowResp{QuestionID: id, Status: schema.KBImportStatusExists}, nil
	}

	slugs := []string{baseTag()}
	for _, t := range req.Tags {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" && t != baseTag() {
			slugs = append(slugs, t)
		}
	}
	known, err := s.existingTags(ctx, slugs)
	if err != nil {
		return nil, err
	}
	tags := make([]*schema.TagItem, 0, len(slugs))
	for _, slug := range slugs {
		if known[slug] {
			tags = append(tags, &schema.TagItem{SlugName: slug, DisplayName: slug})
		}
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "import w panelu administratora"
	}
	qReq := &schema.QuestionAdd{
		Title:   title,
		Content: title + "\n\nWpis bazy wiedzy — odpowiedź poniżej.",
		Tags:    tags,
		UserID:  user.ID,
	}
	qReq.CanUseReservedTag = true
	if _, err = qReq.Check(); err != nil {
		return nil, err
	}
	info, err := s.questionService.AddQuestion(ctx, qReq)
	if err != nil {
		return nil, err
	}
	question, ok := info.(*schema.QuestionInfoResp)
	if !ok || question == nil {
		return nil, errors.InternalServer(reason.UnknownError)
	}

	aReq := &schema.AnswerAddReq{
		QuestionID: question.ID,
		Content:    strings.TrimSpace(req.Answer) + strings.Replace(footer, "%s", source, 1),
		UserID:     user.ID,
	}
	if _, err = aReq.Check(); err != nil {
		return nil, err
	}
	answerID, err := s.answerService.Insert(ctx, aReq)
	if err != nil {
		return nil, err
	}
	resp := &schema.KBImportRowResp{QuestionID: question.ID, UrlTitle: question.UrlTitle,
		Status: schema.KBImportStatusPublished}
	if question.Status != entity.QuestionStatusAvailable {
		resp.Status = schema.KBImportStatusPending // the reviewer queued it (Admin → To approve)
	}
	if err = s.answerService.AcceptAnswer(ctx, &schema.AcceptAnswerReq{QuestionID: question.ID, AnswerID: answerID,
		UserID: user.ID}); err != nil {
		log.Warnf("kb import: accepting the answer to %s failed: %v", question.ID, err)
	}

	s.activityLogService.LogDetail(ctx, &entity.ActivityLog{UserID: req.LoginUserID, Action: ActionKBImport,
		ObjectType: constant.QuestionObjectType, ObjectID: question.ID, TargetUserID: user.ID},
		activity_log.Detail{"title": title, "source": source, "tags": len(tags)})
	return resp, nil
}
