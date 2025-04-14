package solanaswapgo

import (
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

type InputTransfer struct {
	TransferData
}

type OutputTransfer struct {
	TransferData
}

func (p *Parser) processPumpSwapSwaps(instructionIndex int) []SwapData {
	var swaps []SwapData

	innerInstructions := p.getInnerInstructions(instructionIndex)
	if len(innerInstructions) == 0 {
		p.Log.Warnf("No inner instructions found for PumpSwap swap at index %d", instructionIndex)
		return swaps
	}

	// Assume user is the fee payer
	userAccount := p.allAccountKeys[0]

	// Map token account keys to their owners
	tokenAccountOwners := make(map[solana.PublicKey]string)
	for _, balance := range p.txMeta.PreTokenBalances {
		accountKey := p.allAccountKeys[balance.AccountIndex]
		tokenAccountOwners[accountKey] = balance.Owner.String()
	}

	for _, inner := range innerInstructions {
		progID := p.allAccountKeys[inner.ProgramIDIndex]
		if progID.Equals(solana.TokenProgramID) && len(inner.Data) >= 12 && inner.Data[0] == 3 { // TransferChecked
			// Accounts: [source, mint, destination, owner, ...]
			sourceKey := p.allAccountKeys[inner.Accounts[0]]
			destinationKey := p.allAccountKeys[inner.Accounts[2]]
			mintKey := p.allAccountKeys[inner.Accounts[1]].String()
			amount := binary.LittleEndian.Uint64(inner.Data[4:12])

			decimals, ok := p.splDecimalsMap[mintKey]
			if !ok {
				p.Log.Warnf("Decimals not found for mint %s", mintKey)
				continue
			}

			sourceOwner, sourceOk := tokenAccountOwners[sourceKey]
			destOwner, destOk := tokenAccountOwners[destinationKey]
			if !sourceOk || !destOk {
				p.Log.Debugf("Owner info missing for transfer accounts %s or %s", sourceKey, destinationKey)
				continue
			}

			if sourceOwner == userAccount.String() && destOwner != userAccount.String() {
				// Input: user -> DEX
				swaps = append(swaps, SwapData{
					Type: PUMP_SWAP,
					Data: &InputTransfer{
						TransferData: TransferData{
							Mint:     mintKey,
							Info:     TransferInfo{Amount: amount},
							Decimals: decimals,
						},
					},
				})
			} else if sourceOwner != userAccount.String() && destOwner == userAccount.String() {
				// Output: DEX -> user
				swaps = append(swaps, SwapData{
					Type: PUMP_SWAP,
					Data: &OutputTransfer{
						TransferData: TransferData{
							Mint:     mintKey,
							Info:     TransferInfo{Amount: amount},
							Decimals: decimals,
						},
					},
				})
			}
		}
	}

	if len(swaps) == 0 {
		p.Log.Warnf("No valid PumpSwap swap transfers extracted at index %d", instructionIndex)
	}
	return swaps
}