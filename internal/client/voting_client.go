package client

import (
	"context"
	"crypto/ecdsa"
	"errors"
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

type StakeClient struct {
	Client         *ethclient.Client
	contract       *bind.BoundContract
	StakeManager   *voting.ContractABI
	cfg            *config.Config
	contractABI    abi.ABI
	contractAddr   common.Address
	privateKey     *ecdsa.PrivateKey // Приватный ключ для подписания транзакций
	publicKeyECDSA *ecdsa.PublicKey
	publicKey      common.Address
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

func NewStakeClient(cfg *config.Config, log *slog.Logger) (*StakeClient, error) {
	log.Info("Attempting to create new stake client...")

	if cfg == nil {
		log.Error("Config is nil during StakeClient creation.")
		return nil, fmt.Errorf("config is nil")
	}
	if cfg.Blockchain.RpcUrl == "" {
		log.Error("RPC URL is empty in config.")
		return nil, fmt.Errorf("RPC URL is empty")
	}
	rpcURL := cfg.Blockchain.RpcUrl
	log.Info("Connecting to RPC", "url", rpcURL)
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Error("Failed to connect to Ethereum client", "url", rpcURL, "error", err)
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}
	log.Info("Successfully connected to Ethereum client.")

	// И так далее для каждой проверки...
	// Например:
	if cfg.Blockchain.StakeManagerContractAddress == "" {
		log.Error("Stake manager contract address is empty in config.")
		return nil, fmt.Errorf("stake manager contract address is empty")
	}

	//rpcURL := cfg.Blockchain.RpcUrl
	if !strings.HasPrefix(rpcURL, "http://") && !strings.HasPrefix(rpcURL, "https://") {
		rpcURL = "http://" + rpcURL
	}

	//client, err := ethclient.Dial(rpcURL)
	/*if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node at %s: %v", rpcURL, err)
	}*/

	contractABI, err := voting.GetStakeABI()
	if err != nil {
		return nil, fmt.Errorf("failed to load contract ABI: %v", err)
	}

	if cfg.Blockchain.StakeManagerContractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}
	if !common.IsHexAddress(cfg.Blockchain.StakeManagerContractAddress) {
		return nil, fmt.Errorf("invalid contract address format: %s", cfg.Blockchain.StakeManagerContractAddress)
	}
	contractAddr := common.HexToAddress(cfg.Blockchain.StakeManagerContractAddress)

	privateKey, err := crypto.HexToECDSA(cfg.Blockchain.PrivateKey)
	if err != nil {
		log.Error("Failed to parse private key", "error", err)
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Вычисляем публичный адрес из приватного ключа
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	publicKeyAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	if fromAddress.Hex() != strings.ToLower(cfg.Blockchain.WalletAddress) && fromAddress.Hex() != cfg.Blockchain.WalletAddress {
		log.Warn("Configured WalletAddress does not match address derived from PrivateKey",
			slog.String("derived_address", fromAddress.Hex()),
			slog.String("config_address", cfg.Blockchain.WalletAddress))
	}

	return &StakeClient{
		Client:       client,
		contract:     bind.NewBoundContract(contractAddr, contractABI, client, client, client),
		cfg:          cfg,
		contractABI:  contractABI,
		contractAddr: contractAddr,
		privateKey:   privateKey,
		publicKey:    publicKeyAddress,
		FromAddress:  fromAddress,
		log:          log,
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

// Stake вызывает функцию stake из контракта, отправляя ETH
func (sc *StakeClient) Stake(amount *big.Int) (common.Hash, error) {
	nonce, err := sc.Client.PendingNonceAt(context.Background(), sc.FromAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := sc.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(sc.privateKey, big.NewInt(31337)) // ChainID для Anvil
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = amount            // <--- Самое важное: прикрепляем ETH к транзакции
	auth.GasLimit = uint64(300000) // Лимит газа для функции stake
	auth.GasPrice = gasPrice

	sc.log.Info("Preparing to send stake transaction",
		slog.Any("amount_wei", amount), // Логируем сумму в Wei
		slog.String("from_address", sc.FromAddress.Hex()))

	tx, err := sc.contract.Transact(auth, "stake") // Вызываем функцию stake без аргументов
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send stake transaction: %w", err)
	}

	sc.log.Info("Stake transaction sent", slog.String("tx_hash", tx.Hash().Hex()))

	return tx.Hash(), nil
}

// Unstake отправляет транзакцию для вывода застейканного ETH
func (sc *StakeClient) Unstake() (common.Hash, error) {
	chainID, err := sc.Client.ChainID(context.Background())
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(sc.privateKey, chainID)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transactor: %w", err)
	}
	tx, err := sc.contract.Transact(auth, "unstake") // Используем Auth, который был инициализирован в NewVotingClient
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send unstake transaction: %w", err)
	}

	return tx.Hash(), nil
}

// GetTokens отправляет транзакцию для получения токенов (наград)
func (sc *StakeClient) GetTokens(ctx context.Context) (*types.Transaction, error) { // Убрали stakerAddress, т.к. он из PrivateKey
	if sc == nil {
		return nil, errors.New("stake client is nil")
	}

	nonce, err := sc.Client.PendingNonceAt(ctx, sc.publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending nonce for %s: %w", sc.publicKey.Hex(), err)
	}

	gasPrice, err := sc.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	chainID, err := sc.Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(sc.privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0) // Эта функция не отправляет ETH
	auth.GasLimit = 300000     // Адекватный лимит газа
	auth.GasPrice = gasPrice

	tx, err := sc.GetTokens(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to send getTokens transaction: %w", err)
	}

	sc.log.Info("GetTokens transaction sent", "tx_hash", tx.Hash().Hex(), "from_address", sc.publicKey.Hex())
	return tx, nil
}
