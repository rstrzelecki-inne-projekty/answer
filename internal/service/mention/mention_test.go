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

package mention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func lookup(token string) (displayName, username string, ok bool) {
	switch token {
	case "rstrzelecki", "rstrzelecki@cashdirector.pl", "a1b2c3":
		return "Rafał Strzelecki", "a1b2c3", true
	case "marzena-wasilek":
		return "Marzena Wasiłek", "marzena-wasilek", true
	}
	return "", "", false
}

func TestRewrite(t *testing.T) {
	t.Run("login, e-mail and local part become links with the display name", func(t *testing.T) {
		text, users := Rewrite("@rstrzelecki zobacz, @rstrzelecki@cashdirector.pl. Dzięki @a1b2c3!", lookup)
		assert.Equal(t, "[@Rafał Strzelecki](/users/a1b2c3) zobacz, [@Rafał Strzelecki](/users/a1b2c3). Dzięki [@Rafał Strzelecki](/users/a1b2c3)!", text)
		assert.Equal(t, []string{"a1b2c3"}, users)
	})
	t.Run("existing links are kept and counted, unknown tokens and e-mails in text stay", func(t *testing.T) {
		text, users := Rewrite("[@Rafał Strzelecki](/users/a1b2c3) i @nieznany, pisz na pomoc@czasobslugi.pl, @marzena-wasilek", lookup)
		assert.Equal(t, "[@Rafał Strzelecki](/users/a1b2c3) i @nieznany, pisz na pomoc@czasobslugi.pl, [@Marzena Wasiłek](/users/marzena-wasilek)", text)
		assert.Equal(t, []string{"a1b2c3", "marzena-wasilek"}, users)
	})
	t.Run("trailing punctuation, line start and code are handled", func(t *testing.T) {
		text, users := Rewrite("@rstrzelecki.\n@marzena-wasilek- ok `@rstrzelecki`", lookup)
		assert.Equal(t, "[@Rafał Strzelecki](/users/a1b2c3).\n[@Marzena Wasiłek](/users/marzena-wasilek)- ok `@rstrzelecki`", text)
		assert.Equal(t, []string{"a1b2c3", "marzena-wasilek"}, users)
	})
	t.Run("a hand-written link cannot show one name and point at another; unknown logins become text", func(t *testing.T) {
		text, users := Rewrite("[@Marzena Wasiłek](/users/a1b2c3) i [@Ktoś](/users/nie-ma) oraz [@x](/users/rstrzelecki)", lookup)
		assert.Equal(t, "[@Rafał Strzelecki](/users/a1b2c3) i @Ktoś oraz [@Rafał Strzelecki](/users/a1b2c3)", text)
		assert.Equal(t, []string{"a1b2c3"}, users)
	})
	t.Run("nothing to do", func(t *testing.T) {
		text, users := Rewrite("zwykły tekst", lookup)
		assert.Equal(t, "zwykły tekst", text)
		assert.Empty(t, users)
	})
}

func TestEmailVisibleFor(t *testing.T) {
	t.Setenv("MENTION_EMAIL_DOMAINS", "")
	assert.True(t, EmailVisibleFor("Jan.Kowalski@CashDirector.pl"))
	assert.False(t, EmailVisibleFor("jan@biuro.cashdirector.pl"))
	assert.False(t, EmailVisibleFor("jan@fakecashdirector.pl"))
	assert.False(t, EmailVisibleFor(""))
	t.Setenv("MENTION_EMAIL_DOMAINS", "example.com, biuro.cashdirector.pl")
	assert.True(t, EmailVisibleFor("jan@biuro.cashdirector.pl"))
	assert.True(t, EmailVisibleFor("smoke@example.com"))
	assert.False(t, EmailVisibleFor("jan@cashdirector.pl"))
}
