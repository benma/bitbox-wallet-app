// SPDX-License-Identifier: Apache-2.0

package bitbox02

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/BitBoxSwiss/bitbox02-api-go/api/common"
	"github.com/BitBoxSwiss/bitbox02-api-go/api/firmware"
	"github.com/BitBoxSwiss/bitbox02-api-go/api/firmware/messages"
	firmwaremocks "github.com/BitBoxSwiss/bitbox02-api-go/api/firmware/mocks"
	"github.com/BitBoxSwiss/bitbox02-api-go/util/semver"
	"github.com/flynn/noise"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestKeystoreRootFingerprintCacheClearedOnInitializedStatusChange(t *testing.T) {
	firstFingerprint := []byte{0xaa, 0xbb, 0xcc, 0xdd}
	secondFingerprint := []byte{0x11, 0x22, 0x33, 0x44}
	responseCipher := newTestCipherState()
	queryCalls := 0

	device := NewDevice(
		"test-device",
		semver.NewSemVer(9, 0, 0),
		common.ProductBitBox02Multi,
		&firmwaremocks.Config{},
		&firmwaremocks.Communication{
			MockQuery: func([]byte) ([]byte, error) {
				var fingerprint []byte
				switch queryCalls {
				case 0:
					fingerprint = firstFingerprint
				case 1:
					fingerprint = secondFingerprint
				default:
					require.FailNow(t, "unexpected extra device query")
				}
				queryCalls++
				return encodeFingerprintResponse(t, responseCipher, fingerprint), nil
			},
			MockClose: func() {},
		},
	)
	prepareDeviceForEncryptedQueries(t, device)

	fingerprint, err := device.keystore.RootFingerprint()
	require.NoError(t, err)
	require.Equal(t, firstFingerprint, fingerprint)

	invokeDeviceStatusChanged(t, device, firmware.StatusInitialized)

	fingerprint, err = device.keystore.RootFingerprint()
	require.NoError(t, err)
	require.Equal(t, secondFingerprint, fingerprint)
	require.Equal(t, 2, queryCalls)
}

func prepareDeviceForEncryptedQueries(t *testing.T, device *Device) {
	t.Helper()
	setUnexportedField(t, &device.Device, "sendCipher", newTestCipherState())
	setUnexportedField(t, &device.Device, "receiveCipher", newTestCipherState())
	setUnexportedField(t, &device.Device, "channelHashAppVerified", true)
	setUnexportedField(t, &device.Device, "channelHashDeviceVerified", true)
}

func newTestCipherState() *noise.CipherState {
	suite := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	return noise.UnsafeNewCipherState(suite, [32]byte{}, 0)
}

func encodeFingerprintResponse(
	t *testing.T,
	responseCipher *noise.CipherState,
	fingerprint []byte,
) []byte {
	t.Helper()
	responseBytes, err := proto.Marshal(&messages.Response{
		Response: &messages.Response_Fingerprint{
			Fingerprint: &messages.RootFingerprintResponse{
				Fingerprint: fingerprint,
			},
		},
	})
	require.NoError(t, err)
	encryptedResponse, err := responseCipher.Encrypt(nil, nil, responseBytes)
	require.NoError(t, err)
	return append([]byte{0x00, 0x00}, encryptedResponse...)
}

func invokeDeviceStatusChanged(t *testing.T, device *Device, status firmware.Status) {
	t.Helper()
	setUnexportedField(t, &device.Device, "status", status)
	onEvent := getUnexportedField[func(firmware.Event, interface{})](t, &device.Device, "onEvent")
	require.NotNil(t, onEvent)
	onEvent(firmware.EventStatusChanged, nil)
}

func getUnexportedField[T any](t *testing.T, target any, fieldName string) T {
	t.Helper()
	targetValue := reflect.ValueOf(target)
	require.Equal(t, reflect.Pointer, targetValue.Kind())
	fieldValue := targetValue.Elem().FieldByName(fieldName)
	require.Truef(t, fieldValue.IsValid(), "field %q not found", fieldName)
	return reflect.NewAt(fieldValue.Type(), unsafe.Pointer(fieldValue.UnsafeAddr())).Elem().
		Interface().(T)
}

func setUnexportedField(t *testing.T, target any, fieldName string, value any) {
	t.Helper()
	targetValue := reflect.ValueOf(target)
	require.Equal(t, reflect.Pointer, targetValue.Kind())
	fieldValue := targetValue.Elem().FieldByName(fieldName)
	require.Truef(t, fieldValue.IsValid(), "field %q not found", fieldName)
	reflect.NewAt(fieldValue.Type(), unsafe.Pointer(fieldValue.UnsafeAddr())).Elem().
		Set(reflect.ValueOf(value))
}
