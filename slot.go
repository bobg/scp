package scp

type SlotID int

type Slot struct {
	ID SlotID
	Ph Phase
	V  *Node
}

type Phase int

const (
	PhNom Phase = iota
	PhPrep
	PhCommit
	PhExt
)
