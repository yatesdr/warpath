package nodestate

type NodeState struct {
	NodeID    int64
	NodeName  string
	Zone      string
	Enabled   bool
	Items     []BinItem
	ItemCount int
}

type BinItem struct {
	ID                int64  `json:"id"`
	PayloadCode       string `json:"payload_code"`
	Label             string `json:"label"`
	ManifestConfirmed bool   `json:"manifest_confirmed"`
	UOPRemaining      int    `json:"uop_remaining"`
	ClaimedBy         *int64 `json:"claimed_by,omitempty"`
}

