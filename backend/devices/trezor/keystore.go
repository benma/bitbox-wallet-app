package trezor

import (
	"bytes"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/btc/addresses"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/coin"
	keystorePkg "github.com/digitalbitbox/bitbox-wallet-app/backend/keystore"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/signing"
	"github.com/digitalbitbox/bitbox-wallet-app/util/errp"
	"github.com/ethereum/go-ethereum/accounts/usbwallet/proto/trezor"
	"github.com/golang/protobuf/proto"
)

type keystore struct {
	device        *Device
	configuration *signing.Configuration
	cosignerIndex int
}

// CosignerIndex implements keystore.Keystore.
func (keystore *keystore) CosignerIndex() int {
	return keystore.cosignerIndex
}

func getCoinName(coin coin.Coin) (string, error) {
	coinName, ok := map[string]string{
		"tbtc": "Testnet",
		"tltc": "Testnet",
		"btc":  "Bitcoin",
		"ltc":  "Litecoin",
	}[coin.Code()]
	if !ok {
		return "", errp.Newf("coin %s not supported", coin.Code())
	}
	return coinName, nil
}

// HasSecureOutput implements keystore.Keystore.
func (keystore *keystore) HasSecureOutput(
	*signing.Configuration, coin.Coin) bool {
	return true
}

// OutputAddress implements keystore.Keystore.
func (keystore *keystore) OutputAddress(configuration *signing.Configuration, coin coin.Coin) error {
	coinName, err := getCoinName(coin)
	if err != nil {
		return err
	}
	yes := true
	if configuration.Multisig() {
		signingThreshold := uint32(configuration.SigningThreshold())
		spendMultisig := trezor.InputScriptType_SPENDMULTISIG
		_, err = keystore.device.trezorCall(
			&trezor.GetAddress{
				AddressN:    configuration.AbsoluteKeypath().ToUInt32(),
				CoinName:    &coinName,
				ShowDisplay: &yes,
				ScriptType:  &spendMultisig,
				Multisig: &trezor.MultisigRedeemScriptType{
					Pubkeys: toTrezorXPubs(configuration, nil),
					M:       &signingThreshold,
				},
			},
			new(trezor.Address),
		)
	} else {
		_, err = keystore.device.trezorCall(
			&trezor.GetAddress{
				AddressN:    configuration.AbsoluteKeypath().ToUInt32(),
				CoinName:    &coinName,
				ShowDisplay: &yes,
				ScriptType:  toTrezorInputScriptType(configuration.ScriptType()),
			},
			new(trezor.Address),
		)
	}
	return err
}

// ExtendedPublicKey implements keystore.Keystore.
func (keystore *keystore) ExtendedPublicKey(
	keyPath signing.AbsoluteKeypath) (*hdkeychain.ExtendedKey, error) {
	pk := new(trezor.PublicKey)
	_, err := keystore.device.trezorCall(
		&trezor.GetPublicKey{AddressN: keyPath.ToUInt32()},
		pk,
	)
	if err != nil {
		return nil, err
	}
	return hdkeychain.NewKeyFromString(*pk.Xpub)
}

func toTrezorInputScriptType(scriptType signing.ScriptType) *trezor.InputScriptType {
	trezorScriptType, ok := map[signing.ScriptType]trezor.InputScriptType{
		signing.ScriptTypeP2PKH:      trezor.InputScriptType_SPENDADDRESS,
		signing.ScriptTypeP2WPKHP2SH: trezor.InputScriptType_SPENDP2SHWITNESS,
		signing.ScriptTypeP2WPKH:     trezor.InputScriptType_SPENDWITNESS,
	}[scriptType]
	if !ok {
		panic("unsupported script type")
	}
	return &trezorScriptType
}

func toTrezorOutputScriptType(scriptType signing.ScriptType) *trezor.OutputScriptType {
	trezorScriptType, ok := map[signing.ScriptType]trezor.OutputScriptType{
		signing.ScriptTypeP2PKH:      trezor.OutputScriptType_PAYTOADDRESS,
		signing.ScriptTypeP2WPKHP2SH: trezor.OutputScriptType_PAYTOP2SHWITNESS,
		signing.ScriptTypeP2WPKH:     trezor.OutputScriptType_PAYTOWITNESS,
	}[scriptType]
	if !ok {
		panic("unsupported script type")
	}
	return &trezorScriptType
}

