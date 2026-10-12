package client_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/douyinpay/douyinpay-go/client"
	"github.com/douyinpay/douyinpay-go/tools/auth"
	"github.com/douyinpay/douyinpay-go/tools/auth/signers"
	"github.com/douyinpay/douyinpay-go/tools/auth/verifiers"
	"github.com/douyinpay/douyinpay-go/tools/consts"
)

type multipartTestSigner struct {
	message string
}

func (s *multipartTestSigner) Sign(_ context.Context, message string) (*auth.SignatureResult, error) {
	s.message = message
	return &auth.SignatureResult{
		MchID:               "mchid",
		CertificateSerialNo: "merchant-serial",
		Signature:           "signature",
	}, nil
}

func (s *multipartTestSigner) Algorithm() string {
	return "TEST"
}

type multipartTestVerifier struct {
	called bool
}

func (v *multipartTestVerifier) Verify(_ context.Context, _, _, _ string) error {
	v.called = true
	return nil
}

func (v *multipartTestVerifier) GetSerial(_ context.Context) (string, error) {
	return "platform-serial", nil
}

func TestRequestMultipartSignsMetaAndSendsMultipartBody(t *testing.T) {
	merchantKey := multipartRSAKey(t)
	platformKey := multipartRSAKey(t)
	for _, tc := range []struct {
		name, contentType, expectedType string
		tamper                          bool
	}{
		{"explicit file MIME", "image/jpeg", "image/jpeg", false},
		{"default file MIME", "", "application/octet-stream", false},
		{"escaped filename", "image/png", "image/png", false},
		{"tampered response", "image/png", "image/png", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var wireBody []byte
			var wireHeader http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wireHeader = r.Header.Clone()
				var err error
				wireBody, err = ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request: %v", err)
				}
				const signedBody = `{"media_id":"media_1"}`
				timestamp := strconv.FormatInt(time.Now().Unix(), 10)
				w.Header().Set(consts.DouyinPayTimestamp, timestamp)
				w.Header().Set(consts.DouyinPayNonce, "nonce")
				w.Header().Set(consts.DouyinPaySerial, "platform-serial")
				w.Header().Set(consts.DouyinPaySignature,
					multipartRSASign(t, platformKey, timestamp+"\nnonce\n"+signedBody+"\n"))
				responseBody := signedBody
				if tc.tamper {
					responseBody = `{"media_id":"tampered"}`
				}
				_, _ = io.WriteString(w, responseBody)
			}))
			defer server.Close()

			entity := validMultipartEntity(server.URL)
			entity.QueryParams = map[string][]string{"kind": {"license"}}
			entity.Header.Set("Content-Type", "application/json")
			entity.ContentType = tc.contentType
			if tc.name == "escaped filename" {
				entity.FileName = "license\";\r\nX-Injected: yes.png"
			}
			entity.FileContent = []byte{0, 255, '\r', '\n', 1, 2}
			entity.Fields["note"] = "ordinary field"
			c := newMultipartTestClient(t, &signers.SHA256WithRSASigner{
				MchID: "mchid", CertificateSerialNo: "merchant-serial", PrivateKey: merchantKey,
			}, &verifiers.SHA256WithRSAVerifier{
				SerialNumber: "platform-serial", PublicKey: &platformKey.PublicKey,
			})
			result, err := c.RequestMultipart(context.Background(), entity)
			if result == nil || result.Response == nil {
				t.Fatalf("missing HTTP response: %v", err)
			}
			defer result.Response.Body.Close()
			if tc.tamper {
				if err == nil || !strings.Contains(err.Error(), "validate verify fail") {
					t.Errorf("tampered response must fail RSA verification: %v", err)
				}
			} else if err != nil {
				t.Errorf("valid RSA response rejected: %v", err)
			}
			responseBody, readErr := ioutil.ReadAll(result.Response.Body)
			if readErr != nil || (!tc.tamper && string(responseBody) != `{"media_id":"media_1"}`) {
				t.Errorf("response body not preserved: %q, %v", responseBody, readErr)
			}

			authHeader := wireHeader.Get(consts.Authorization)
			params := map[string]string{}
			for _, match := range regexp.MustCompile(`(\w+)="([^"]*)"`).FindAllStringSubmatch(authHeader, -1) {
				params[match[1]] = match[2]
			}
			if !strings.HasPrefix(authHeader, "DouyinPay-RSA ") || params["mchid"] != "mchid" ||
				params["serial_no"] != "merchant-serial" {
				t.Errorf("incorrect RSA Authorization: %s", authHeader)
			}
			prefix := fmt.Sprintf("POST\n/v1/merchant/upload?kind=license\n%s\n%s\n",
				params["timestamp"], params["nonce_str"])
			if err := multipartRSAVerify(&merchantKey.PublicKey, prefix+entity.SignBody+"\n", params["signature"]); err != nil {
				t.Errorf("meta-only request signature rejected: %v", err)
			}
			for _, unsignedBody := range []string{string(wireBody), entity.SignBody + " ", string(entity.FileContent)} {
				if multipartRSAVerify(&merchantKey.PublicKey, prefix+unsignedBody+"\n", params["signature"]) == nil {
					t.Error("request signature unexpectedly covers a non-meta body")
				}
			}
			mediaType, params, err := mime.ParseMediaType(wireHeader.Get(consts.ContentType))
			if err != nil || mediaType != consts.MultipartFormData || params["boundary"] == "" {
				t.Fatalf("invalid multipart boundary: %s, %v", wireHeader.Get(consts.ContentType), err)
			}
			reader := multipart.NewReader(bytes.NewReader(wireBody), params["boundary"])
			count := 0
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("parse multipart body: %v", err)
				}
				content, err := ioutil.ReadAll(part)
				if err != nil {
					t.Fatal(err)
				}
				count++
				switch part.FormName() {
				case "meta":
					if part.Header.Get("Content-Type") != "application/json" || string(content) != entity.SignBody {
						t.Errorf("meta MIME/body: %q / %q", part.Header.Get("Content-Type"), content)
					}
				case "note":
					if part.Header.Get("Content-Type") != "" || string(content) != "ordinary field" {
						t.Errorf("ordinary field defaults changed: %v / %q", part.Header, content)
					}
				case "file":
					_, disposition, parseErr := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
					if parseErr != nil || part.Header.Get("X-Injected") != "" ||
						part.Header.Get("Content-Type") != tc.expectedType || disposition["filename"] != entity.FileName ||
						!bytes.Equal(content, entity.FileContent) {
						t.Errorf("file MIME/name/body: %v / %q", part.Header, content)
					}
				default:
					t.Errorf("unexpected multipart field: %s", part.FormName())
				}
			}
			if count != 3 {
				t.Errorf("expected 3 parts, got %d", count)
			}
		})
	}
}

