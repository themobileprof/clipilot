package catalog

import "strings"

var stopwords = map[string]bool{
	"i": true, "want": true, "to": true, "how": true, "do": true, "can": true,
	"you": true, "please": true, "the": true, "a": true, "an": true, "is": true,
	"of": true, "in": true, "on": true, "for": true, "and": true, "or": true,
	"with": true, "help": true, "me": true, "my": true, "this": true, "that": true,
	"abeg": true, "sha": true, "sef": true, "abi": true, "o": true, "na": true,
	"wey": true, "make": true, "e": true, "im": true, "em": true,
	"take": true, "use": true, "using": true, "command": true, "terminal": true,
}

// pidgin maps Nigerian Pidgin / campus slang to search terms.
var pidgin = map[string]string{
	"wetin": "what", "comot": "delete", "don": "full", "finish": "full",
	"jam": "stuck", "jammed": "stuck", "hang": "stuck", "gree": "work",
	"data": "network", "sub": "network", "slow": "memory", "lagging": "slow",
	"lag": "slow", "storage": "disk", "pics": "file", "photos": "file",
	"assignment": "file", "lecture": "pdf", "repo": "git", "coding": "code",
	"wan": "want", "inside": "here", "phone": "device",
}

func tokenize(input string) []string {
	raw := strings.Fields(strings.ToLower(input))
	seen := make(map[string]bool)
	var out []string

	add := func(t string) {
		if t == "" || len(t) < 2 || stopwords[t] || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, t)
	}

	for _, t := range raw {
		t = strings.Trim(t, "?!.,;:\"'()")
		add(t)
		if mapped, ok := pidgin[t]; ok {
			add(mapped)
		}
	}
	return out
}
