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
	"context"
	"strings"
	"unicode"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/service/mention"
	"github.com/apache/answer/pkg/converter"
	"golang.org/x/text/unicode/norm"
)

// ł has no decomposition, every other Polish letter drops its mark after NFD
var usernameFold = strings.NewReplacer("ł", "l", "Ł", "L")

// UsernameBase an ASCII login made of the text: Answer accepts only [A-Za-z0-9_.-] in usernames, so
// "Marzena Wasiłek" becomes "marzena-wasilek"; other characters are dropped. Up to 26 characters, leaving room for
// the number MakeUsername appends when the login is taken.
func UsernameBase(text string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(usernameFold.Replace(text)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_'):
			b.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r) || r == '-':
			b.WriteRune('-')
		}
	}
	s := strings.Trim(strings.Join(strings.FieldsFunc(b.String(), func(r rune) bool { return r == '-' }), "-"), "._")
	if len(s) > 26 {
		s = strings.Trim(s[:26], "-._")
	}
	return s
}

// RewriteMentions [cd] resolves "@token" in markdown to profile links (see package mention) and returns the
// markdown, its HTML and the logins of everyone mentioned. Matching by e-mail (full or the part before @) only
// for authors from MENTION_EMAIL_DOMAINS; by login for everyone. Used for comments, questions and answers.
func (us *UserCommon) RewriteMentions(ctx context.Context, authorUserID, text string) (string, string, []string) {
	byEmail := false
	if author, exist, err := us.userRepo.GetByUserID(ctx, authorUserID); err == nil && exist {
		byEmail = mention.EmailVisibleFor(author.EMail)
	}
	rewritten, usernames := mention.Rewrite(text, func(token string) (string, string, bool) {
		token = strings.ToLower(token)
		local, domain, isEmail := strings.Cut(token, "@")
		if !isEmail {
			if u, exist, err := us.userRepo.GetByUsername(ctx, token); err == nil && exist && u.Status == entity.UserStatusAvailable {
				return u.DisplayName, u.Username, true
			}
		}
		if !byEmail {
			return "", "", false
		}
		if u, exist, err := us.userRepo.GetByMailbox(ctx, local, domain); err == nil && exist {
			return u.DisplayName, u.Username, true
		}
		return "", "", false
	})
	return rewritten, converter.Markdown2HTML(rewritten), usernames
}

// LoginCandidates logins to try for a new account, best first: the e-mail before @ (people know it from their
// mailbox, so "@rstrzelecki" works as a mention), then the names. Empty, too short and repeated ones are dropped.
func LoginCandidates(email string, names ...string) []string {
	local, _, _ := strings.Cut(email, "@")
	var out []string
	seen := map[string]bool{}
	for _, raw := range append([]string{local}, names...) {
		if c := UsernameBase(raw); len(c) >= 2 && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}
