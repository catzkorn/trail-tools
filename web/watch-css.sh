#!/bin/sh
# Rebuilds dist/index.css whenever the web sources change, for make watch.
#
# Tailwind's own --watch doesn't work here: file change events don't reach
# the container through its mount, and it exits once stdin is closed, as it
# is for a background job. Instead, this polls the sources' modification
# times every second.
set -u

stamp=/tmp/watch-css.stamp

build() {
	touch "$stamp"
	npx -s @tailwindcss/cli -i base.css -o dist/index.css
}

# changed prints a source file modified since the last build, if any.
changed() {
	find . \( -path ./node_modules -o -path ./dist -o -path ./gen \) -prune -o \
		\( -name '*.ts' -o -name '*.tsx' -o -name '*.css' \) \
		-newer "$stamp" -print | head -n 1
}

build
while sleep 1; do
	if [ -n "$(changed)" ]; then
		build
	fi
done
