package solanaswapgo

import (
	"strconv"
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
		return swaps
	}

	userAccount := p.allAccountKeys[0]

	preBalances := make(map[string]uint64)
	postBalances := make(map[string]uint64)
	mintDecimals := make(map[string]uint8)

	for _, balance := range p.txMeta.PreTokenBalances {
		if int(balance.AccountIndex) >= len(p.allAccountKeys) || balance.Owner == nil {
			continue
		}

		if balance.Owner.String() == userAccount.String() {
			amt, err := strconv.ParseUint(balance.UiTokenAmount.Amount, 10, 64)
			if err == nil {
				key := balance.Mint.String()
				preBalances[key] = amt
				mintDecimals[key] = uint8(balance.UiTokenAmount.Decimals)
			}
		}
	}

	for _, balance := range p.txMeta.PostTokenBalances {
		if int(balance.AccountIndex) >= len(p.allAccountKeys) || balance.Owner == nil {
			continue
		}

		if balance.Owner.String() == userAccount.String() {
			amt, err := strconv.ParseUint(balance.UiTokenAmount.Amount, 10, 64)
			if err == nil {
				key := balance.Mint.String()
				postBalances[key] = amt
				mintDecimals[key] = uint8(balance.UiTokenAmount.Decimals)
			}
		}
	}

	if len(p.txMeta.PreBalances) > 0 && len(p.txMeta.PostBalances) > 0 {
		userPreBalance := p.txMeta.PreBalances[0]
		userPostBalance := p.txMeta.PostBalances[0]

		txFee := uint64(10000)

		if userPreBalance > userPostBalance+txFee {
			solAmount := userPreBalance - userPostBalance - txFee

			swaps = append(swaps, SwapData{
				Type: PUMP_SWAP,
				Data: &InputTransfer{
					TransferData: TransferData{
						Mint:     NATIVE_SOL_MINT_PROGRAM_ID.String(),
						Info:     TransferInfo{Amount: solAmount},
						Decimals: 9,
					},
				},
			})
		} else if userPostBalance > userPreBalance {
			solAmount := userPostBalance - userPreBalance

			swaps = append(swaps, SwapData{
				Type: PUMP_SWAP,
				Data: &OutputTransfer{
					TransferData: TransferData{
						Mint:     NATIVE_SOL_MINT_PROGRAM_ID.String(),
						Info:     TransferInfo{Amount: solAmount},
						Decimals: 9,
					},
				},
			})
		}
	}

	for mint, preBal := range preBalances {
		postBal, exists := postBalances[mint]

		if exists && preBal > postBal {
			amount := preBal - postBal
			decimals := mintDecimals[mint]

			swaps = append(swaps, SwapData{
				Type: PUMP_SWAP,
				Data: &InputTransfer{
					TransferData: TransferData{
						Mint:     mint,
						Info:     TransferInfo{Amount: amount},
						Decimals: decimals,
					},
				},
			})
		}
	}

	for mint, postBal := range postBalances {
		preBal, exists := preBalances[mint]

		if !exists {
			decimals := mintDecimals[mint]

			swaps = append(swaps, SwapData{
				Type: PUMP_SWAP,
				Data: &OutputTransfer{
					TransferData: TransferData{
						Mint:     mint,
						Info:     TransferInfo{Amount: postBal},
						Decimals: decimals,
					},
				},
			})
		} else if postBal > preBal {
			amount := postBal - preBal
			decimals := mintDecimals[mint]

			swaps = append(swaps, SwapData{
				Type: PUMP_SWAP,
				Data: &OutputTransfer{
					TransferData: TransferData{
						Mint:     mint,
						Info:     TransferInfo{Amount: amount},
						Decimals: decimals,
					},
				},
			})
		}
	}

	return swaps
}
