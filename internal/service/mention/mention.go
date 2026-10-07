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

// Package mention [cd] resolves "@someone" written in comments to a profile link with the person's display name.
// The frontend inserts "[@Display Name](/users/login)" for people picked from the suggestion list; everything typed by
// hand — "@login", "@mailbox" (the e-mail before @) or "@mailbox@domain" — is matched here on the server.
package mention

import (
	"os"
	"regexp"
	"strings"

	"github.com/apache/answer/pkg/converter"
)

// Lookup finds a user by a token typed after "@"; ok is false when nobody matches (the token is left as is).
type Lookup func(token string) (displayName, username string, ok bool)

// "@" at the start or after a separator (not inside a word, an e-mail address or an existing "[@…](…)" link),
// then a login or an e-mail address
var tokenRe = regexp.MustCompile(`(^|[^\p{L}\p{N}_\[/@])@([\w.\-]+(?:@[\w.\-]+)?)`)

var codeSpanRe = regexp.MustCompile("`[^`\n]*`")

// an existing mention link; the text in brackets is not trusted (see normalizeLinks)
var linkRe = regexp.MustCompile(`\[@([^\]]+)\]\(/users/([^)\s]+)\)`)

// normalizeLinks rewrites every "[@anything](/users/login)" so the name in brackets is the real display name of
// that login (a comment could otherwise show one person's name and link to another) and demotes links to a
// login nobody has to plain text.
func normalizeLinks(text string, lookup Lookup) string {
	return linkRe.ReplaceAllStringFunc(text, func(link string) string {
		m := linkRe.FindStringSubmatch(link)
		displayName, username, ok := lookup(m[2])
		if !ok || username == "" {
			return "@" + m[1]
		}
		if displayName == "" || strings.ContainsAny(displayName, "[]()") {
			displayName = username
		}
		return "[@" + displayName + "](/users/" + username + ")"
	})
}

// Rewrite replaces every resolvable "@token" in markdown with "[@Display Name](/users/login)" and returns the
// logins of all mentioned people (existing links included), in order of appearance, without duplicates.
func Rewrite(text string, lookup Lookup) (string, []string) {
	text = normalizeLinks(text, lookup)
	code := codeSpanRe.FindAllStringIndex(text, -1)
	inCode := func(pos int) bool {
		for _, c := range code {
			if pos >= c[0] && pos < c[1] {
				return true
			}
		}
		return false
	}
	var b strings.Builder
	last := 0
	for _, m := range tokenRe.FindAllStringSubmatchIndex(text, -1) {
		atPos, tokStart, tokEnd := m[3], m[4], m[5]
		token := strings.TrimRight(text[tokStart:tokEnd], ".-_")
		if token == "" || inCode(atPos) {
			continue
		}
		displayName, username, ok := lookup(token)
		if !ok || username == "" {
			continue
		}
		if displayName == "" || strings.ContainsAny(displayName, "[]()") {
			displayName = username
		}
		b.WriteString(text[last:atPos])
		b.WriteString("[@" + displayName + "](/users/" + username + ")")
		last = tokStart + len(token)
	}
	b.WriteString(text[last:])
	out := b.String()
	seen := map[string]bool{}
	var usernames []string
	for _, u := range converter.GetMentionUsernameList(out) {
		if !seen[u] {
			seen[u] = true
			usernames = append(usernames, u)
		}
	}
	return out, usernames
}

// EmailVisibleFor whether a person with this e-mail may see other people's e-mail addresses in mention
// suggestions and mention them by e-mail: only accounts in MENTION_EMAIL_DOMAINS (default cashdirector.pl, exact
// domain, so subcontractors at biuro.cashdirector.pl are out).
func EmailVisibleFor(email string) bool {
	_, domain, found := strings.Cut(strings.ToLower(strings.TrimSpace(email)), "@")
	if !found || domain == "" {
		return false
	}
	raw := strings.TrimSpace(os.Getenv("MENTION_EMAIL_DOMAINS"))
	if raw == "" {
		raw = "cashdirector.pl"
	}
	for _, d := range strings.Split(raw, ",") {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" && d == domain {
			return true
		}
	}
	return false
}
