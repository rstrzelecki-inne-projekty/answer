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

package schema

// [cd] knowledge base import from the admin panel (question–answer pairs)

// Row statuses
const (
	KBImportStatusNew       = "new"       // check: would be published
	KBImportStatusExists    = "exists"    // a question with the same title already exists
	KBImportStatusDuplicate = "duplicate" // repeats an earlier row of the file
	KBImportStatusInvalid   = "invalid"   // see message
	KBImportStatusPublished = "published" // published and visible
	KBImportStatusPending   = "pending"   // published, waiting in Admin → To approve
)

// KBImportCheckReq the pasted / uploaded TSV
type KBImportCheckReq struct {
	Content string `validate:"required,lte=5000000" json:"content"`
}

// KBImportRow one row of the file
type KBImportRow struct {
	Line     int      `json:"line"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Tags     []string `json:"tags"`
	Status   string   `json:"status"`
	// invalid: missing_field, question_length (6-150 characters), answer_too_short, too_many_rows, csv_error: …
	Message     string `json:"message,omitempty"`
	DuplicateOf int    `json:"duplicate_of,omitempty"`
	QuestionID  string `json:"question_id,omitempty"`
}

// KBImportCheckResp rows with statuses + what the publication needs (the account and the base tag)
type KBImportCheckResp struct {
	Delimiter    string         `json:"delimiter"`
	Rows         []*KBImportRow `json:"rows"`
	Statuses     map[string]int `json:"statuses"`
	UnknownTags  []string       `json:"unknown_tags"`
	Username     string         `json:"username"`
	DisplayName  string         `json:"display_name"`
	AccountFound bool           `json:"account_found"`
	BaseTag      string         `json:"base_tag"`
	BaseTagFound bool           `json:"base_tag_found"`
}

// KBImportRowReq one pair to publish
type KBImportRowReq struct {
	Question    string   `validate:"required,notblank,gte=6,lte=150" json:"question"`
	Answer      string   `validate:"required,notblank,gte=6,lte=60000" json:"answer"`
	Tags        []string `validate:"lte=10" json:"tags"`
	Source      string   `validate:"lte=200" json:"source"`
	LoginUserID string   `json:"-"`
}

// KBImportRowResp the published question
type KBImportRowResp struct {
	QuestionID string `json:"question_id"`
	UrlTitle   string `json:"url_title,omitempty"`
	Status     string `json:"status"`
}
