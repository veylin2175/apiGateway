package voting

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	votingABIFile = "C:/Users/Matvey/Singularity/TrustVote/out/Voting.sol/Voting.json"
	stakeABIFile  = "C:/Users/Matvey/Singularity/TrustVote/out/TokenDropForStakers.sol/TokenDistributorForStakers.json"
)

type ContractABI struct {
	ABI json.RawMessage `json:"abi"`
}

func GetVotingABI() (abi.ABI, error) {
	absPath, err := filepath.Abs(votingABIFile)
	if err != nil {
		return abi.ABI{}, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return abi.ABI{}, err
	}

	var contractABI ContractABI
	if err := json.Unmarshal(data, &contractABI); err != nil {
		return abi.ABI{}, err
	}

	return abi.JSON(bytes.NewReader(contractABI.ABI))
}

func GetStakeABI() (abi.ABI, error) {
	absPath, err := filepath.Abs(stakeABIFile)
	if err != nil {
		return abi.ABI{}, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return abi.ABI{}, err
	}

	var contractABI ContractABI
	if err := json.Unmarshal(data, &contractABI); err != nil {
		return abi.ABI{}, err
	}

	return abi.JSON(bytes.NewReader(contractABI.ABI))
}
