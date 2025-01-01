# Assistant Name Normalization Tests

## Simple Lowercase
!MyAssistant help
Should normalize to "myassistant"

!ASSISTANT analyze
Should normalize to "assistant"

!TestAssistant process
Should normalize to "testassistant"

## Preserve Format
!my-assistant help
Should preserve as "my-assistant"

!test_assistant analyze
Should preserve as "test_assistant"

!my.assistant process
Should preserve as "my.assistant"

## Mixed Case with Format
!My-Assistant help
Should normalize to "my-assistant"

!TEST_ASSISTANT analyze
Should normalize to "test_assistant"

!My.Assistant process
Should normalize to "my.assistant"

## Numbers and Symbols
!Assistant2 help
Should normalize to "assistant2"

!TEST-2 analyze
Should normalize to "test-2"

!My_3_Assistant process
Should normalize to "my_3_assistant"

## Edge Cases
!A help
Should normalize to "a"

!_ analyze
Should preserve as "_"

!-test- process
Should preserve as "-test-"
