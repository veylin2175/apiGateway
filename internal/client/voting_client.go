package client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
	"time"

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

type VoteSessionCreatedEvent struct {
	VoteSessionId *big.Int
	Name          string
	StartTime     *big.Int
	EndTime       *big.Int
}

// Enum для VoteAccess - соответствует Solidity enum
const (
	VoteAccessNoAccess uint8 = iota
	VoteAccessHasAccess
	VoteAccessOther // если есть другие
)

type VotingClient struct {
	Client         *ethclient.Client
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

	if cfg.Blockchain.VotingContractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}
	if !common.IsHexAddress(cfg.Blockchain.VotingContractAddress) {
		return nil, fmt.Errorf("invalid contract address format: %s", cfg.Blockchain.VotingContractAddress)
	}
	contractAddr := common.HexToAddress(cfg.Blockchain.VotingContractAddress)

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
	}

	return &VotingClient{
		Client:         client,
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

// GetVotingParticipatedByAddress возвращает массив uint - массив голосования, в которых юзер участвовал
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

// AddVoteSession отправляет транзакцию для создания новой сессии голосования
// AddVoteSession вызывает функцию addVoteSession из контракта
// Теперь возвращает ID голосования, адрес создателя и хеш транзакции.
func (vc *VotingClient) AddVoteSession(
	title string,
	description string,
	startTime *big.Int,
	endTime *big.Int,
	minNumberVotes *big.Int,
	isPrivate bool,
	voters []Voter,
	choices []string,
) (*big.Int, common.Address, common.Hash, error) { // Возвращаемые значения
	nonce, err := vc.Client.PendingNonceAt(context.Background(), vc.FromAddress)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := vc.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, common.Address{}, common.Hash{}, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(vc.privateKey, big.NewInt(31337)) // ChainID для Anvil
	if err != nil {
		return nil, common.Address{}, common.Hash{}, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(5000000)
	auth.GasPrice = gasPrice

	vc.log.Info("Preparing to send AddVoteSession transaction",
		slog.String("from_address", vc.FromAddress.Hex()),
		slog.String("title", title))

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
		return nil, common.Address{}, common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	vc.log.Info("Waiting for AddVoteSession transaction to be mined...", slog.String("tx_hash", tx.Hash().Hex()))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	receipt, err := bind.WaitMined(ctx, vc.Client, tx)
	if err != nil {
		return nil, common.Address{}, tx.Hash(), fmt.Errorf("failed to mine transaction %s: %w", tx.Hash().Hex(), err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, common.Address{}, tx.Hash(), fmt.Errorf("transaction %s reverted with status %d", tx.Hash().Hex(), receipt.Status)
	}

	// Парсим событие VoteSessionCreated из vLog.Data
	var voteSessionID *big.Int
	creatorAddress := vc.FromAddress // Адрес создателя - это отправитель транзакции

	// Находим описание события 'VoteSessionCreated' в ABI
	event, found := vc.contractABI.Events["VoteSessionCreated"]
	if !found {
		return nil, common.Address{}, tx.Hash(), fmt.Errorf("event VoteSessionCreated not found in contract ABI")
	}

	for _, vLog := range receipt.Logs {
		// Проверяем, что лог исходит от нашего контракта и соответствует сигнатуре события
		if vLog.Address == vc.contractAddr && len(vLog.Topics) > 0 && vLog.Topics[0] == event.ID {
			// Лог соответствует событию VoteSessionCreated
			var unpackedEvent VoteSessionCreatedEvent
			err := vc.contractABI.UnpackIntoInterface(&unpackedEvent, event.Name, vLog.Data)
			if err != nil {
				return nil, common.Address{}, tx.Hash(), fmt.Errorf("failed to unpack log data for VoteSessionCreated: %w", err)
			}

			voteSessionID = unpackedEvent.VoteSessionId

			vc.log.Info("Parsed VoteSessionCreated event from data",
				slog.String("voting_id", voteSessionID.String()),
				slog.String("name", unpackedEvent.Name),
				slog.String("start_time", unpackedEvent.StartTime.String()),
				slog.String("end_time", unpackedEvent.EndTime.String()),
				slog.String("creator", creatorAddress.Hex()), // Используем FromAddress как создателя
				slog.String("tx_hash", tx.Hash().Hex()))

			return voteSessionID, creatorAddress, tx.Hash(), nil
		}
	}

	return nil, common.Address{}, tx.Hash(), fmt.Errorf("VoteSessionCreated event not found in transaction receipt for tx_hash: %s", tx.Hash().Hex())
}

// GetVotingCreatedByAddress
// Эта функция вызывает метод контракта `getVotingCreatedByAddress`
// и возвращает массив ID голосований, созданных данным адресом.
func (vc *VotingClient) GetVotingCreatedByAddress(address string) ([]*big.Int, error) {
	if vc == nil || vc.contract == nil {
		return nil, fmt.Errorf("voting client is not properly initialized")
	}

	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid address format: %s", address)
	}
	addr := common.HexToAddress(address)

	var rawResult []interface{}
	// Вызываем метод контракта "getVotingCreatedByAddress"
	err := vc.contract.Call(
		&bind.CallOpts{
			Context: context.Background(),
		},
		&rawResult,
		"getVotingCreatedByAddress", // Имя функции в контракте, точно как в Solidity
		addr,
	)
	if err != nil {
		vc.log.Error("Contract call 'getVotingCreatedByAddress' failed", slog.String("address", address), slog.Any("error", err))
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	if len(rawResult) == 0 {
		return []*big.Int{}, nil // Возвращаем пустой срез, если нет результатов
	}

	// Проверяем тип возвращаемого значения
	result, ok := rawResult[0].([]*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected type in result for getVotingCreatedByAddress: %T", rawResult[0])
	}

	return result, nil
}

// Vote вызывает функцию vote из контракта Voting.sol
func (vc *VotingClient) Vote(voteSessionID *big.Int, indChoice *big.Int) (common.Hash, error) {
	nonce, err := vc.Client.PendingNonceAt(context.Background(), vc.FromAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := vc.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(vc.privateKey, big.NewInt(31337)) // ChainID для Anvil
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // Голосование не переводит ETH
	auth.GasLimit = uint64(5000000) // Обычный лимит для голосования, можно подкорректировать
	auth.GasPrice = gasPrice

	vc.log.Info("Preparing to send vote transaction",
		slog.Any("vote_session_id", voteSessionID),
		slog.Any("choice_index", indChoice),
		slog.String("from_address", vc.FromAddress.Hex()))

	tx, err := vc.contract.Transact(auth, "vote", voteSessionID, indChoice)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send vote transaction: %w", err)
	}

	vc.log.Info("Vote transaction sent", slog.String("tx_hash", tx.Hash().Hex()))

	return tx.Hash(), nil
}
