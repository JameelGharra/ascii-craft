package main

type VoteTally struct {
	votes map[*Client]string
}

func NewVoteTally() *VoteTally {
	return &VoteTally{
		votes: make(map[*Client]string),
	}
}

// registers user's vote, if already voted then will be ignored (1 user = 1 vote)
func (vt *VoteTally) Record(client *Client, cmd string) {
	if _, alreadyVoted := vt.votes[client]; !alreadyVoted {
		vt.votes[client] = cmd
	}
}

// counts collected votes
func (vt *VoteTally) Winner() (command string, maxVotes int) {
	tally := make(map[string]int)
	for _, cmd := range vt.votes {
		tally[cmd]++
	}

	for cmd, count := range tally {
		if count > maxVotes {
			maxVotes = count
			command = cmd
		}
	}
	return command, maxVotes
}

func (vt *VoteTally) Reset() {
	vt.votes = make(map[*Client]string)
}

// returns true if at least one vote was cast this round
func (vt *VoteTally) HasVotes() bool {
	return len(vt.votes) > 0
}
