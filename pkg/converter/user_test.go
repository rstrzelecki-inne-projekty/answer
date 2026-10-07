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

package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMentionUsernameList(t *testing.T) {
	// [cd] the login comes from the profile URL, the text in brackets is the display name
	text := "cześć [@Rafał Strzelecki](/users/rstrzelecki) i [@old-login](/users/old-login), [nie wzmianka](/users/x)"
	assert.Equal(t, []string{"rstrzelecki", "old-login"}, GetMentionUsernameList(text))
	assert.Empty(t, GetMentionUsernameList("bez wzmianek @ktoś"))
}
