package trezor

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/davecgh/go-spew/spew"
	devicepkg "github.com/digitalbitbox/bitbox-wallet-app/backend/devices/device"
	keystoreInterface "github.com/digitalbitbox/bitbox-wallet-app/backend/keystore"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/signing"
	"github.com/digitalbitbox/bitbox-wallet-app/util/errp"
	"github.com/digitalbitbox/bitbox-wallet-app/util/locker"
	"github.com/digitalbitbox/bitbox-wallet-app/util/logging"
	"github.com/ethereum/go-ethereum/accounts/usbwallet/proto/trezor"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
)

// ProductName is the name of the trezor product.
const ProductName = "trezor"

type Device struct {
	deviceID            string
	device              io.ReadWriteCloser
	deviceLock          locker.Locker
	features            *trezor.Features
	onEvent             func(devicepkg.Event, interface{})
	pinCh, passphraseCh chan string
	status              string
	log                 *logrus.Entry
}

func NewDevice(deviceID string, device io.ReadWriteCloser) *Device {
	log := logging.Get().WithGroup("device").WithField("deviceID", deviceID)
	log.Info("Plugged in device")
	return &Device{
		deviceID:     deviceID,
		device:       device,
		pinCh:        make(chan string),
		passphraseCh: make(chan string),
		log:          log,
	}
}

// ProductName implements device.Device.
func (device *Device) ProductName() string {
	return ProductName
}

// Init implements device.Device.
func (device *Device) Init(testing bool) error {
	spew.Dump("INIT")
	features := new(trezor.Features)
	if _, err := device.trezorExchange(&trezor.Initialize{}, features); err != nil {
		return err
	}
	device.features = features
	spew.Dump("FEATURES", features)
	return device.ping()
}

func (device *Device) trezorCall(req proto.Message, results ...proto.Message) (int, error) {
	allResults := []proto.Message{
		new(trezor.PinMatrixRequest),
		new(trezor.PassphraseRequest),
	}
	allResults = append(allResults, results...)
	res, err := device.trezorExchange(
		req,
		allResults...,
	)
	if err != nil {
		return 0, err
	}
	if res == 0 {
		device.status = "pinRequired"
		device.fireEvent("statusChanged", nil)
		fmt.Println("asking pin")
		pin := <-device.pinCh
		fmt.Println("got pin 2", pin)
		_, err := device.trezorCall(
			&trezor.PinMatrixAck{Pin: &pin},
			results...)

		if trezorErr, ok := errp.Cause(err).(*trezorError); ok {
			switch *trezorErr.Code {
			case trezor.FailureType_Failure_PinExpected:
				fallthrough
			case trezor.FailureType_Failure_PinCancelled:
				fallthrough
			case trezor.FailureType_Failure_PinInvalid:
				return device.trezorCall(req, results...)
			}
		}

	} else if res == 1 {
		device.status = "passphraseRequired"
		device.fireEvent("statusChanged", nil)
		fmt.Println("asking passphrase")
		passphrase := <-device.passphraseCh
		device.status = ""
		device.fireEvent("statusChanged", nil)
		return device.trezorCall(
			&trezor.PassphraseAck{Passphrase: &passphrase},
			results...)
	}
	return res - 2, nil
}

func (device *Device) ping() error {
	yes := true
	_, err := device.trezorCall(
		&trezor.Ping{
			PinProtection:        &yes,
			PassphraseProtection: &yes,
		},
		new(trezor.Success))
	if err != nil {
		return err
	}
	device.status = "seeded"
	device.fireEvent("statusChanged", nil)
	device.fireEvent(devicepkg.EventKeystoreAvailable, nil)
	return nil
}

// Identifier implements device.Device.
func (device *Device) Identifier() string {
	return device.deviceID
}

// KeystoreForConfiguration implements device.Device.
func (device *Device) KeystoreForConfiguration(configuration *signing.Configuration, cosignerIndex int) keystoreInterface.Keystore {
	return &keystore{
		device:        device,
		configuration: configuration,
		cosignerIndex: cosignerIndex,
	}
}

// SetOnEvent implements device.Device.
func (device *Device) SetOnEvent(onEvent func(devicepkg.Event, interface{})) {
	device.onEvent = onEvent
}

