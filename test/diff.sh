#!/bin/bash

# Diff the metrics page, ignoring stuff that Prometheus libraries adds and
# we don't care about.

cat "$2" \
		| grep -v -E '(^| )go_gc_' \
		| grep -v -E '(^| )go_goroutines' \
		| grep -v -E '(^| )go_info' \
		| grep -v -E '(^| )go_memstats' \
		| grep -v -E '(^| )go_threads' \
		| grep -v -E '(^| )process_' \
		| grep -v -E '(^| )promhttp_' \
		> "$2.filtered"

exec diff -u "$1" "$2.filtered" > "$2.diff"
