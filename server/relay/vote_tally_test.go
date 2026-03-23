package main

import (
	"testing"
)

// checks the 1 vote per user rule
func TestVoteTally_RecordAndWinner(t *testing.T) {
	vt := NewVoteTally()

	c1 := &Client{}
	c2 := &Client{}
	c3 := &Client{}

	vt.Record(c1, "!jump")
	vt.Record(c2, "!jump")
	vt.Record(c3, "!w")

	if !vt.HasVotes() {
		t.Fatal("Expected HasVotes to be true after recording votes")
	}

	winner, votes := vt.Winner()
	if winner != "!jump" || votes != 2 {
		t.Fatalf("Expected '!jump' with 2 votes, got '%s' with %d votes", winner, votes)
	}

	vt.Record(c1, "!w") // c1 tries to change their vote or vote again

	winner, votes = vt.Winner()
	if winner != "!jump" || votes != 2 {
		t.Fatalf("Expected one-vote-per-client to prevent vote change. Got '%s' with %d votes", winner, votes)
	}

	vt.Reset()
	if vt.HasVotes() {
		t.Fatal("Expected HasVotes to be false after Reset()")
	}

	winner, votes = vt.Winner()
	if winner != "" || votes != 0 {
		t.Fatalf("Expected empty winner and 0 votes after reset, got '%s' with %d votes", winner, votes)
	}
}