func toTrezorXPubs(configuration *signing.Configuration, addressN []uint32) []*trezor.HDNodePathType {
	xpubs := configuration.SortedExtendedPublicKeys()
	result := make([]*trezor.HDNodePathType, len(xpubs))
	for index, xpub := range xpubs {
		depth := uint32(xpub.Depth())
		fingerprint := xpub.ParentFingerprint()
		childNum := xpub.ChildNum()
		publicKey, err := xpub.ECPubKey()
		if err != nil {
			panic(err)
		}
		result[index] = &trezor.HDNodePathType{
			Node: &trezor.HDNodeType{
				Depth:       &depth,
				Fingerprint: &fingerprint,
				ChildNum:    &childNum,
				ChainCode:   xpub.ChainCode(),
				PublicKey:   publicKey.SerializeCompressed(),
			},
			AddressN: addressN,
		}
	}
	return result
}

func toTrezorMultisig(address *addresses.AccountAddress) *trezor.MultisigRedeemScriptType {
	signingThreshold := uint32(address.Configuration.SigningThreshold())
	return &trezor.MultisigRedeemScriptType{
		Pubkeys: toTrezorXPubs(address.AccountConfiguration, address.RelativeKeypath.ToUInt32()),
		M:       &signingThreshold,
	}
}

func reverse(b []byte) []byte {
	result := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		result[i] = b[len(b)-1-i]
	}
	return result
}

// findPreviousTx takes a tx hash provided by the trezor and returns the corresponding input
// transaction. Under normal operations, this should not return an error, as trezor only asks for
// input transactions referenced by the main tx, and we have those.
func findPreviousTx(btcProposedTx *btc.ProposedTransaction, txHash []byte) (*wire.MsgTx, error) {
	hash, err := chainhash.NewHash(reverse(txHash))
	if err != nil {
		return nil, errp.WithStack(err)
	}
	for outPoint, txOut := range btcProposedTx.PreviousOutputs {
		if outPoint.Hash == *hash {
			return txOut.Tx, nil
		}
	}
	return nil, errp.Newf("prevoius tx not found: %s", hash)
}

