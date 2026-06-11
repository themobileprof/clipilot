package catalog

import "testing"

func TestSearchDiskSpace(t *testing.T) {
	hits := Search("how much storage is left on my phone")
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	if hits[0].Entry.Name != "df" && hits[0].Entry.Name != "du" {
		t.Fatalf("got %q, want df or du", hits[0].Entry.Name)
	}
}

func TestSearchPidgin(t *testing.T) {
	hits := Search("data no dey work abeg")
	if len(hits) == 0 {
		t.Fatal("expected hits for network query")
	}
	top := hits[0].Entry.Name
	if top != "ping" && top != "curl" {
		t.Fatalf("got %q, expected ping or curl", top)
	}
}

func TestSearchCopyFile(t *testing.T) {
	hits := Search("how do I duplicate a file")
	if len(hits) == 0 || hits[0].Entry.Name != "cp" {
		t.Fatalf("got %+v", hits)
	}
}

func TestTokenizePidgin(t *testing.T) {
	tokens := tokenize("wetin dey inside folder abeg")
	if len(tokens) < 2 {
		t.Fatalf("expected tokens, got %v", tokens)
	}
}
