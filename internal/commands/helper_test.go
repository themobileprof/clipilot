package commands

import (
	"testing"
)

func TestParseManSection(t *testing.T) {
	ch := &CommandHelper{}

	manText := `
NAME
       grep - print lines matching a pattern

SYNOPSIS
       grep [OPTIONS] PATTERN [FILE...]
       grep [OPTIONS] [-e PATTERN | -f FILE] [FILE...]

DESCRIPTION
       grep  searches  for  PATTERN  in  each  FILE.
       A FILE of "-" stands for standard input.

       In  addition,  the  variant programs egrep, fgrep and rgrep are the same as
       grep -E, grep -F, and grep -r, respectively.

OPTIONS:
       Generic Program Information
       --help Output a usage message and exit.

       -V, --version
              Output the version number of grep and exit.

EXAMPLES
       grep -i 'hello' file.txt
              Search for hello case-insensitively.

EXIT STATUS
       0      Selected lines are selected.
	`

	tests := []struct {
		name        string
		sectionName string
		want        string
	}{
		{
			// Expect preserved indentation
			name:        "Find SYNOPSIS",
			sectionName: "SYNOPSIS",
			want:        "       grep [OPTIONS] PATTERN [FILE...]\n       grep [OPTIONS] [-e PATTERN | -f FILE] [FILE...]",
		},
		{
			// Expect preserved indentation and stop at OPTIONS:
			name:        "Find DESCRIPTION (case insensitive)",
			sectionName: "Description",
			want:        "       grep  searches  for  PATTERN  in  each  FILE.\n       A FILE of \"-\" stands for standard input.\n\n       In  addition,  the  variant programs egrep, fgrep and rgrep are the same as\n       grep -E, grep -F, and grep -r, respectively.",
		},
		{
			// Expect finding OPTIONS: section (colon handling)
			name:        "Find OPTIONS (with colon)",
			sectionName: "OPTIONS",
			want:        "       Generic Program Information\n       --help Output a usage message and exit.\n\n       -V, --version\n              Output the version number of grep and exit.",
		},
		{
			name:        "Find EXAMPLES",
			sectionName: "EXAMPLES",
			want:        "       grep -i 'hello' file.txt\n              Search for hello case-insensitively.",
		},
		{
			name:        "Section Not Found",
			sectionName: "AUTHOR",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ch.parseManSection(manText, tt.sectionName)
			if got != tt.want {
				t.Errorf("parseManSection() = \n%q\nwant \n%q", got, tt.want)
			}
		})
	}
}
