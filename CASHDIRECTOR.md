# Fork Apache Answer dla portalu Pomoc AO AOA (CashDirector)

Gałęzie:
- `main` — czysty upstream (`git fetch upstream && git merge --ff-only upstream/main`). Nigdy nie commitować bezpośrednio.
- `cashdirector` — `main` + nasze łatki. Z niej budowany jest obraz produkcyjny (repo `answers`, `Dockerfile`).

Konwencje łatek:
- jedna funkcjonalność = jeden commit, tytuł z prefiksem `[cd]`; bez poprawek „przy okazji”, bez zmian formatowania cudzych plików;
- przy nowym wydaniu upstreamu: `git rebase v2.x.y` gałęzi `cashdirector`, konflikty tylko w naszych plikach;
- `git format-patch upstream/main..cashdirector` = lista tego, czym się różnimy (miara długu — ma być krótka);
- co się da, oddajemy upstreamowi jako PR; po merge łatka znika stąd.

Tagi produkcyjne: `cd-v<wersja upstreamu>-<numer>` (np. `cd-v2.0.2-1`). Obraz w ECR: `answer:<wersja>-cd<numer>`.

Pełne wytyczne: `answers/docs/fork-answer-wytyczne.md`. Zadania: Jira AA-26.

## Łatki na tej gałęzi
| # | Commit | Co | Status upstream |
|---|---|---|---|
| — | — | (jeszcze brak) | — |
