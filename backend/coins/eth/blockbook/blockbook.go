// Copyright 2022 Shift Crypto AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blockbook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/digitalbitbox/bitbox-wallet-app/backend/accounts"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/coin"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/eth/erc20"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/eth/rpcclient"
	ethtypes "github.com/digitalbitbox/bitbox-wallet-app/backend/coins/eth/types"
	"github.com/digitalbitbox/bitbox-wallet-app/util/errp"
	"github.com/digitalbitbox/bitbox-wallet-app/util/locker"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Blockbook is a client to a Blockbook ETH backend.
type Blockbook struct {
	apiURL string

	websocketLock locker.Locker
	websocketURL  string
	websocketConn *websocket.Conn

	httpClient *http.Client
}

func New(apiURL string, websocketURL string, httpClient *http.Client) *Blockbook {
	b := &Blockbook{
		apiURL:       apiURL,
		websocketURL: websocketURL,
		httpClient:   httpClient,
	}
	var res map[string]interface{}
	fmt.Println("LOL E", b.websocketRequest("getInfo", nil, &res))
	return b
}

func (b *Blockbook) call(endpoint string, params url.Values, result interface{}) error {
	url := b.apiURL + "/api/v2/" + endpoint
	if params != nil {
		url += "?" + params.Encode()
	}
	response, err := b.httpClient.Get(url)
	if err != nil {
		return errp.WithStack(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return errp.Newf("expected 200 OK, got %d", response.StatusCode)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errp.WithStack(err)
	}
	if err := json.Unmarshal(body, result); err != nil {
		return errp.Newf("unexpected response from Blockbook: %s", err)
	}
	return nil
}

func (b *Blockbook) maybeConnect() (*websocket.Conn, error) {
	if b.websocketConn != nil {
		return b.websocketConn, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, b.websocketURL, nil)
	if err != nil {
		return nil, err
	}
	b.websocketConn = conn
	return conn, nil
}

func (b *Blockbook) websocketRequest(method string, params interface{}, result interface{}) error {
	defer b.websocketLock.Lock()()
	conn, err := b.maybeConnect()
	if err != nil {
		return err
	}
	err = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a, _ := json.Marshal(map[string]interface{}{"id": "1", "method": method, "params": params})
		fmt.Println(string(a))
		return wsjson.Write(ctx, conn, map[string]interface{}{"id": "1", "method": method, "params": params})
	}()
	if err != nil {
		return errp.WithStack(err)
	}
	err = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		return wsjson.Read(ctx, conn, result)
	}()
	if err != nil {
		return errp.WithStack(err)
	}
	fmt.Println("LOL", result)
	return nil
}

type jsonBigInt big.Int

func (jsBigInt jsonBigInt) BigInt() *big.Int {
	bigInt := big.Int(jsBigInt)
	return &bigInt
}

// UnmarshalJSON implements json.Unmarshaler.
func (jsBigInt *jsonBigInt) UnmarshalJSON(jsonBytes []byte) error {
	var numberString string
	if err := json.Unmarshal(jsonBytes, &numberString); err != nil {
		return errp.WithStack(err)
	}
	bigInt, ok := new(big.Int).SetString(numberString, 10)
	if !ok {
		return errp.Newf("failed to parse %s", numberString)
	}
	*jsBigInt = jsonBigInt(*bigInt)
	return nil
}

type Transaction struct {
	TxID string
	VIn  [1]struct {
		Addresses [1]string
		IsOwn     bool
	}
	VOut [1]struct {
		Addresses [1]string
		IsOwn     bool
	}
	BlockHeight    int
	Confirmations  int
	BlockTime      int64
	Value          jsonBigInt
	Fees           jsonBigInt
	TokenTransfers []struct {
		Type     string
		From     string
		To       string
		Token    string
		Name     string
		Symbol   string
		Decimals int
		Value    jsonBigInt
	}
	EthereumSpecific struct {
		Status   int
		Nonce    int
		GasLimit int
		GasUsed  int
		GasPrice jsonBigInt
	}
}

func (tx *Transaction) fee() *coin.Amount {
	fee := coin.NewAmount(tx.Fees.BigInt())
	return &fee
}

func (tx *Transaction) isInternal() bool {
	return len(tx.TokenTransfers) > 0
}

func (tx *Transaction) internalID() string {
	id := tx.TxID
	if tx.isInternal() {
		id += "-internal"
	}
	return id
}

func (tx *Transaction) status() accounts.TxStatus {
	if tx.EthereumSpecific.Status == 0 {
		return accounts.TxStatusFailed
	}
	if tx.Confirmations >= ethtypes.NumConfirmationsComplete {
		return accounts.TxStatusComplete
	}
	return accounts.TxStatusPending
}