func (keystore *keystore) signBTCTransaction(btcProposedTx *btc.ProposedTransaction) error {
	tx := btcProposedTx.TXProposal.Transaction
	outputsCount := uint32(len(tx.TxOut))
	inputsCount := uint32(len(tx.TxIn))
	version := uint32(tx.Version)

	coinName, err := getCoinName(btcProposedTx.TXProposal.Coin)
	if err != nil {
		return err
	}
	signTx := &trezor.SignTx{
		OutputsCount: &outputsCount,
		InputsCount:  &inputsCount,
		CoinName:     &coinName,
		Version:      &version,
		LockTime:     &tx.LockTime,
	}
	ser := bytes.Buffer{}
	var send proto.Message = signTx
	for {
		txRequest := new(trezor.TxRequest)
		_, err := keystore.device.trezorCall(
			send,
			txRequest,
		)
		if trezorErr, ok := errp.Cause(err).(*trezorError); ok && *trezorErr.Code == trezor.FailureType_Failure_ActionCancelled {
			return errp.WithStack(keystorePkg.ErrSigningAborted)
		}
		if err != nil {
			return err
		}
		if txRequest.Serialized != nil && len(txRequest.Serialized.SerializedTx) != 0 {
			ser.Write(txRequest.Serialized.SerializedTx)
		}
		if txRequest.Serialized != nil && txRequest.Serialized.SignatureIndex != nil {
			signature, err := btcec.ParseDERSignature(txRequest.Serialized.Signature, btcec.S256())
			if err != nil {
				return errp.WithStack(err)
			}
			btcProposedTx.Signatures[*txRequest.Serialized.SignatureIndex][keystore.CosignerIndex()] = signature
		}

		if *txRequest.RequestType == trezor.RequestType_TXFINISHED {
			break
		}
		if *txRequest.RequestType == trezor.RequestType_TXMETA {
			if len(txRequest.Details.TxHash) == 0 {
				panic("should not have be requested by trezor")
			}
			previousTx, err := findPreviousTx(btcProposedTx, txRequest.Details.TxHash)
			if err != nil {
				return err
			}
			outputsCount := uint32(len(previousTx.TxOut))
			inputsCount := uint32(len(previousTx.TxIn))
			version := uint32(previousTx.Version)

			send = &trezor.TxAck{Tx: &trezor.TransactionType{
				OutputsCnt: &outputsCount,
				InputsCnt:  &inputsCount,
				Version:    &version,
				LockTime:   &previousTx.LockTime,
			}}
		} else if *txRequest.RequestType == trezor.RequestType_TXINPUT {
			var txIn *wire.TxIn
			input := &trezor.TxInputType{}
			if len(txRequest.Details.TxHash) == 0 {
				txIn = tx.TxIn[*txRequest.Details.RequestIndex]
				spentOutput, ok := btcProposedTx.PreviousOutputs[txIn.PreviousOutPoint]
				if !ok {
					panic("There needs to be exactly one output being spent per input!")
				}
				address := btcProposedTx.GetAddress(spentOutput.ScriptHashHex())
				input.AddressN = address.Configuration.AbsoluteKeypath().ToUInt32()
				if address.Configuration.Multisig() {
					spendMultisig := trezor.InputScriptType_SPENDMULTISIG
					input.ScriptType = &spendMultisig
					input.Multisig = toTrezorMultisig(address)
				} else {
					input.ScriptType = toTrezorInputScriptType(address.Configuration.ScriptType())
				}
				amount := uint64(spentOutput.Value)
				input.Amount = &amount
			} else {
				previousTx, err := findPreviousTx(btcProposedTx, txRequest.Details.TxHash)
				if err != nil {
					return err
				}
				txIn = previousTx.TxIn[*txRequest.Details.RequestIndex]
				input.ScriptSig = txIn.SignatureScript
			}
			input.PrevHash = reverse(txIn.PreviousOutPoint.Hash.CloneBytes())
			input.PrevIndex = &txIn.PreviousOutPoint.Index
			input.Sequence = &txIn.Sequence
			send = &trezor.TxAck{Tx: &trezor.TransactionType{Inputs: []*trezor.TxInputType{input}}}
		} else if *txRequest.RequestType == trezor.RequestType_TXOUTPUT {
			if len(txRequest.Details.TxHash) == 0 {
				txOut := tx.TxOut[*txRequest.Details.RequestIndex]
				output := &trezor.TxOutputType{}
				amount := uint64(txOut.Value)
				output.Amount = &amount
				changeAddress := btcProposedTx.TXProposal.ChangeAddress
				isChange := changeAddress != nil && bytes.Equal(
					changeAddress.PubkeyScript(),
					txOut.PkScript,
				)
				if isChange {
					output.AddressN = changeAddress.Configuration.AbsoluteKeypath().ToUInt32()
					if changeAddress.Configuration.Multisig() {
						spendMultisig := trezor.OutputScriptType_PAYTOMULTISIG
						output.ScriptType = &spendMultisig
						output.Multisig = toTrezorMultisig(changeAddress)
					} else {
						output.ScriptType = toTrezorOutputScriptType(changeAddress.Configuration.ScriptType())
					}
				} else {
					address := btcProposedTx.RecipientAddress
					output.Address = &address
					payToAddress := trezor.OutputScriptType_PAYTOADDRESS
					output.ScriptType = &payToAddress
				}
				send = &trezor.TxAck{Tx: &trezor.TransactionType{Outputs: []*trezor.TxOutputType{output}}}
			} else {
				previousTx, err := findPreviousTx(btcProposedTx, txRequest.Details.TxHash)
				if err != nil {
					return err
				}
				txOut := previousTx.TxOut[*txRequest.Details.RequestIndex]
				amount := uint64(txOut.Value)
				output := &trezor.TxOutputBinType{
					Amount:       &amount,
					ScriptPubkey: txOut.PkScript,
				}
				send = &trezor.TxAck{Tx: &trezor.TransactionType{BinOutputs: []*trezor.TxOutputBinType{output}}}
			}
		}
	}
	return nil
}

// SignTransaction implements keystore.Keystore.
func (keystore *keystore) SignTransaction(proposedTx coin.ProposedTransaction) error {
	switch specificProposedTx := proposedTx.(type) {
	case *btc.ProposedTransaction:
		return keystore.signBTCTransaction(specificProposedTx)
	default:
		panic("unknown proposal type")
	}
}
