package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log/slog"

	"apiGateway/contracts/voting"
	"apiGateway/internal/config"
)

// Voter и VoteAccess - скопируйте их из вашего контракта Solidity или создайте их аналоги на Go
// Это упрощенные версии, возможно, вам потребуется более точное соответствие
// если ваш контракт Solidity использует более сложные структуры.
// В идеале, если вы используете abigen, эти структуры будут сгенерированы автоматически.
type Voter struct {
	Addr     common.Address `json:"addr"`
	HasVoted bool           `json:"hasVoted"`
	Choice   string         `json:"choice"`
	CanVote  uint8          `json:"canVote"` // 0 for NoAccess, 1 for HasAccess, etc.
}

// Enum для VoteAccess - соответствует Solidity enum
const (
	VoteAccessNoAccess uint8 = iota
	VoteAccessHasAccess
	VoteAccessOther // если есть другие
)

type VotingClient struct {
	client         *ethclient.Client
	contract       *bind.BoundContract
	cfg            *config.Config
	contractABI    abi.ABI
	contractAddr   common.Address
	privateKey     *ecdsa.PrivateKey // Приватный ключ для подписания транзакций
	publicKeyECDSA *ecdsa.PublicKey
	FromAddress    common.Address
	log            *slog.Logger // Добавляем логгер
}

func NewVotingClient(cfg *config.Config, log *slog.Logger) (*VotingClient, error) {
	if cfg == nil || cfg.Blockchain.RpcUrl == "" {
		return nil, fmt.Errorf("invalid configuration: RPC URL is required")
	}

	rpcURL := cfg.Blockchain.RpcUrl
	if !strings.HasPrefix(rpcURL, "http://") && !strings.HasPrefix(rpcURL, "https://") {
		rpcURL = "http://" + rpcURL
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node at %s: %v", rpcURL, err)
	}

	contractABI, err := voting.GetVotingABI()
	if err != nil {
		return nil, fmt.Errorf("failed to load contract ABI: %v", err)
	}

	if cfg.Blockchain.ContractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}
	if !common.IsHexAddress(cfg.Blockchain.ContractAddress) {
		return nil, fmt.Errorf("invalid contract address format: %s", cfg.Blockchain.ContractAddress)
	}
	contractAddr := common.HexToAddress(cfg.Blockchain.ContractAddress)

	// Инициализация приватного ключа
	privateKey, err := crypto.HexToECDSA(cfg.Blockchain.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	if fromAddress.Hex() != strings.ToLower(cfg.Blockchain.WalletAddress) && fromAddress.Hex() != cfg.Blockchain.WalletAddress {
		log.Warn("Configured WalletAddress does not match address derived from PrivateKey",
			slog.String("derived_address", fromAddress.Hex()),
			slog.String("config_address", cfg.Blockchain.WalletAddress))
		// Можно здесь вернуть ошибку, если это критично.
		// Сейчас просто предупреждение, так как транзакция будет отправлена с адреса из приватного ключа.
	}

	return &VotingClient{
		client:         client,
		contract:       bind.NewBoundContract(contractAddr, contractABI, client, client, client),
		cfg:            cfg,
		contractABI:    contractABI,
		contractAddr:   contractAddr,
		privateKey:     privateKey,
		publicKeyECDSA: publicKeyECDSA,
		FromAddress:    fromAddress,
		log:            log,
	}, nil
}

// GetVotingParticipatedByAddress - ваш существующий метод
func (vc *VotingClient) GetVotingParticipatedByAddress(address string) ([]*big.Int, error) {
	if vc == nil || vc.contract == nil {
		return nil, fmt.Errorf("voting client is not properly initialized")
	}

	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid address format: %s", address)
	}
	addr := common.HexToAddress(address)

	var rawResult []interface{}
	err := vc.contract.Call(
		&bind.CallOpts{
			Context: context.Background(),
			// Для view-функций from может быть пустым или любым валидным адресом
			// Если сеть требует From для view-функций, используйте vc.fromAddress
			// From:    vc.fromAddress,
		},
		&rawResult,
		"getVotingParticipatedByAddress",
		addr,
	)
	if err != nil {
		vc.log.Error("Contract call 'getVotingParticipatedByAddress' failed", slog.String("address", address), slog.Any("error", err))
		return nil, fmt.Errorf("contract call failed: %w", err) // Используем %w для оборачивания ошибки
	}

	if len(rawResult) == 0 {
		return []*big.Int{}, nil
	}

	result, ok := rawResult[0].([]*big.Int) // Ожидаем массив big.Int
	if !ok {
		return nil, fmt.Errorf("unexpected type in result for getVotingParticipatedByAddress: %T", rawResult[0])
	}

	return result, nil
}

/*// AddVoteSession отправляет транзакцию для создания новой сессии голосования
func (vc *VotingClient) AddVoteSession(
	title string,
	description string,
	startTime *big.Int, // Используем *big.Int для uint256
	endTime *big.Int,
	minNumberVotes *big.Int,
	isPrivate bool,
	voters []Voter, // Используем нашу Go-структуру Voter
	choices []string,
) (common.Hash, error) {
	if vc == nil || vc.contract == nil || vc.privateKey == nil {
		return common.Hash{}, fmt.Errorf("voting client is not properly initialized for sending transactions")
	}

	vc.log.Info("Preparing to send AddVoteSession transaction",
		slog.String("from_address", vc.FromAddress.Hex()),
		slog.String("contract_address", vc.contractAddr.Hex()))

	nonce, err := vc.client.PendingNonceAt(context.Background(), vc.FromAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := vc.client.SuggestGasPrice(context.Background())
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	// Создаем опции транзакции
	auth, err := bind.NewKeyedTransactorWithChainID(vc.privateKey, big.NewInt(1337)) // Используйте ChainID вашей сети (для Hardhat по умолчанию 1337)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // Отправка Ether с транзакцией (для этой функции 0)
	auth.GasLimit = uint64(5000000) // Установите достаточный лимит газа (может потребоваться корректировка)
	auth.GasPrice = gasPrice

	// Вызов функции контракта
	// Параметры должны точно соответствовать сигнатуре функции addVoteSession в Solidity.
	// `bind.BoundContract.Transact` ожидает `auth *bind.TransactOpts` в качестве первого аргумента.
	tx, err := vc.contract.Transact(auth, "addVoteSession",
		title,
		description,
		startTime,
		endTime,
		minNumberVotes,
		isPrivate,
		voters,
		choices,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send transaction to addVoteSession: %w", err)
	}

	vc.log.Info("Transaction sent for AddVoteSession", slog.String("tx_hash", tx.Hash().Hex()))

	return tx.Hash(), nil
}

// GetVotingCount - пример view функции, которую можно добавить, если она есть в контракте
func (vc *VotingClient) GetVotingCount() (*big.Int, error) {
	if vc == nil || vc.contract == nil {
		return nil, fmt.Errorf("voting client is not properly initialized")
	}

	var rawResult []interface{}
	err := vc.contract.Call(
		&bind.CallOpts{Context: context.Background()},
		&rawResult,
		"countVoteSessions", // Имя функции в контракте Solidity
	)
	if err != nil {
		return nil, fmt.Errorf("contract call 'countVoteSessions' failed: %w", err)
	}

	if len(rawResult) == 0 {
		return big.NewInt(0), nil
	}

	result, ok := rawResult[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected type in result for countVoteSessions: %T", rawResult[0])
	}

	return result, nil
}
*/
