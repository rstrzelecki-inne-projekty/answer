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

package badge

import (
	"testing"

	"github.com/apache/answer/internal/entity"
	"github.com/stretchr/testify/require"
)

// stopnie z badges.json: szeregowy 1, kapral 6, sierżant 15, kapitan 80
func rankBadges() []*entity.Badge {
	return []*entity.Badge{
		{ID: "10090000000000013", Name: "szeregowy", Single: entity.BadgeSingleAward, Param: `{"amount":"1"}`},
		{ID: "10090000001128368", Name: "kapral", Single: entity.BadgeSingleAward, Param: `{"amount":"6"}`},
		{ID: "10090000001906889", Name: "sierzant", Single: entity.BadgeSingleAward, Param: `{"amount":"15"}`},
		{ID: "10090000001527725", Name: "kapitan", Single: entity.BadgeSingleAward, Param: `{"amount":"80"}`},
		{ID: "10090000001807370", Name: "general", Single: entity.BadgeSingleAward, Param: `{"amount":"200"}`},
	}
}

func TestHighestTierReached_OnlyTheTopRankIsHandedOut(t *testing.T) {
	tiers := rankBadges()

	require.Nil(t, highestTierReached(tiers, 0), "bez zaakceptowanych odpowiedzi nie ma stopnia")
	require.Equal(t, "szeregowy", highestTierReached(tiers, 1).Name)
	require.Equal(t, "kapral", highestTierReached(tiers, 6).Name)
	require.Equal(t, "kapral", highestTierReached(tiers, 14).Name)
	require.Equal(t, "sierzant", highestTierReached(tiers, 15).Name, "przeskok kilku progów naraz = jeden stopień")
	require.Equal(t, "kapitan", highestTierReached(tiers, 199).Name)
	require.Equal(t, "general", highestTierReached(tiers, 200).Name)
}

func TestHighestHeldTierOf_KnowsTheRankTheUserAlreadyHas(t *testing.T) {
	tiers := rankBadges()

	require.Equal(t, int64(0), highestHeldTierOf(tiers, map[string]bool{}))
	require.Equal(t, int64(80), highestHeldTierOf(tiers, map[string]bool{"10090000001527725": true}),
		"kapitan nadany ręcznie przez admina")
	require.Equal(t, int64(80), highestHeldTierOf(tiers, map[string]bool{
		"10090000001527725": true, "10090000000000013": true,
	}), "liczy się najwyższy posiadany, nie ostatni nadany")
}

// Właściwy błąd: użytkownik z Kapitanem dostawał Szeregowego przy pierwszej zaakceptowanej odpowiedzi.
func TestRankNotAwardedBelowTheRankAlreadyHeld(t *testing.T) {
	tiers := rankBadges()
	held := highestHeldTierOf(tiers, map[string]bool{"10090000001527725": true}) // kapitan

	best := highestTierReached(tiers, 1) // pierwsza zaakceptowana odpowiedź
	require.Equal(t, "szeregowy", best.Name)
	require.False(t, best.GetIntParam("amount") > held, "szeregowy nie może trafić do kapitana")

	best = highestTierReached(tiers, 200) // dorobił się generalskiego pułapu
	require.Equal(t, "general", best.Name)
	require.True(t, best.GetIntParam("amount") > held, "wyższy stopień nadal się należy")

	// a kapitan zdobyty naturalnie nie jest nadawany drugi raz
	heldGeneral := highestHeldTierOf(tiers, map[string]bool{"10090000001807370": true})
	require.False(t, highestTierReached(tiers, 200).GetIntParam("amount") > heldGeneral)
}

func TestSplitSingleAwardBadges_MultiAwardKeepsOldBehaviour(t *testing.T) {
	badges := append(rankBadges(), &entity.Badge{
		ID: "10090000000000014", Name: "trafiona-odpowiedz", Single: entity.BadgeMultiAward, Param: `{"amount":"3"}`,
	})

	tiers, others := splitSingleAwardBadges(badges)
	require.Len(t, tiers, 5)
	require.Len(t, others, 1)
	require.Equal(t, "trafiona-odpowiedz", others[0].Name)
}
