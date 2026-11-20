#!/usr/bin/env bash
# Runs harness scenarios against a -tags refdebug build and fails when one of
# them exits non-zero, panics, or reports a refdebug finding (a reference
# operation on a freed struct, or structs that outlived CEF) - or never got a
# browser up, never finished loading a page, or is missing the log lines that
# show its feature worked. A run without them checks nothing (#41).
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
	set -- OSR FRAME BINDING BINDING_POPUP BINDING+V8_LISTENER POPUP NAV COOKIE CONTEXT \
		DIALOG DOWNLOAD INTERCEPT SCHEME HANDLER DEVTOOLS
fi
seconds=${GOMIUM_RUN_SECONDS:-12}
logs=${REFDEBUG_LOGS:-build/refdebug}
cef_log=${CEF_DEBUG_LOG:-$(dirname "$harness")/debug.log}
mkdir -p "$logs" shots
findings_re='refdebug:|^panic:|fatal error:|:FATAL:|Check failed|WARNING: DATA RACE'

# expected_markers prints, one per line, the fixed strings a scenario part
# must have logged: results, not actions - "the page answered", not "we
# clicked". Every part also needs a finished page load (checked below).
expected_markers() {
	case "$1" in
	FRAME) echo 'frame painted' ;;
	BINDING)
		echo 'BINDING_MAIN_RESULT pong'
		echo 'BINDING_ADD 5'
		echo 'BINDING_FAIL Error: boom'
		echo 'BINDING_FRAME_ERR'
		;;
	BINDING_POPUP) echo 'BINDING_POPUP_RESULT pong' ;;
	V8_LISTENER) echo 'V8_LISTENER_INJECTED main=true' ;;
	POPUP | NAV) echo 'url=https://example.com/' ;;
	COOKIE) echo 'cookie-test: delete ok' ;;
	CONTEXT) echo 'context-test: browser in context A created' ;;
	DIALOG)
		echo 'dialog-test: page-side answers'
		echo 'menu-test: context menu intercepted'
		;;
	DOWNLOAD) echo 'complete=true' ;;
	SCHEME) echo 'heading color is rgb(0, 128, 0) (want rgb(0, 128, 0))' ;;
	HANDLER) echo 'handler-test: main frame changed' ;;
	DEVTOOLS) echo 'devtools: evaluate' ;;
	esac
}

status=0
for scenario in "$@"; do
	vars=(GOMIUM_RUN_SECONDS="$seconds")
	parts=()
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
	# a run that never got a browser, or never finished a page, checks nothing
	browsers=$(grep -a -c 'browser received' "$log")
	loaded=$(grep -a -c 'loading=false' "$log")
	missing=()
	for part in ${parts[@]+"${parts[@]}"}; do
		while IFS= read -r marker; do
			[ -n "$marker" ] || continue
			grep -a -F -q -- "$marker" "$log" || missing+=("$marker")
		done <<<"$(expected_markers "$part")"
	done
	if [ "$code" -ne 0 ] || [ "$findings" -ne 0 ] || [ "$browsers" -eq 0 ] ||
		[ "$loaded" -eq 0 ] || [ ${#missing[@]} -ne 0 ]; then
		echo "FAIL $scenario (exit $code, $findings findings, $browsers browsers, $loaded pages loaded) - $log"
		for marker in ${missing[@]+"${missing[@]}"}; do
			echo "     missing: $marker"
		done
		grep -a -E -A40 "$findings_re" "$log" | head -80
		status=1
	else
		echo "ok   $scenario"
	fi
done
exit $status
