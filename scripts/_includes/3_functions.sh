# Function for outputting a "info" on the script
info () {
  echo -e "${BOLD}${@}${NORMAL}"
}

# Function for outputting a "info" on the script
notice () {
  echo -e "${BLINK}${BOLD}${YELLOW}!!!${NORMAL} ${@}${NORMAL}"
}

# Function for outputting a "step" on the script
step () {
  echo -e "${GREEN}***${DEFAULT}${BOLD} ${@}${NORMAL}"
}

# Function for outputting a "minor step" on the script
step_wait () {
  TEXT=$(printLeftAndRight "$(echo -e ${@})" "" true)
  echo -ne "${GREEN}***${DEFAULT}${BOLD} ${TEXT}${NORMAL}"
}

# Function for outputting a "minor step" on the script
step_continue () {
  echo -ne "${@}"
}

# Function for outputting a "minor step" on the script
minor_step () {
  echo -e "${BOLD}  *${NORMAL} ${@}"
}

# Function for outputting a "minor step" on the script
minor_step_wait () {
  TEXT=$(printLeftAndRight "$(echo -e ${@})" "" true)
  echo -ne "${BOLD}  *${NORMAL} ${TEXT}"
}

# Function for outputting a "minor step" on the script
minor_step_continue () {
  echo -e "${@}"
}

# Function to output a "divider"
output_divider() {
  TERMINAL_WIDTH="$(tput cols)"
  PADDING="$(printf '%0.1s' ={1..500})"
  DATE=$(date '+%Y-%m-%d %H:%M:%S')
  TEXT="${1} (${DATE})"
  printf '%*.*s %s %*.*s\n' 0 "$(((TERMINAL_WIDTH-2-${#TEXT})/2))" "$PADDING" "${TEXT}" 0 "$(((TERMINAL_WIDTH-1-${#TEXT})/2))" "$PADDING"
}

# Function for outputting a "all done" on the script
all_done () {
  echo -e "\n${GREEN}✔${NORMAL} ${BOLD}All done${NORMAL}"
}

# Function to check did the previous command succeed or not
continue_if_succeeded () {
  # Get the exit code of previous command
  RESULT=$?

  # If it did not exit with code 0 (succesful)
  if [ "$RESULT" != 0 ]; then
    hard_fail "Failed" $RESULT
  fi
}

# Function to check did the previous command succeed or not
exit_on_fail () {
  # Get the exit code of previous command
  RESULT=$?
  # Verbose
  VERBOSE=${1:-false}

  # If it did not exit with code 0 (succesful)
  if [ "$RESULT" != 0 ] && ! ${VERBOSE}; then
    echo -e "${BOLD}${RED}FAILED ${BLINK}!!!${NORMAL}"
    cat ${SCRIPT_EXECUTION_LOG}
    exit ${RESULT}
  elif [ "$RESULT" != 0 ] && ${VERBOSE}; then
    hard_fail "Failed" $RESULT
  elif ! ${VERBOSE}; then
    echo -e "${BOLD}${GREEN}OK${NORMAL}"
  fi
}

# Output an error
step_failed () {
  # First argument is Message, by default it is "Failed"
  MESSAGE=${1:-"Failed"}
  # Second argument is the exit code, by default it is 1
  STATUS=${2:-1}

  # Define the error message prefix/suffix
  ERROR="${BLINK}${BOLD}${RED}!!!${NORMAL}"

  # Echo the error message with prefix & suffix and exit with given status code
  echo -e "${ERROR} ${MESSAGE} ${ERROR}"
}

# Output an error
hard_fail () {
  # First argument is Message, by default it is "Failed"
  MESSAGE=${1:-"Failed"}
  # Second argument is the exit code, by default it is 1
  STATUS=${2:-1}

  step_failed "${MESSAGE}" "${STATUS}"
  exit $STATUS
}

# Echo with colors
echo_color () {
  echo -e "${@}"
}

# Echo STEP title, run commands and exit on fail
run_step_executor () {
  # STEP
  STEP=${1:-"step"}
  # Verbosity
  VERBOSE=${2:-true}
  # Get the title from the arguments
  TITLE=${3}
  # Get the commands
  COMMANDS=${@:4}

  # Echo step title
  ${STEP} "${TITLE}"
  # Run the commands
  if ${VERBOSE}; then
    ${COMMANDS}
  else
    ${COMMANDS} &> ${SCRIPT_EXECUTION_LOG}
  fi
  # Continue if the command(s) succeeded or exit if it failed
  exit_on_fail ${VERBOSE}
}

# Echo STEP title, run commands and exit on fail
run_step () {
  # Get the title from the arguments
  TITLE=${1}
  # Get the commands
  COMMANDS=${@:2}

  run_step_executor "step" true "${TITLE}" "${COMMANDS}"
}

# Echo STEP title, run commands and exit on fail
run_minor_step () {
  # Get the title from the arguments
  TITLE=${1}
  # Get the commands
  COMMANDS=${@:2}

  run_step_executor "minor_step" true "${TITLE}" "${COMMANDS}"
}

# Echo STEP title, run commands and exit on fail
silent_step () {
  # Get the title from the arguments
  TITLE=${1}
  # Get the commands
  COMMANDS=${@:2}

  run_step_executor "step_wait" false "${TITLE}" "${COMMANDS}"
}

# Echo STEP title, run commands and exit on fail
silent_minor_step () {
  # Get the title from the arguments
  TITLE=${1}
  # Get the commands
  COMMANDS=${@:2}

  run_step_executor "minor_step_wait" false "${TITLE}" "${COMMANDS}"
}

printLeftAndRight () {
  checkTerminalWidth
  LEFT=${1}
  RIGHT=${2}
  NEW_LINE=${3:-true}
  PREFIX="${CLEAR_LINE}"

  EXTRA_SPACERS=14

  PADDING_MAX_LENGTH=${PADDED_LENGTH}

  if [ "${PADDED_LENGTH}" -gt "${TERMINAL_WIDTH}" ]; then
    PADDING_MAX_LENGTH=${TERMINAL_WIDTH}
  fi

  STRIPPED=$(echo ${LEFT} | perl -pe 's/\x1b\[[^m]+m//g;')

  if [ "${#STRIPPED}" -gt $((PADDING_MAX_LENGTH - ${EXTRA_SPACERS})) ]; then
    LEFT="${LEFT:0:$((PADDING_MAX_LENGTH - ${EXTRA_SPACERS} - 3))}..."
  fi

  PADDING_MAX_LENGTH=$((PADDING_MAX_LENGTH - ${EXTRA_SPACERS}))

  LINE=$(printf "%s%*s%s" "${LEFT}" $(expr ${PADDING_MAX_LENGTH} - ${#LEFT}) "${RIGHT}")

  if ${NEW_LINE}; then
    PREFIX=""
  fi

  echo -ne "${PREFIX}${LINE}"
}

pidExists () {
  if ! kill -0 $1 2>/dev/null; then
    return 1
  else
    return 0
  fi
}

checkTerminalWidth () {
  TERMINAL_WIDTH=$(tput cols)
}

hideCursor () {
  tput civis
}

showCursor () {
  tput cnorm
}