func (tx *Transaction) TransactionData() *accounts.TransactionData {
	fromOurs := tx.VIn[0].IsOwn
	to := tx.VOut[0].Addresses[0]
	toOurs := tx.VOut[0].IsOwn
	var txType accounts.TxType
	switch {
	case fromOurs && toOurs:
		txType = accounts.TxTypeSendSelf
	case !fromOurs && toOurs:
		txType = accounts.TxTypeReceive
	case fromOurs && !toOurs:
		txType = accounts.TxTypeSend
	default:
		if len(tx.TokenTransfers) != 0 {
			return nil
		}
		panic("not our transaction")
	}
	timestamp := time.Unix(tx.BlockTime, 0)
	amount := coin.NewAmount(tx.Value.BigInt())
	return &accounts.TransactionData{
		Fee:                      tx.fee(),
		FeeIsDifferentUnit:       false,
		Timestamp:                &timestamp,
		TxID:                     tx.TxID,
		InternalID:               tx.internalID(),
		Height:                   tx.BlockHeight,
		NumConfirmations:         tx.Confirmations,
		NumConfirmationsComplete: ethtypes.NumConfirmationsComplete,
		Status:                   tx.status(),
		Type:                     txType,
		Amount:                   amount,
		Addresses: []accounts.AddressAndAmount{{
			Address: to,
			Amount:  amount,
		}},
	}
}

func (tx *Transaction) TransactionDataERC20(ourAddress string, erc20Token *erc20.Token) []*accounts.TransactionData {
	contractAddressLower := strings.ToLower(erc20Token.ContractAddress().Hex())
	ourAddressLower := strings.ToLower(ourAddress)
	result := []*accounts.TransactionData{}
	for _, tokenTransfer := range tx.TokenTransfers {
		if strings.ToLower(tokenTransfer.Token) != contractAddressLower {
			continue
		}
		fromOurs := strings.ToLower(tokenTransfer.From) == ourAddressLower
		toOurs := strings.ToLower(tokenTransfer.To) == ourAddressLower
		var txType accounts.TxType
		switch {
		case fromOurs && toOurs:
			txType = accounts.TxTypeSendSelf
		case !fromOurs && toOurs:
			txType = accounts.TxTypeReceive
		case fromOurs && !toOurs:
			txType = accounts.TxTypeSend
		default:
			panic("not our transaction")
		}
		timestamp := time.Unix(tx.BlockTime, 0)
		amount := coin.NewAmount(tokenTransfer.Value.BigInt())
		result = append(result, &accounts.TransactionData{
			Fee:                      tx.fee(),
			FeeIsDifferentUnit:       true,
			Timestamp:                &timestamp,
			TxID:                     tx.TxID,
			InternalID:               tx.internalID(),
			Height:                   tx.BlockHeight,
			NumConfirmations:         tx.Confirmations,
			NumConfirmationsComplete: ethtypes.NumConfirmationsComplete,
			Status:                   tx.status(),
			Type:                     txType,
			Amount:                   amount,
			Addresses: []accounts.AddressAndAmount{{
				Address: tokenTransfer.To,
				Amount:  amount,
			}},
		})
	}
	return result
}

type AddressResult struct {
	Page               int
	Balance            jsonBigInt
	UnconfirmedBalance jsonBigInt
	Nonce              jsonBigInt
	TotalPages         int
	ItemsOnPage        int
	Transactions       []Transaction
	Tokens             []struct {
		Contract string
		Balance  jsonBigInt
	}
}

func (b *Blockbook) Transactions(
	blockTipHeight *big.Int,
	address common.Address,
	endBlock *big.Int,
	erc20Token *erc20.Token) ([]*accounts.TransactionData, error) {
	var result AddressResult
	params := url.Values{}
	params.Set("page", "1")
	params.Set("pageSize", "1000")
	params.Set("details", "txs")
	if err := b.call("address/"+address.Hex(), params, &result); err != nil {
		return nil, err
	}
	for page := 2; page <= result.TotalPages; page++ {
		params.Set("page", fmt.Sprint(page))
		var nextResult AddressResult
		if err := b.call("address/"+address.Hex(), params, &nextResult); err != nil {
			return nil, err
		}
		result.Transactions = append(result.Transactions, nextResult.Transactions...)
	}

	transactions := []*accounts.TransactionData{}
	isERC20 := erc20Token != nil
	for _, tx := range result.Transactions {
		if isERC20 {
			transactions = append(transactions, tx.TransactionDataERC20(address.Hex(), erc20Token)...)
		} else {
			txData := tx.TransactionData()
			if txData != nil {
				transactions = append(transactions, txData)
			}
		}

	}
	return transactions, nil
}

