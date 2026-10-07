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

package user_admin

import (
	"testing"

	"github.com/apache/answer/internal/schema"
	usercommon "github.com/apache/answer/internal/service/user_common"
	"github.com/stretchr/testify/assert"
)

func TestParseUserImport(t *testing.T) {
	t.Run("excel semicolon export with a Polish header and first/last name columns", func(t *testing.T) {
		rows, delim := parseUserImport("\ufeffImię;Nazwisko;E-mail\r\nAnna;Kowalska;Anna.Kowalska@Biuro.Morganizer.pl\r\n;;\r\nJan;Nowak;jan@biuro.cashdirector.pl\r\n")
		assert.Equal(t, "semicolon", delim)
		assert.Len(t, rows, 2)
		assert.Equal(t, "Anna Kowalska", rows[0].DisplayName)
		assert.Equal(t, "anna.kowalska@biuro.morganizer.pl", rows[0].Email)
		assert.Equal(t, schema.ImportUserStatusNew, rows[0].Status)
		assert.Equal(t, 2, rows[0].Line)
		assert.Equal(t, 4, rows[1].Line)
	})
	t.Run("tsv without a header, e-mail found by @", func(t *testing.T) {
		rows, delim := parseUserImport("jan@biuro.cashdirector.pl\tJan  Nowak\nEwa Lis\tewa@biuro.morganizer.pl\n")
		assert.Equal(t, "tab", delim)
		assert.Equal(t, "Jan Nowak", rows[0].DisplayName)
		assert.Equal(t, "ewa@biuro.morganizer.pl", rows[1].Email)
		assert.Equal(t, "Ewa Lis", rows[1].DisplayName)
	})
	t.Run("comma csv with a quoted name and a header in English", func(t *testing.T) {
		rows, delim := parseUserImport("name,email\n\"Nowak, Jan\",jan@x.pl\n")
		assert.Equal(t, "comma", delim)
		assert.Equal(t, "Nowak, Jan", rows[0].DisplayName)
	})
	t.Run("invalid rows and duplicates", func(t *testing.T) {
		rows, _ := parseUserImport("name;email\nJan;jan@x.pl\nJan bis;JAN@x.pl\nX;x@x.pl\nBez maila;\nZły;zly@@x\nBardzo długie imię i nazwisko ponad limit;a@x.pl\n")
		assert.Equal(t, schema.ImportUserStatusNew, rows[0].Status)
		assert.Equal(t, schema.ImportUserStatusDuplicate, rows[1].Status)
		assert.Equal(t, 2, rows[1].DuplicateOf)
		assert.Equal(t, "invalid_name", rows[2].Message)
		assert.Equal(t, "missing_email", rows[3].Message)
		assert.Equal(t, "invalid_email", rows[4].Message)
		assert.Equal(t, "invalid_name", rows[5].Message)
	})
}

func TestUsernameBase(t *testing.T) {
	assert.Equal(t, "marzena-wasilek", usercommon.UsernameBase("Marzena Wasiłek"))
	assert.Equal(t, "zaneta-zolc-gesla", usercommon.UsernameBase("Żaneta  Żółć-Gęśla"))
	assert.Equal(t, "lukasz-oneill", usercommon.UsernameBase("Łukasz O'Neill."))
	assert.Equal(t, "anna.kowalska", usercommon.UsernameBase("anna.kowalska"))
	assert.Equal(t, "", usercommon.UsernameBase("你好"))
	assert.Len(t, usercommon.UsernameBase("Bardzo Długie Imię Oraz Nazwisko Dwuczłonowe"), 26)
}
