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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmailInAllowEmailDomain(t *testing.T) {
	allowed := []string{"cashdirector.pl", "@Biuro.Morganizer.pl"}
	assert.True(t, EmailInAllowEmailDomain("a@cashdirector.pl", allowed))
	assert.True(t, EmailInAllowEmailDomain("A@CashDirector.PL", allowed))
	assert.True(t, EmailInAllowEmailDomain("a@biuro.cashdirector.pl", allowed))
	assert.True(t, EmailInAllowEmailDomain("a@biuro.morganizer.pl", allowed))
	assert.False(t, EmailInAllowEmailDomain("a@fakecashdirector.pl", allowed))
	assert.False(t, EmailInAllowEmailDomain("a@cashdirector.pl.evil.com", allowed))
	assert.False(t, EmailInAllowEmailDomain("a@morganizer.pl", allowed))
	assert.False(t, EmailInAllowEmailDomain("cashdirector.pl", allowed))
	assert.True(t, EmailInAllowEmailDomain("a@anything.com", nil))
	assert.False(t, EmailInAllowEmailDomain("a@x.pl", []string{" "}))
}
