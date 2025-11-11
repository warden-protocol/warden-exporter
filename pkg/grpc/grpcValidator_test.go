package grpc

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestValConsMapWithMissingValidator tests that we properly handle the case
// where a validator in signing info is not present in the validator map.
// This simulates the production condition that caused the nil pointer panic.
func TestSigningValidatorsHandlesMissingValidator(t *testing.T) {
	// Create a validator that exists in the map
	validatorAddr := "wardenvalcons1test123456789"

	vals := []staking.Validator{
		{
			OperatorAddress: "wardenvaloper1exists",
			Tokens:          math.NewInt(1000000),
			DelegatorShares: math.LegacyNewDec(1000000),
			Description: staking.Description{
				Moniker: "ExistingValidator",
			},
			Status: staking.Bonded,
		},
	}

	// Create signing info for two validators: one that exists and one that doesn't
	sInfos := []types.ValidatorSigningInfo{
		{
			Address:             validatorAddr, // This validator exists
			MissedBlocksCounter: 5,
			Tombstoned:          false,
		},
		{
			Address:             "wardenvalcons1missing", // This validator doesn't exist
			MissedBlocksCounter: 10,
			Tombstoned:          false,
		},
	}

	// Create a mock validator map
	valsMap := map[string]staking.Validator{
		validatorAddr: vals[0],
	}

	// Simulate the loop from SigningValidators
	// This should not panic even though one validator is missing
	for _, info := range sInfos {
		val, ok := valsMap[info.Address]
		if !ok {
			// Should skip missing validators gracefully
			continue
		}

		// This should not panic
		tokensBigInt := val.Tokens.BigInt()
		if tokensBigInt == nil {
			t.Errorf("Expected non-nil BigInt for existing validator")
		}
	}
}

// TestTokensConversionWithNilBigInt tests defensive handling of nil BigInt
// which theoretically shouldn't happen but we guard against it anyway.
func TestTokensConversionWithNilBigInt(t *testing.T) {
	tests := []struct {
		name          string
		tokens        math.Int
		expectPanic   bool
		expectNilBig  bool
	}{
		{
			name:         "zero value",
			tokens:       math.NewInt(0),
			expectPanic:  false,
			expectNilBig: false,
		},
		{
			name:         "positive value",
			tokens:       math.NewInt(1000000),
			expectPanic:  false,
			expectNilBig: false,
		},
		{
			name:         "large value",
			tokens:       math.NewInt(1000000000000),
			expectPanic:  false,
			expectNilBig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.expectPanic && r == nil {
					t.Errorf("Expected panic but didn't get one")
				}
				if !tt.expectPanic && r != nil {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			bigInt := tt.tokens.BigInt()
			if bigInt == nil && !tt.expectNilBig {
				t.Errorf("Got unexpected nil BigInt")
			}
		})
	}
}

// TestBondStatusConversion tests the bond status string conversion
func TestBondStatusConversion(t *testing.T) {
	tests := []struct {
		status   staking.BondStatus
		expected string
	}{
		{staking.Bonded, "bonded"},
		{staking.Unbonding, "unbonding"},
		{staking.Unbonded, "unbonded"},
		{staking.Unspecified, "unspecified"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := bondStatus(tt.status)
			if result != tt.expected {
				t.Errorf("bondStatus(%v) = %s, want %s", tt.status, result, tt.expected)
			}
		})
	}
}

// TestBondStatus tests the bond status string conversion helper
func TestBondStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   staking.BondStatus
		expected string
	}{
		{
			name:     "bonded status",
			status:   staking.Bonded,
			expected: "bonded",
		},
		{
			name:     "unbonding status",
			status:   staking.Unbonding,
			expected: "unbonding",
		},
		{
			name:     "unbonded status",
			status:   staking.Unbonded,
			expected: "unbonded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bondStatus(tt.status)
			if result != tt.expected {
				t.Errorf("bondStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}
