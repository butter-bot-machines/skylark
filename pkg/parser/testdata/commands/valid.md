# Valid Commands

# the following is valid, the _possible_ assistant is named "command" (e.g. leading whitespace doesn't matter)
  !command without exclamation

# the following is valid, the _possible_ assistant is named "missing" (e.g. leading whitespace doesn't matter)
! missing command text

# the following command is valid, it will use the "default" assistant
!@invalid-chars command

# the following command is valid, the _possible_ assistant is named "assistant" (multiple references are allowed, if they exist; otherwise it will be passed through as prompt, with no "replacements")
!assistant with # invalid # reference format

# the following command is valid, but the assistant is defaulted to "default" with a warning message about the length
!very-long-assistant-name-that-exceeds-the-maximum-allowed-length some text