func TestRequestMultipartContentTypes(t *testing.T) {
	for _, tc := range []struct {
		name, fileType, fieldType string
		wantError                 bool
	}{
		{"unconfigured fields retain defaults", "", "", false},
		{"configured field MIME", "", "text/plain; charset=utf-8", false},
		{"invalid field MIME", "", "application/json\r\nX-Injected: yes", true},
		{"invalid file MIME", "image/png\r\nX-Injected: yes", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				reader, err := r.MultipartReader()
				if err != nil {
					t.Errorf("multipart reader: %v", err)
					return
				}
				for {
					part, err := reader.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Errorf("read part: %v", err)
						return
					}
					if part.FormName() == "meta" && part.Header.Get("Content-Type") != tc.fieldType {
						t.Errorf("field MIME: got %q, want %q", part.Header.Get("Content-Type"), tc.fieldType)
					}
					_, _ = ioutil.ReadAll(part)
				}
				setValidResponseHeaders(w)
				_, _ = io.WriteString(w, `{}`)
			}))
			defer server.Close()
			entity := validMultipartEntity(server.URL)
			entity.FieldContentTypes = map[string]string{"meta": tc.fieldType}
			entity.ContentType = tc.fileType
			result, err := newMultipartTestClient(t, new(multipartTestSigner), new(multipartTestVerifier)).
				RequestMultipart(context.Background(), entity)
			if result != nil && result.Response != nil {
				defer result.Response.Body.Close()
			}
			if tc.wantError {
				if err == nil || !strings.Contains(err.Error(), "content type") || called {
					t.Errorf("invalid MIME must be rejected before HTTP: err=%v, called=%v", err, called)
				}
			} else if err != nil || !called {
				t.Errorf("valid MIME request: err=%v, called=%v", err, called)
			}
		})
	}
}

func multipartRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func multipartRSASign(t *testing.T, key *rsa.PrivateKey, message string) string {
	t.Helper()
	hash := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(signature)
}

func multipartRSAVerify(key *rsa.PublicKey, message, signature string) error {
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], decoded)
}

func TestRequestMultipartReturnsAPIErrorForNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"PARAM_ERROR","message":"invalid file"}`))
	}))
	defer server.Close()

	result, err := newMultipartTestClient(t, new(multipartTestSigner), new(multipartTestVerifier)).RequestMultipart(
		context.Background(), validMultipartEntity(server.URL))

	assert.Equal(t, true, result != nil)
	apiErr, ok := err.(*client.APIError)
	assert.Equal(t, true, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "PARAM_ERROR", apiErr.Code)
}

func newMultipartTestClient(t *testing.T, signer auth.Signer, verifier auth.Verifier) *client.Client {
	t.Helper()
	c, err := client.NewClientWithDialSettings(context.Background(), &client.DialSettings{
		Signer: signer, Verifier: verifier,
	})
	assert.Equal(t, nil, err)
	return c
}

func validMultipartEntity(serverURL string) *client.MultipartRequestEntity {
	return &client.MultipartRequestEntity{
		Method:      http.MethodPost,
		RequestPath: serverURL + "/v1/merchant/upload",
		Header:      http.Header{},
		SignBody:    `{"filename":"license.jpg","sha256":"abc"}`,
		Fields: map[string]string{
			"meta": `{"filename":"license.jpg","sha256":"abc"}`,
		},
		FieldContentTypes: map[string]string{"meta": "application/json"},
		FileField:         "file",
		FileName:          "license.jpg",
		FileContent:       []byte("image-content"),
	}
}

func setValidResponseHeaders(w http.ResponseWriter) {
	w.Header().Set(consts.DouyinPaySerial, "platform-serial")
	w.Header().Set(consts.DouyinPaySignature, "signature")
	w.Header().Set(consts.DouyinPayTimestamp, strconv.FormatInt(time.Now().Unix(), 10))
	w.Header().Set(consts.DouyinPayNonce, "nonce")
}
