#!/usr/bin/env bash
# Runs harness scenarios against a -tags refdebug build and fails when one of
# them exits non-zero, panics, or reports a refdebug finding (a reference
# operation on a freed struct, or structs that outlived CEF).
#
#   go build -tags refdebug -o build/harness-refdebug.exe ./cmd/harness
#   cmd/harness/refdebug.sh build/harness-refdebug.exe [scenario ...]
#
# A scenario is the part of a GOMIUM_*_TEST variable between GOMIUM_ and
# _TEST, several joined with "+"; "OSR" is the plain default run. Needs the
# CEF runtime in ./Release, run from the repository root.
set -u

harness=${1:?usage: refdebug.sh <harness binary> [scenario ...]}
shift
if [ $# -eq 0 ]; then
	set -- OSR BINDING BINDING_POPUP BINDING+V8_LISTENER POPUP NAV COOKIE CONTEXT \
		DIALOG DOWNLOAD INTERCEPT SCHEME HANDLER DEVTOOLS
fi
seconds=${GOMIUM_RUN_SECONDS:-12}
logs=${REFDEBUG_LOGS:-build/refdebug}
mkdir -p "$logs" shots

status=0
for scenario in "$@"; do
	vars=(GOMIUM_RUN_SECONDS="$seconds")
	if [ "$scenario" != OSR ]; then
		IFS=+ read -ra parts <<<"$scenario"
		for part in "${parts[@]}"; do
			vars+=("GOMIUM_${part}_TEST=1")
		done
	fi
	log="$logs/${scenario//+/_}.log"
	env "${vars[@]}" timeout $((seconds + 60)) "$harness" >"$log" 2>&1
	code=$?
	findings=$(grep -a -c -E 'refdebug:|^panic:|fatal error:' "$log")
	if [ "$code" -ne 0 ] || [ "$findings" -ne 0 ]; then
		echo "FAIL $scenario (exit $code, $findings findings) - $log"
		grep -a -E -A40 'refdebug:|^panic:|fatal error:' "$log" | head -80
		status=1
	else
		echo "ok   $scenario"
	fi
done
exit $status