func (b *Blockbook) Balance(ctx context.Context, account common.Address) (*big.Int, error) {
	var result AddressResult
	params := url.Values{}
	params.Set("details", "basic")
	if err := b.call("address/"+account.Hex(), params, &result); err != nil {
		return nil, err
	}
	return result.Balance.BigInt(), nil
}

// ERC20Balance implements rpc.Interface.
func (b *Blockbook) ERC20Balance(account common.Address, erc20Token *erc20.Token) (*big.Int, error) {
	var result AddressResult
	params := url.Values{}
	params.Set("details", "tokenBalances")
	if err := b.call("address/"+account.Hex(), params, &result); err != nil {
		return nil, err
	}
	contractAddressLower := strings.ToLower(erc20Token.ContractAddress().Hex())
	for _, token := range result.Tokens {
		if strings.ToLower(token.Contract) == contractAddressLower {
			return token.Balance.BigInt(), nil
		}
	}
	return big.NewInt(0), nil
}

// CallContract implements rpc.Interface.
func (b *Blockbook) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	panic("TODO")
}

type feeResult struct {
	Data []struct {
		FeeLimit   string
		FeePerUnit string
	}
}

func (b *Blockbook) estimateFee(ctx context.Context, msg ethereum.CallMsg) (*feeResult, error) {
	var result feeResult
	err := b.websocketRequest("estimateFee", map[string]interface{}{
		// blocks is ignored in case of ETH, gas limit is alwasy the same and the backend calls the
		// `eth_gasPrice` RPC function, which does not take any blocks argument.  There has to be at
		// least one element for the result to contain one element. This is a API wart so the same
		// API also works with Bitcoin.
		"blocks": []int{1},
		"specific": map[string]interface{}{
			"from":     msg.From.Hex(),
			"to":       msg.To.Hex(),
			"data":     hexutil.Encode(msg.Data),
			"gas":      hexutil.EncodeUint64(msg.Gas),
			"gasPrice": hexutil.EncodeBig(msg.GasPrice),
		},
	}, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Data) != 1 {
		return nil, errp.New("unexpected result for estimateFee")
	}
	return &result, nil
}

// EstimateGas implements rpc.Interface.
func (b *Blockbook) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	result, err := b.estimateFee(ctx, msg)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(result.Data[0].FeeLimit, 10, 64)
}

// BlockNumber implements rpc.Interface.
func (b *Blockbook) BlockNumber(ctx context.Context) (*big.Int, error) {
	var result struct {
		Backend struct {
			Blocks int64
		}
	}
	params := url.Values{}
	if err := b.call("status", params, &result); err != nil {
		return nil, err
	}
	return big.NewInt(result.Backend.Blocks), nil
}

// PendingNonceAt implements rpc.Interface.
func (b *Blockbook) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result AddressResult
	params := url.Values{}
	params.Set("details", "basic")
	if err := b.call("address/"+account.Hex(), params, &result); err != nil {
		return 0, err
	}
	return result.Nonce.BigInt().Uint64(), nil
}

// SendTransaction implements rpc.Interface.
func (b *Blockbook) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return errp.WithStack(err)
	}
	fmt.Println("LOL SEND", (hexutil.Encode(encodedTx)))
	response, err := b.httpClient.Post(
		b.apiURL+"/api/v2/sendtx/",
		"text/plain",
		bytes.NewBuffer([]byte(hexutil.Encode(encodedTx))),
	)
	if err != nil {
		return errp.WithStack(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errp.WithStack(err)
	}
	if response.StatusCode != http.StatusOK {
		return errp.Newf("expected 200 OK, got %d. Body: %s", response.StatusCode, string(body))
	}
	var result struct {
		Result string
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return errp.Newf("unexpected response from Blockbook: %s", err)
	}
	fmt.Println("LOL", result)

	return nil
}

// SuggestGasPrice implements rpc.Interface.
func (b *Blockbook) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return nil, errp.New("not supported")
}

// TransactionByHash implements rpc.Interface.
func (b *Blockbook) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	panic("TODO")
}

// TransactionReceiptWithBlockNumber implements rpc.Interface.
func (b *Blockbook) TransactionReceiptWithBlockNumber(
	ctx context.Context, hash common.Hash) (*rpcclient.RPCTransactionReceipt, error) {
	panic("TODO")
}
