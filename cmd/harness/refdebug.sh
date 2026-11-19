#!/usr/bin/env bash
# Runs harness scenarios against a -tags refdebug build and fails when one of
# them exits non-zero, panics, or reports a refdebug finding (a reference
# operation on a freed struct, or structs that outlived CEF) - or never got a
# browser up, which would check nothing.
#
#   go build -tags refdebug -o build/harness-refdebug.exe ./cmd/harness
#   cmd/harness/refdebug.sh build/harness-refdebug.exe [scenario ...]
#
# CEF's own log (debug.log next to the harness, written by every process) is
# appended to each scenario log and scanned for fatal log lines and failed
# CHECKs/DCHECKs. With the debug runtime (cefdl -debug) that is where an
# over-release or a wrapper object alive at cef_shutdown shows up; see the
# cef-debug workflow.
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
cef_log=${CEF_DEBUG_LOG:-$(dirname "$harness")/debug.log}
mkdir -p "$logs" shots
findings_re='refdebug:|^panic:|fatal error:|:FATAL:|Check failed'

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
	rm -f "$cef_log"
	# Not timeout(1): after a failed DCHECK the debug libcef can wedge the
	# process in its exit path (a second DCHECK during DLL detach), where only
	# an elevated kill ends it and timeout would wait forever. Past the
	# deadline the scenario fails and the process is left behind.
	env "${vars[@]}" "$harness" >"$log" 2>&1 &
	pid=$!
	deadline=$((SECONDS + seconds + ${GOMIUM_SCENARIO_GRACE:-60}))
	while kill -0 "$pid" 2>/dev/null && [ "$SECONDS" -lt "$deadline" ]; do
		sleep 1
	done
	if kill -0 "$pid" 2>/dev/null; then
		kill -9 "$pid" 2>/dev/null
		disown "$pid" 2>/dev/null
		echo "--- refdebug.sh: still running after the deadline, abandoned ---" >>"$log"
		code=124
	else
		wait "$pid"
		code=$?
	fi
	if [ -f "$cef_log" ]; then
		echo "--- $cef_log ---" >>"$log"
		cat "$cef_log" >>"$log"
	fi
	findings=$(grep -a -c -E "$findings_re" "$log")
	# a run that never got a browser checks nothing
	browsers=$(grep -a -c 'browser received' "$log")
	if [ "$code" -ne 0 ] || [ "$findings" -ne 0 ] || [ "$browsers" -eq 0 ]; then
		echo "FAIL $scenario (exit $code, $findings findings, $browsers browsers) - $log"
		grep -a -E -A40 "$findings_re" "$log" | head -80
		status=1
	else
		echo "ok   $scenario"
	fi
done
exit $status
