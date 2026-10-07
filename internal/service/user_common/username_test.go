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

package usercommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsernameBase(t *testing.T) {
	assert.Equal(t, "marzena-wasilek", UsernameBase("Marzena Wasiłek"))
	assert.Equal(t, "jan.kowalski", UsernameBase("Jan.Kowalski"))
	assert.Equal(t, "zazolc-gesla-jazn", UsernameBase("Zażółć   gęślą-jaźń!"))
	assert.Equal(t, "zo", UsernameBase("żó"))
	assert.LessOrEqual(t, len(UsernameBase("abcdefghijklmnopqrstuvwxyz0123456789")), 26)
}

func TestLoginCandidates(t *testing.T) {
	// [cd] the e-mail before @ first (people know it), then the name; too short or empty ones are dropped
	assert.Equal(t, []string{"rstrzelecki", "rafal-strzelecki"}, LoginCandidates("RStrzelecki@cashdirector.pl", "Rafał Strzelecki", "rafal strzelecki"))
	assert.Equal(t, []string{"rafal"}, LoginCandidates("", "Rafał"))
	assert.Equal(t, []string{"a.b"}, LoginCandidates("a.b@x.pl", "ż"))
	assert.Empty(t, LoginCandidates("", ""))
}
