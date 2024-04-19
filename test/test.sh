#!/bin/bash

set -e
cd "$(dirname "${0}")"

# Set traps to kill our subprocesses when we exit (for any reason).
trap ":" TERM      # Avoid the EXIT handler from killing bash.
trap "exit 2" INT  # Ctrl-C, make sure we fail in that case.
trap "kill 0" EXIT # Kill children on exit.


if [ "$V" == "1" ]; then
	set -v
fi


echo "# Build"

go build httpd.go

( cd ..; go build $GOFLAGS )


echo "# Test"

function wait_until_ready() {
	PORT=$1

	while ! bash -c "true < /dev/tcp/localhost/$PORT" 2>/dev/null ; do
		sleep 0.01
	done
}

function test_one() {
	echo "## $1"

	# Launch httpd.
	./httpd &
	HTTPD_PID=$?
	wait_until_ready 30081

	# Launch prometheus-expvar-exporter
	../prometheus-expvar-exporter "-config=$1.toml" > .pee.log 2>&1 &
	PEE_PID=$?
	wait_until_ready 30080

	# Get the exported metrics.
	curl -s localhost:30080/metrics > "$1.got"
	kill $PEE_PID $HTTPD_PID

	# Compare the results.
	if ! ./diff.sh "$1.expected" "$1.got"; then
		cat "$1.got.diff"
		echo
		echo
		echo failed
		exit 1
	fi
}

for i in t-*.toml; do
	test_one "$(basename "$i" .toml)"
done
