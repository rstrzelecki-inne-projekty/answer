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

package checker

import "strings"

// EmailInAllowEmailDomain the e-mail domain equals an allowed domain or is its subdomain ("example.com" allows
// a@example.com and a@team.example.com, not a@fakeexample.com). Case and a leading "@" in the setting are ignored.
func EmailInAllowEmailDomain(email string, allowEmailDomains []string) bool {
	if len(allowEmailDomains) == 0 {
		return true
	}

	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(email[at+1:]))
	for _, domain := range allowEmailDomains {
		domain = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(domain), "@"))
		if domain == "" {
			continue
		}
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}

	return false
}
