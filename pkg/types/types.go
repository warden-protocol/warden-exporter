package types

type Validator struct {
	OperatorAddress string
	ConsAddress     string
	MissedBlocks    int64
	Moniker         string
	Jailed          bool
	Tombstoned      bool
	BondStatus      string
}
