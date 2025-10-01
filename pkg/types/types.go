package types

type Validator struct {
	OperatorAddress string
	ConsAddress     string
	MissedBlocks    int64
	Moniker         string
	Jailed          bool
	Tombstoned      bool
	BondStatus      string
	BlocksProposed  int64
	Tokens          float64
	DelegatorShares float64
}
