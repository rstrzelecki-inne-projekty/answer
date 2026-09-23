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
	"time"

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

// §3 i §3.4–3.5 regulaminu konkursu „Pomocnik Automatyzacji"
func TestContestAnswerPoints_RulesFromTheContestTable(t *testing.T) {
	require.Equal(t, 10.0, contestAnswerPoints(true, false, false), "rozwiązanie")
	require.Equal(t, 5.0, contestAnswerPoints(false, true, false), "odpowiedź z głosem w górę")
	require.Equal(t, 15.0, contestAnswerPoints(true, true, false), "rozwiązanie + głos, czapka 15")
	require.Equal(t, 0.0, contestAnswerPoints(false, false, false), "sama odpowiedź nie punktuje automatycznie")
	require.Equal(t, 7.5, contestAnswerPoints(true, true, true), "odpowiedź na własne pytanie = pół stawki")
	require.Equal(t, 2.5, contestAnswerPoints(false, true, true))
	require.Equal(t, 0.0, contestAnswerPoints(false, false, true))
}

func TestContestTotalPoints_QuestionsCappedAt40Percent(t *testing.T) {
	// 30 PP z odpowiedzi → pytania mogą dołożyć najwyżej 20 PP (20/50 = 40%)
	require.Equal(t, 50.0, contestTotalPoints(30, 40))
	require.Equal(t, 34.0, contestTotalPoints(30, 4), "poniżej czapki nic nie przepada")
	require.Equal(t, 0.0, contestTotalPoints(0, 12), "same pytania bez odpowiedzi nie punktują")
	require.Equal(t, 12.5, contestTotalPoints(7.5, 5))
}

func TestContestQuarterStart_FirstDayOfTheCalendarQuarter(t *testing.T) {
	for _, c := range []struct {
		now   time.Time
		start time.Time
	}{
		{time.Date(2026, 9, 22, 17, 40, 0, 0, time.UTC), time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2027, 1, 12, 9, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
	} {
		require.Equal(t, c.start, contestQuarterStart(c.now))
	}
}

func TestContestExcluded_ServiceAccountsOutOfTheContest(t *testing.T) {
	require.True(t, contestExcludedUsernames["baza-wiedzy"])
	require.True(t, contestExcludedUsernames["asystent-ai"], "§2.2 wyklucza konta serwisowe")
	require.False(t, contestExcludedUsernames["anna-kowalczyk"])
}

func TestContestMonthStart_FirstDayOfTheCalendarMonth(t *testing.T) {
	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		contestMonthStart(time.Date(2026, 9, 23, 18, 5, 0, 0, time.UTC)))
	require.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		contestMonthStart(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)))
}

// §4.7: ranking miesięczny to wycinek tego samego kwartału
func TestContestItemsSince_KeepsOnlyTheEventsOfTheShorterPeriod(t *testing.T) {
	monthStart := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	items := map[string][]*contestItem{
		"10000000000000001": {
			{Kind: contestKindSolution, Points: 10, At: time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)},
			{Kind: contestKindSolution, Points: 10, At: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)},
		},
		"10000000000000002": {
			{Kind: contestKindQuestionUpvote, Points: 4, At: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)},
		},
	}

	month := contestItemsSince(items, monthStart)
	require.Len(t, month, 1, "użytkownik bez zdarzeń w tym miesiącu wypada z rankingu miesięcznego")
	require.Len(t, month["10000000000000001"], 1)
	require.Equal(t, 10.0, contestAggregate(month["10000000000000001"]).answerPoints)
	require.Equal(t, 20.0, contestAggregate(items["10000000000000001"]).answerPoints, "kwartał liczy oba")
}

func TestContestAggregate_QuestionTagScoresLikeAnUpvote(t *testing.T) {
	at := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	score := contestAggregate([]*contestItem{
		{Kind: contestKindQuestionKB, Points: 4, At: at},
		{Kind: contestKindSolution, Points: 10, At: at},
		{Kind: contestKindKnowledgeBonus, Points: 5, At: at},
	})
	require.Equal(t, 4.0, score.questionPoints)
	require.Equal(t, 15.0, score.answerPoints, "bonus za uzupełnienie bazy wiedzy liczy się jak punkt z odpowiedzi")
	require.Equal(t, 1, score.solved)
}
