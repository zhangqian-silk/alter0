#!/usr/bin/env sh
set -eu

trim_text() {
  printf '%s' "$1" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//'
}

json_escape() {
  printf '%s' "$1" | sed ':a;N;$!ba;s/\\/\\\\/g;s/"/\\"/g;s/\r//g;s/\n/\\n/g'
}

write_json_line() {
  printf '%s\n' "$1"
}

resolve_mock_thread_id() {
  prompt_hash="$(printf '%s' "$1" | cksum | awk '{print $1}')"
  printf 'mock-thread-%s' "$prompt_hash"
}

resolve_mock_reply() {
  trimmed="$(trim_text "$1")"
  lowered="$(printf '%s' "$trimmed" | tr '[:upper:]' '[:lower:]')"
  case "$lowered" in
    reply\ with\ exactly:*)
      suffix="${trimmed#*:}"
      trim_text "$suffix"
      return
      ;;
    respond\ with\ exactly:*)
      suffix="${trimmed#*:}"
      trim_text "$suffix"
      return
      ;;
    output\ *\ lines)
      count="$(printf '%s' "$lowered" | awk '{print $2}')"
      if [ -z "$count" ] || ! printf '%s' "$count" | grep -Eq '^[0-9]+$'; then
        count=1
      fi
      if [ "$count" -gt 200 ]; then
        count=200
      fi
      if [ "$count" -lt 1 ]; then
        count=1
      fi
      seq 1 "$count" | sed 's/^/line /'
      return
      ;;
  esac
  if [ -z "$trimmed" ]; then
    printf 'mock-empty'
    return
  fi
  printf 'mock: %s' "$trimmed"
}

if [ "$#" -lt 1 ] || [ "$1" != "exec" ]; then
  echo "unsupported mock invocation" >&2
  exit 2
fi

prompt=""
thread_id=""
is_resume="false"

shift
if [ "$#" -ge 1 ] && [ "$1" = "resume" ]; then
  is_resume="true"
  shift
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --json|--skip-git-repo-check|--color|never|--sandbox|danger-full-access)
        shift
        ;;
      -*)
        shift
        ;;
      *)
        thread_id="$1"
        shift
        break
        ;;
    esac
  done
  if [ -z "$thread_id" ] || [ "$#" -lt 1 ]; then
    echo "missing thread id or prompt" >&2
    exit 2
  fi
  prompt="$*"
else
  while [ "$#" -gt 1 ]; do
    shift
  done
  prompt="${1-}"
  thread_id="$(resolve_mock_thread_id "$prompt")"
fi
if [ "$prompt" = "-" ]; then
  prompt="$(cat)"
  if [ "$is_resume" != "true" ]; then
    thread_id="$(resolve_mock_thread_id "$prompt")"
  fi
fi

reply="$(resolve_mock_reply "$prompt")"
working_dir="$(pwd)"
command_output=$(printf 'WorkingDirectory\n---------------\n%s' "$working_dir")

write_json_line "{\"type\":\"thread.started\",\"thread_id\":\"$(json_escape "$thread_id")\"}"
write_json_line '{"type":"turn.started"}'
if [ "$is_resume" = "true" ]; then
  reasoning_text="Resume prior Codex CLI context."
else
  reasoning_text="Inspect workspace and prepare response."
fi
write_json_line "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_reasoning\",\"type\":\"reasoning\",\"text\":\"$(json_escape "$reasoning_text")\"}}"
write_json_line '{"type":"item.started","item":{"id":"item_command","type":"command_execution","command":"pwd"}}'
write_json_line "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_command\",\"type\":\"command_execution\",\"command\":\"pwd\",\"aggregated_output\":\"$(json_escape "$command_output")\",\"status\":\"completed\",\"exit_code\":0}}"
write_json_line "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_message\",\"type\":\"agent_message\",\"text\":\"$(json_escape "$reply")\"}}"
write_json_line '{"type":"turn.completed","usage":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":1}}'
