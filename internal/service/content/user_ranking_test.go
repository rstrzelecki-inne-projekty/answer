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

package content

import (
	"sort"
	"testing"

	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/stretchr/testify/require"
)

func rankingEntry(username string, reputation, rankAmount int) *schema.UserRankingSimpleInfo {
	entry := &schema.UserRankingSimpleInfo{Username: username, Rank: reputation}
	if rankAmount > 0 {
		entry.Prestige = &schema.UserPrestige{RankBadgeAmount: rankAmount}
	}
	return entry
}

func TestHigherRanking_InsigniaBeatsReputation(t *testing.T) {
	general := rankingEntry("general", 120, 200)
	sergeant := rankingEntry("sierzant", 900, 15)
	noRank := rankingEntry("bez-stopnia", 4000, 0)

	list := []*schema.UserRankingSimpleInfo{noRank, sergeant, general}
	sort.SliceStable(list, func(i, j int) bool { return higherRanking(list[i], list[j]) })

	require.Equal(t, []string{"general", "sierzant", "bez-stopnia"},
		[]string{list[0].Username, list[1].Username, list[2].Username})
}

func TestHigherRanking_ReputationBreaksTheTie(t *testing.T) {
	quiet := rankingEntry("cichy", 300, 80)
	loud := rankingEntry("glosny", 900, 80)

	require.True(t, higherRanking(loud, quiet))
	require.False(t, higherRanking(quiet, loud))
}

func TestExcludedFromRanking_HidesTheKnowledgeBaseAccount(t *testing.T) {
	require.True(t, excludedFromRanking(&entity.User{Username: "baza-wiedzy"}))
	require.True(t, excludedFromRanking(&entity.User{Username: "Baza-Wiedzy"}))
	require.False(t, excludedFromRanking(&entity.User{Username: "asystent-ai"}))
	require.False(t, excludedFromRanking(&entity.User{Username: "anna-kowalczyk"}))
	require.True(t, excludedFromRanking(nil))
}
