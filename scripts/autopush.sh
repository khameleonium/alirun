#!/usr/bin/env bash
# Автоматический коммит и отправка (push) текущей ветки на GitHub.
#
# Использование:
#   scripts/autopush.sh                      # сообщение коммита сформируется автоматически
#   scripts/autopush.sh "fix: описание"      # своё сообщение коммита
#   scripts/autopush.sh -n "docs: правки"    # без проверки сборки и тестов
#
# Что делает:
#   1. Переходит в корень репозитория (откуда бы скрипт ни был запущен).
#   2. Проверяет, что среди изменений нет похожих на секреты файлов (.env, ключи)
#      и файлов больше 50 МБ (GitHub не принимает файлы больше 100 МБ).
#   3. Если менялись .go-файлы, go.mod или go.sum — запускает go vet и go test
#      (отключается флагом -n). При ошибке ничего не коммитит.
#   4. Добавляет все изменения (с учётом .gitignore) и создаёт коммит.
#   5. Если на GitHub появились новые коммиты — подтягивает их (git pull --rebase).
#   6. Отправляет коммиты в origin.
#
# Если изменений нет, но есть неотправленные коммиты — просто отправляет их.

set -euo pipefail

readonly MAX_FILE_MB=50

die() {
	echo "✖ $*" >&2
	exit 1
}

info() {
	echo "▶ $*"
}

usage() {
	sed -n '2,19p' "$0" | sed 's/^# \{0,1\}//'
}

# ---------- Аргументы ----------
run_checks=true
while [[ $# -gt 0 ]]; do
	case "$1" in
	-n | --no-checks)
		run_checks=false
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	--)
		shift
		break
		;;
	-*)
		die "Неизвестный флаг: $1 (см. --help)"
		;;
	*)
		break
		;;
	esac
done
message="$*"

# ---------- Репозиторий и ветка ----------
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git -C "$script_dir" rev-parse --show-toplevel 2>/dev/null)" ||
	die "Скрипт должен лежать внутри git-репозитория"
cd "$repo_root"

git_dir="$(git rev-parse --git-dir)"
if [[ -d "$git_dir/rebase-merge" || -d "$git_dir/rebase-apply" || -f "$git_dir/MERGE_HEAD" ]]; then
	die "Не завершены rebase или merge — сначала разберитесь с ними (git status)"
fi

branch="$(git symbolic-ref --quiet --short HEAD)" ||
	die "HEAD не указывает на ветку (detached HEAD) — переключитесь на ветку"

git remote get-url origin >/dev/null 2>&1 || die "Не настроен remote 'origin'"

# Go может быть не в PATH (как и в Makefile)
for go_bin in /usr/local/go/bin "$HOME/go/bin"; do
	[[ -d "$go_bin" ]] && PATH="$go_bin:$PATH"
done
export PATH

# ---------- Что будет закоммичено ----------
# Список файлов, которые добавит "git add -A" (dry-run ничего не меняет)
mapfile -t pending < <(git -c core.quotepath=off add -A --dry-run |
	sed -E "s/^(add|remove) '(.*)'$/\1 \2/")

if [[ ${#pending[@]} -gt 0 ]]; then
	info "Изменения (${#pending[@]}):"
	git -c core.quotepath=off status --short

	suspicious=()
	too_big=()
	go_changed=false
	for entry in "${pending[@]}"; do
		action="${entry%% *}"
		path="${entry#* }"

		case "$path" in
		*.go | go.mod | go.sum | */go.mod | */go.sum) go_changed=true ;;
		esac
		[[ "$action" == "add" ]] || continue # удаляемые файлы проверять не нужно

		base="${path##*/}"
		case "$base" in
		.env | .env.* | *.pem | *.key | *.p12 | *.pfx | id_rsa* | id_ed25519* | id_ecdsa* | *.kdbx)
			suspicious+=("$path")
			;;
		esac

		if [[ -f "$path" ]]; then
			size_mb=$(($(stat -c %s -- "$path") / 1024 / 1024))
			if ((size_mb >= MAX_FILE_MB)); then
				too_big+=("$path (${size_mb} МБ)")
			fi
		fi
	done

	if [[ ${#suspicious[@]} -gt 0 ]]; then
		printf '  %s\n' "${suspicious[@]}" >&2
		die "Эти файлы похожи на секреты. Добавьте их в .gitignore или закоммитьте вручную, если это осознанно."
	fi
	if [[ ${#too_big[@]} -gt 0 ]]; then
		printf '  %s\n' "${too_big[@]}" >&2
		die "Файлы больше ${MAX_FILE_MB} МБ. Добавьте их в .gitignore или используйте Git LFS."
	fi

	# ---------- Проверки ----------
	if $run_checks && $go_changed; then
		command -v go >/dev/null || die "Менялись Go-файлы, но go не найден (пропустить проверки: -n)"
		info "Проверка: go vet ./..."
		go vet ./... || die "go vet нашёл ошибки — коммит отменён (пропустить проверки: -n)"
		info "Проверка: go test ./..."
		go test ./... || die "Тесты не прошли — коммит отменён (пропустить проверки: -n)"
	fi

	# ---------- Коммит ----------
	git add -A
	if [[ -z "$message" ]]; then
		count=$(git diff --cached --name-only | wc -l)
		message="chore: автосохранение $(date '+%Y-%m-%d %H:%M') ($count файл(ов))"
		body="$(git -c core.quotepath=off diff --cached --name-status | head -n 30)"
		git commit --quiet -m "$message" -m "$body"
	else
		git commit --quiet -m "$message"
	fi
	info "Коммит: $(git log --oneline -1)"
else
	info "Нет новых изменений для коммита"
fi

# ---------- Синхронизация с GitHub ----------
upstream=""
if git rev-parse --abbrev-ref --symbolic-full-name '@{u}' >/dev/null 2>&1; then
	upstream="$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}')"
	info "Получаю изменения из $upstream..."
	git fetch --quiet origin || die "Не удалось связаться с origin (проверьте сеть и доступ)"

	behind=$(git rev-list --count "HEAD..$upstream")
	if ((behind > 0)); then
		info "На GitHub есть $behind новых коммит(ов) — переношу свои поверх них (rebase)..."
		if ! git pull --rebase --quiet; then
			git rebase --abort 2>/dev/null || true
			die "Конфликт при rebase. Ваш коммит сохранён локально; выполните 'git pull --rebase', разрешите конфликты и запустите скрипт снова."
		fi
	fi

	ahead=$(git rev-list --count "$upstream..HEAD")
	if ((ahead == 0)); then
		info "Всё уже отправлено — $branch совпадает с $upstream"
		exit 0
	fi
	info "Отправляю $ahead коммит(ов) в $upstream..."
	git push --quiet origin "$branch"
else
	info "У ветки $branch ещё нет пары на GitHub — создаю origin/$branch..."
	git push --quiet -u origin "$branch"
fi

echo "✔ Готово: $(git log --oneline -1) → origin/$branch"
