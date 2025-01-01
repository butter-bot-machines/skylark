# Default Assistant Fallback Tests

## Unknown Assistant Names
!nonexistent analyze this
Should fall back to default assistant

!missing-assistant help me
Should fall back to default assistant

!unknown_one process this
Should fall back to default assistant

## Mixed Case Unknown Names
!NonExistent analyze this
Should fall back to default assistant

!MISSING-ASSISTANT help me
Should fall back to default assistant

!Unknown_One process this
Should fall back to default assistant

## With Extra Space
!  nonexistent   analyze this
Should fall back to default assistant

!  MISSING-ASSISTANT   help me
Should fall back to default assistant

## Multiple Commands
!nonexistent first command
!missing-assistant second command
!unknown_one third command
Should all fall back to default assistant

## Edge Cases
!unknown
Should fall back to default assistant

!  unknown  
Should fall back to default assistant

!unknown-assistant
Should fall back to default assistant
