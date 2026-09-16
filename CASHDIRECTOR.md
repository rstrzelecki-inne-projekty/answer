# Fork Apache Answer dla portalu Pomoc AO AOA (CashDirector)

Gałęzie:
- `main` — czysty upstream (`git fetch upstream && git merge --ff-only upstream/main`). Nigdy nie commitować bezpośrednio.
- `cashdirector` — `main` + nasze łatki. Z niej budowany jest obraz produkcyjny (repo `answers`, `Dockerfile`).

Konwencje łatek:
- jedna funkcjonalność = jeden commit, tytuł z prefiksem `[cd]`; bez poprawek „przy okazji”, bez zmian formatowania cudzych plików;
- przy nowym wydaniu upstreamu: `git rebase v2.x.y` gałęzi `cashdirector`, konflikty tylko w naszych plikach;
- `git format-patch upstream/main..cashdirector` = lista tego, czym się różnimy (miara długu — ma być krótka);
- co się da, oddajemy upstreamowi jako PR; po merge łatka znika stąd;
- każda łatka z logiką w Go ma test jednostkowy (fałszywki repozytoriów, bez bazy — wzór: `internal/service/badge/badge_award_admin_test.go`)
  albo scenariusz w `answers/dev/smoke.sh`; `go test ./internal/... ./pkg/...` musi przechodzić przed tagiem produkcyjnym.

Tagi produkcyjne: `cd-v<wersja upstreamu>-<numer>` (np. `cd-v2.0.2-1`). Obraz w ECR: `answer:<wersja>-cd<numer>`.

Pełne wytyczne: `answers/docs/fork-answer-wytyczne.md`. Zadania: Jira AA-26.

## Łatki na tej gałęzi
| # | Commit | Co | Status upstream |
|---|---|---|---|
| 7 | Sortowanie „Najwięcej wyświetleń” | `order=views` na liście pytań (`question.view_count DESC`), przycisk w `QUESTION_ORDER_KEYS`, i18n `question.views` | PR https://github.com/apache/answer/pull/1623 (otwarty 2026-09-16, gałąź `feat/most-viewed-order`); po merge usunąć łatkę przy rebase |
| 6 | Log aktywności społeczności | tabela `activity_log` + `internal/service/activity_log` (odbiorca kolejki zdarzeń; kolejka obsługuje wielu odbiorców), hooki: logowanie (`userRepo.UpdateLastLoginDate`), odznaki, review, `ChangeUserRank` (kontekst `WithRankContext`), role/blokady; admin API `/answer/admin/api/activity-log` (+`/actions`, `/export` TSV), beacon `POST /answer/api/v1/activity-log/view`, cron retencji `ACTIVITY_LOG_PAGEVIEW_DAYS` (90); UI Admin → Społeczność → Log aktywności | lokalne (AA-37); do rozważenia jako propozycja upstream |
| 5 | Przeładowanie przy nieaktualnym bundle | `ui/src/router/index.tsx`: nieudany `import()` strony (stary chunk po wdrożeniu) przeładowuje kartę raz (`sessionStorage cd_stale_chunk_reload`), zamiast ekranu „50X”; `internal/router/ui.go`: brakujący `/static/*` → 404 zamiast `index.html` (Caddy robi to samo na warstwie wyżej) | PR https://github.com/apache/answer/pull/1619 (otwarty 2026-09-15, gałąź `fix/stale-chunk-reload`); po merge usunąć łatkę przy rebase |
| 4 | Głos w dół i reputacja za komentarze | typy aktywności `comment.vote_down/voted_up/voted_down` (config ID 901–903; na istniejącej bazie seed przez `answers/17-configure-reputation.sh`), `getActivities` rozróżnia up/down dla komentarza, `vote_status` w odpowiedzi komentarza, kciuk w dół w `Comment/ActionBar`, licznik z odpowiedzi serwera | lokalne (AA-35); upstream ma tylko `comment.vote_up`=0 |
| 3 | Ręczne odznaki (admin) | API `POST/DELETE /answer/admin/api/badge/award` + komponent `AwardBadgeButton` (profil→Odznaki z cofaniem, strona odznaki, Admin→Odznaki, lista pytań, pytanie/odpowiedzi); powiadomienie w dzwonku przez `BadgeAwardService.Award` | kandydat do PR upstream (AA-34) |
| 2 | Content-Type uploadów | `AvatarThumb` ustawiał `image/<ext>` dla każdego pliku z `/uploads` (`image/svg` → przeglądarki nie renderują SVG); teraz `mime.TypeByExtension` z fallbackiem | PR https://github.com/apache/answer/pull/1618 (otwarty 2026-09-15); po merge usunąć łatkę przy rebase |
| 1 | i18n pl_PL | `Polski` w `language_options`, pełne `pl_PL.yaml` (upstream + 393 uzupełnienia), klucze odznak `badge.aoa.*` w en/pl | tłumaczenie do zgłoszenia przez Crowdin; klucze odznak są nasze (nie do upstreamu) |
