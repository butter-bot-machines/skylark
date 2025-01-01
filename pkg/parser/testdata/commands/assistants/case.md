# Assistant Name Case Tests

## Lowercase Names
!researcher analyze this
!assistant help me
!summarizer process this

## Uppercase Names
!RESEARCHER analyze this
!ASSISTANT help me
!SUMMARIZER process this

## Mixed Case Names
!Researcher analyze this
!AsSiStAnT help me
!SumMarIzer process this

## Special Cases
!DEFAULT analyze this
!Default analyze this
!default analyze this

## With Extra Space
!  researcher   analyze this
!  ASSISTANT   help me
!  Summarizer   process this

## Multiple Commands
!researcher first command
!RESEARCHER second command
!Researcher third command

## Mixed in Text
This is a paragraph with a !researcher command
that should not be matched because ! must be
at start of line (ignoring whitespace).

## Edge Cases
!
! 
!  
!researcher
!RESEARCHER
!Researcher
