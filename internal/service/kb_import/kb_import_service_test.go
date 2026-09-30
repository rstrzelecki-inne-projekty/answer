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

package kb_import

import (
	"testing"

	"github.com/apache/answer/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestParseRows(t *testing.T) {
	t.Run("header in any order, tags and a quoted multi-line answer", func(t *testing.T) {
		rows, delim := parseRows("Tagi\tPytanie\tOdpowiedź\r\n" +
			"KSeF, zus-platnik\tJak wysłać  fakturę do KSeF?\t\"Krok 1: otwórz moduł.\nKrok 2: wyślij.\"\r\n" +
			"\t\t\r\n" +
			"\tCo to jest AOA w skrócie?\tAutomatyczna obsługa: linia 1\\nlinia 2\r\n")
		assert.Equal(t, "tab", delim)
		assert.Len(t, rows, 2)
		assert.Equal(t, "Jak wysłać fakturę do KSeF?", rows[0].Question)
		assert.Equal(t, "Krok 1: otwórz moduł.\nKrok 2: wyślij.", rows[0].Answer)
		assert.Equal(t, []string{"ksef", "zus-platnik"}, rows[0].Tags)
		assert.Equal(t, schema.KBImportStatusNew, rows[0].Status)
		assert.Equal(t, "Automatyczna obsługa: linia 1\nlinia 2", rows[1].Answer)
		assert.Equal(t, []string{}, rows[1].Tags)
	})
	t.Run("no header: question, answer, tags", func(t *testing.T) {
		rows, _ := parseRows("Jak zamknąć miesiąc w AO?\tMenu Okresy, potem Zamknij.\tzamykanie-miesiaca\n")
		assert.Equal(t, "Jak zamknąć miesiąc w AO?", rows[0].Question)
		assert.Equal(t, []string{"zamykanie-miesiaca"}, rows[0].Tags)
	})
	t.Run("semicolon csv from Excel", func(t *testing.T) {
		rows, delim := parseRows("pytanie;odpowiedź\nJak dodać kontrahenta?;Kartoteki, potem Dodaj.\n")
		assert.Equal(t, "semicolon", delim)
		assert.Equal(t, "Kartoteki, potem Dodaj.", rows[0].Answer)
	})
	t.Run("invalid rows and duplicates (title without case and spaces)", func(t *testing.T) {
		rows, _ := parseRows("pytanie\todpowiedź\nKrótk\tOdpowiedź długa\nPytanie bez odpowiedzi?\t\n" +
			"Poprawne pytanie?\tok\nJak to działa?\tTak to działa.\njak  TO działa?\tInaczej.\n")
		assert.Equal(t, "question_length", rows[0].Message)
		assert.Equal(t, "missing_field", rows[1].Message)
		assert.Equal(t, "answer_too_short", rows[2].Message)
		assert.Equal(t, schema.KBImportStatusNew, rows[3].Status)
		assert.Equal(t, schema.KBImportStatusDuplicate, rows[4].Status)
		assert.Equal(t, 5, rows[4].DuplicateOf)
	})
}
