package client

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/douyinpay/douyinpay-go/tools/auth"
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
	const meta = `{"filename":"license.jpg","sha256":"abc"}`
	signer := new(multipartTestSigner)
	verifier := new(multipartTestVerifier)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, consts.MultipartFormData, r.Header.Get(consts.ContentType)[:len(consts.MultipartFormData)])
		assert.Equal(t, nil, r.ParseMultipartForm(1024*1024))
		assert.Equal(t, meta, r.FormValue("meta"))

		file, _, err := r.FormFile("file")
		assert.Equal(t, nil, err)
		defer file.Close()
		content, err := ioutil.ReadAll(file)
		assert.Equal(t, nil, err)
		assert.Equal(t, "image-content", string(content))

		setValidResponseHeaders(w)
		_, _ = w.Write([]byte(`{"media_id":"media_1"}`))
	}))
	defer server.Close()

	c := newMultipartTestClient(t, signer, verifier)
	result, err := c.RequestMultipart(context.Background(), &MultipartRequestEntity{
		Method:      http.MethodPost,
		RequestPath: server.URL + "/v1/merchant/upload",
		Header:      http.Header{},
		SignBody:    meta,
		Fields:      map[string]string{"meta": meta},
		FileField:   "file",
		FileName:    "license.jpg",
		FileContent: []byte("image-content"),
	})

	assert.Equal(t, nil, err)
	assert.Equal(t, meta, signBody(signer.message))
	assert.Equal(t, true, verifier.called)
	assert.Equal(t, consts.MultipartFormData, result.Request.Header.Get(consts.ContentType)[:len(consts.MultipartFormData)])
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
	apiErr, ok := err.(*APIError)
	assert.Equal(t, true, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "PARAM_ERROR", apiErr.Code)
}

func newMultipartTestClient(t *testing.T, signer auth.Signer, verifier auth.Verifier) *Client {
	t.Helper()
	c, err := NewClientWithDialSettings(context.Background(), &DialSettings{
		Signer: signer, Verifier: verifier,
	})
	assert.Equal(t, nil, err)
	return c
}

func validMultipartEntity(serverURL string) *MultipartRequestEntity {
	return &MultipartRequestEntity{
		Method:      http.MethodPost,
		RequestPath: serverURL + "/v1/merchant/upload",
		Header:      http.Header{},
		SignBody:    `{"filename":"license.jpg","sha256":"abc"}`,
		Fields: map[string]string{
			"meta": `{"filename":"license.jpg","sha256":"abc"}`,
		},
		FileField:   "file",
		FileName:    "license.jpg",
		FileContent: []byte("image-content"),
	}
}

func setValidResponseHeaders(w http.ResponseWriter) {
	w.Header().Set(consts.DouyinPaySerial, "platform-serial")
	w.Header().Set(consts.DouyinPaySignature, "signature")
	w.Header().Set(consts.DouyinPayTimestamp, strconv.FormatInt(time.Now().Unix(), 10))
	w.Header().Set(consts.DouyinPayNonce, "nonce")
}

func signBody(message string) string {
	return strings.Split(message, "\n")[4]
}