func (device *Device) fireEvent(event devicepkg.Event, data interface{}) {
	f := device.onEvent
	if f != nil {
		f(event, data)
	}
}

// Close implements device.Device.
func (device *Device) Close() {
	if err := device.device.Close(); err != nil {
		panic(err)
	}
}

func (device *Device) Status() string {
	return device.status
}

func (device *Device) PIN(pin string) error {
	fmt.Println("got pin", pin)
	device.pinCh <- pin
	return nil
}

func (device *Device) Passphrase(passphrase string) error {
	device.passphraseCh <- passphrase
	return nil
}

// errTrezorReplyInvalidHeader is the error message returned by a Trezor data exchange
// if the device replies with a mismatching header. This usually means the device
// is in browser mode.
var errTrezorReplyInvalidHeader = errors.New("trezor: invalid reply header")

type trezorError struct {
	*trezor.Failure
}

func (e trezorError) Error() string {
	return *e.Message
}

// trezorExchange performs a data exchange with the Trezor wallet, sending it a
// message and retrieving the response. If multiple responses are possible, the
// method will also return the index of the destination object used.
func (device *Device) trezorExchange(req proto.Message, results ...proto.Message) (int, error) {
	// Construct the original message payload to chunk up
	data, err := proto.Marshal(req)
	if err != nil {
		return 0, errp.WithStack(err)
	}
	payload := make([]byte, 8+len(data))
	copy(payload, []byte{0x23, 0x23})
	binary.BigEndian.PutUint16(payload[2:], trezor.Type(req))
	binary.BigEndian.PutUint32(payload[4:], uint32(len(data)))
	copy(payload[8:], data)

	// Stream all the chunks to the device
	chunk := make([]byte, 64)
	chunk[0] = 0x3f // Report ID magic number

	for len(payload) > 0 {
		// Construct the new message to stream, padding with zeroes if needed
		if len(payload) > 63 {
			copy(chunk[1:], payload[:63])
			payload = payload[63:]
		} else {
			copy(chunk[1:], payload)
			copy(chunk[1+len(payload):], make([]byte, 63-len(payload)))
			payload = nil
		}
		// Send over to the device
		if _, err := device.device.Write(chunk); err != nil {
			return 0, errp.WithStack(err)
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var (
		kind  uint16
		reply []byte
	)
	for {
		// Read the next chunk from the Trezor wallet
		if _, err := io.ReadFull(device.device, chunk); err != nil {
			return 0, errp.WithStack(err)
		}

		// Make sure the transport header matches
		if chunk[0] != 0x3f || (len(reply) == 0 && (chunk[1] != 0x23 || chunk[2] != 0x23)) {
			return 0, errp.WithStack(errTrezorReplyInvalidHeader)
		}
		// If it's the first chunk, retrieve the reply message type and total message length
		var payload []byte

		if len(reply) == 0 {
			kind = binary.BigEndian.Uint16(chunk[3:5])
			reply = make([]byte, 0, int(binary.BigEndian.Uint32(chunk[5:9])))
			payload = chunk[9:]
		} else {
			payload = chunk[1:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}
	}
	// Try to parse the reply into the requested reply message
	if kind == uint16(trezor.MessageType_MessageType_Failure) {
		// Trezor returned a failure, extract and return the message
		failure := new(trezor.Failure)
		if err := proto.Unmarshal(reply, failure); err != nil {
			return 0, errp.WithStack(err)
		}
		return 0, errp.WithStack(&trezorError{failure})
	}
	if kind == uint16(trezor.MessageType_MessageType_ButtonRequest) {
		// Trezor is waiting for user confirmation, ack and wait for the next message
		return device.trezorExchange(&trezor.ButtonAck{}, results...)
	}
	for i, res := range results {
		if trezor.Type(res) == kind {
			return i, errp.WithStack(proto.Unmarshal(reply, res))
		}
	}
	expected := make([]string, len(results))
	for i, res := range results {
		expected[i] = trezor.Name(trezor.Type(res))
	}
	return 0, errp.Newf("trezor: expected reply types %s, got %s", expected, trezor.Name(kind))
}
