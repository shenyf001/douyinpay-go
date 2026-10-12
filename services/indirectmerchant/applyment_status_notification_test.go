package indirectmerchant

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/douyinpay/douyinpay-go/services/callback"
	"github.com/douyinpay/douyinpay-go/tools/auth/verifiers"
	"github.com/douyinpay/douyinpay-go/tools/consts"
)

func TestParseApplymentStatusNotify(t *testing.T) {
	const aesKey = "01234567890123456789012345678901"
	const wrongAESKey = "11234567890123456789012345678901"
	const serial = "platform-notification-test"
	const timestamp = "1791763200"
	const headerNonce = "request-nonce"
	const resourceNonce = "0123456789ab"
	const associatedData = "merchant"
	const event = "INDIRECT_MERCHANT_APPLYMENT_STATUS"

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &verifiers.SHA256WithRSAVerifier{
		PublicKey: &privateKey.PublicKey, SerialNumber: serial,
	}
	plaintext := func(state string) string {
		return fmt.Sprintf(`{"sp_mchid":"6120230301343998","applyment_id":"NTAM7638162829987562028","business_code":"business_1","indirect_merchant_id":"2026020211234567890","applyment_state":%q,"fail_reason":"fixture reason"}`, state)
	}
	bodyFor := func(t *testing.T, state string, tamperCiphertext bool) []byte {
		t.Helper()
		block, err := aes.NewCipher([]byte(aesKey))
		if err != nil {
			t.Fatal(err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			t.Fatal(err)
		}
		ciphertext := gcm.Seal(nil, []byte(resourceNonce), []byte(plaintext(state)), []byte(associatedData))
		if tamperCiphertext {
			ciphertext[len(ciphertext)-1] ^= 1
		}
		body, err := json.Marshal(map[string]interface{}{
			"id": "notification_1", "create_time": "2026-10-12T08:00:00+08:00",
			"event_type": event, "resource_type": "encrypt-resource", "summary": "status changed",
			"resource": map[string]string{
				"original_type": "merchant", "algorithm": consts.EncryptTypeAES256GCM,
				"ciphertext":      base64.StdEncoding.EncodeToString(ciphertext),
				"associated_data": associatedData, "nonce": resourceNonce,
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	sign := func(t *testing.T, body []byte) string {
		t.Helper()
		message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, headerNonce, body)
		digest := sha256.Sum256([]byte(message))
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
		if err != nil {
			t.Fatal(err)
		}
		return base64.StdEncoding.EncodeToString(signature)
	}
	requestFor := func(body []byte, signature string) *http.Request {
		request := httptest.NewRequest(http.MethodPost, "/notify/indirect-merchant", bytes.NewReader(body))
		request.Header.Set(consts.DouyinPaySerial, serial)
		request.Header.Set(consts.DouyinPaySignature, signature)
		request.Header.Set(consts.DouyinPayTimestamp, timestamp)
		request.Header.Set(consts.DouyinPayNonce, headerNonce)
		return request
	}

	for _, state := range []string{"APPLYMENT_STATE_PASSED", "APPLYMENT_STATE_FUTURE"} {
		t.Run(state, func(t *testing.T) {
			body := bodyFor(t, state, false)
			envelope, content, err := ParseApplymentStatusNotify(context.Background(),
				callback.NewNotifyHandler(aesKey, verifier), requestFor(body, sign(t, body)))
			if err != nil {
				t.Fatal(err)
			}
			if envelope == nil || content == nil {
				t.Fatal("expected decrypted content and notification envelope")
			}
			expected := ApplymentStatusNotification{
				SpMchid: "6120230301343998", ApplymentID: "NTAM7638162829987562028",
				BusinessCode: "business_1", IndirectMerchantID: "2026020211234567890",
				ApplymentState: state, FailReason: "fixture reason",
			}
			if *content != expected {
				t.Fatalf("content = %#v, want %#v", *content, expected)
			}
			if envelope.ID != "notification_1" || envelope.EventType != event ||
				envelope.Resource.Plaintext != plaintext(state) {
				t.Fatalf("unexpected notification envelope: %#v", envelope)
			}
		})
	}

	for _, scenario := range []struct {
		name          string
		tamperBody    bool
		tamperSign    bool
		tamperCipher  bool
		key           string
		errorContains string
	}{
		{name: "tampered_body", tamperBody: true, key: wrongAESKey, errorContains: "invalid notify request:"},
		{name: "tampered_signature", tamperSign: true, key: wrongAESKey, errorContains: "invalid notify request:"},
		{name: "wrong_aes_key", key: wrongAESKey, errorContains: "decrypt request error: cipher: message authentication failed"},
		{name: "resigned_tampered_ciphertext", tamperCipher: true, key: aesKey, errorContains: "decrypt request error: cipher: message authentication failed"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			body := bodyFor(t, "APPLYMENT_STATE_PASSED", scenario.tamperCipher)
			signature := sign(t, body)
			if scenario.tamperBody {
				body = append(body, ' ')
			}
			if scenario.tamperSign {
				sigBytes, err := base64.StdEncoding.DecodeString(signature)
				if err != nil {
					t.Fatal(err)
				}
				sigBytes[0] ^= 1
				signature = base64.StdEncoding.EncodeToString(sigBytes)
			}
			if !scenario.tamperBody && !scenario.tamperSign {
				message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, headerNonce, body)
				if err := verifier.Verify(context.Background(), serial, message, signature); err != nil {
					t.Fatalf("GCM fixture must first pass real RSA verification: %v", err)
				}
			}
			envelope, content, err := ParseApplymentStatusNotify(context.Background(),
				callback.NewNotifyHandler(scenario.key, verifier), requestFor(body, signature))
			if err == nil || !strings.Contains(err.Error(), scenario.errorContains) {
				t.Fatalf("error = %v, want %q", err, scenario.errorContains)
			}
			if envelope != nil || content != nil {
				t.Fatal("failed notification must not expose decrypted content")
			}
		})
	}

	t.Run("nil_handler", func(t *testing.T) {
		envelope, content, err := ParseApplymentStatusNotify(context.Background(), nil, nil)
		if err == nil || err.Error() != "handler is required" || envelope != nil || content != nil {
			t.Fatalf("unexpected nil handler result: %v, %v, %v", envelope, content, err)
		}
	})
}
