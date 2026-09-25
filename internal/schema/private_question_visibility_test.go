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

import (
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"github.com/stretchr/testify/require"
)

const (
	asker     = "10000000000000001"
	stranger  = "10000000000000002"
	answerer  = "10000000000000003"
	qPrivate  = entity.QuestionPrivate
	qNotPriv  = entity.QuestionNotPrivate
	available = entity.QuestionStatusAvailable
)

// [cd] a private thread is visible to the person who asked it and to the staff, to nobody else
func TestCheckVisibility_PrivateQuestion(t *testing.T) {
	question := &SimpleObjectInfo{
		ObjectType:            constant.QuestionObjectType,
		ObjectCreatorUserID:   asker,
		QuestionCreatorUserID: asker,
		QuestionStatus:        available,
		QuestionShow:          entity.QuestionShow,
		QuestionPrivate:       qPrivate,
	}

	require.NoError(t, question.CheckVisibility(asker, false), "the person who asked always sees it")
	require.NoError(t, question.CheckVisibility(stranger, true), "an administrator or a moderator sees it")
	require.Error(t, question.CheckVisibility(stranger, false), "anybody else must not")
	require.Error(t, question.CheckVisibility("", false), "an anonymous reader must not")

	question.QuestionPrivate = qNotPriv
	require.NoError(t, question.CheckVisibility(stranger, false), "a public thread stays public")
}

// the answers and comments of a private thread follow the thread, including their own authors
func TestCheckVisibility_AnswerInsideAPrivateQuestion(t *testing.T) {
	answer := &SimpleObjectInfo{
		ObjectType:            constant.AnswerObjectType,
		ObjectCreatorUserID:   answerer,
		QuestionID:            "10010000000000001",
		QuestionCreatorUserID: asker,
		QuestionStatus:        available,
		QuestionShow:          entity.QuestionShow,
		QuestionPrivate:       qPrivate,
		AnswerStatus:          entity.AnswerStatusAvailable,
	}

	require.NoError(t, answer.CheckVisibility(asker, false))
	require.NoError(t, answer.CheckVisibility(stranger, true))
	require.Error(t, answer.CheckVisibility(answerer, false),
		"the person who answered loses access together with the thread")
	require.Error(t, answer.CheckVisibility(stranger, false))
}
