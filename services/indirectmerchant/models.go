package indirectmerchant

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const maxUploadFileSize = 5 * 1024 * 1024

var businessCodePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// MerchantLocator identifies an indirect merchant by either the platform ID or
// the external merchant and payment institution IDs.
type MerchantLocator struct {
	IndirectMerchantID    string `json:"indirect_merchant_id,omitempty"`
	OutIndirectMerchantID string `json:"out_indirect_merchant_id,omitempty"`
	OutPaymentInstID      string `json:"out_payment_inst_id,omitempty"`
}

func (l MerchantLocator) Validate() error {
	hasPlatformID := l.IndirectMerchantID != ""
	hasExternalID := l.OutIndirectMerchantID != "" || l.OutPaymentInstID != ""
	if hasPlatformID == hasExternalID {
		return errors.New("specify exactly one merchant locator")
	}
	if hasExternalID && (l.OutIndirectMerchantID == "" || l.OutPaymentInstID == "") {
		return errors.New("out_indirect_merchant_id and out_payment_inst_id must be specified together")
	}
	return nil
}

type CreateApplyRequest struct {
	BusinessCode          string                 `json:"business_code"`
	DouyinpaySerial       string                 `json:"-"`
	OutIndirectMerchantID string                 `json:"out_indirect_merchant_id"`
	OutPaymentInstID      string                 `json:"out_payment_inst_id"`
	OutChannelSpID        string                 `json:"out_channel_sp_id,omitempty"`
	OutPaymentInstSubID   string                 `json:"out_payment_inst_sub_id,omitempty"`
	IndirectMerchantID    string                 `json:"indirect_merchant_id,omitempty"`
	SubjectInfo           map[string]interface{} `json:"subject_info,omitempty"`
	IdentificationInfo    map[string]interface{} `json:"identification_info,omitempty"`
	BusinessInfo          map[string]interface{} `json:"business_info,omitempty"`
	AdditionInfo          map[string]interface{} `json:"addition_info,omitempty"`
}

func (r CreateApplyRequest) Validate() error {
	if !businessCodePattern.MatchString(r.BusinessCode) {
		return errors.New("business_code must contain 1-64 lowercase letters, digits, or underscores")
	}
	if r.DouyinpaySerial == "" {
		return errors.New("douyinpay_serial is required")
	}
	if r.OutIndirectMerchantID == "" {
		return errors.New("out_indirect_merchant_id is required")
	}
	if r.OutPaymentInstID == "" {
		return errors.New("out_payment_inst_id is required")
	}
	return nil
}

type CreateApplyResponse struct {
	ApplymentID string `json:"applyment_id"`
}

type QueryApplyRequest struct {
	ApplymentID  string
	BusinessCode string
}

func (r QueryApplyRequest) Validate() error {
	return validateApplymentReference(r.ApplymentID, r.BusinessCode)
}

type QueryApplyResponse struct {
	BusinessCode       string                 `json:"business_code"`
	ApplymentID        string                 `json:"applyment_id"`
	IndirectMerchantID string                 `json:"indirect_merchant_id"`
	ApplymentState     string                 `json:"applyment_state"`
	AuditDetail        map[string]interface{} `json:"audit_detail"`
}

type CancelApplyRequest struct {
	ApplymentID  string `json:"applyment_id,omitempty"`
	BusinessCode string `json:"business_code,omitempty"`
}

func (r CancelApplyRequest) Validate() error {
	return validateApplymentReference(r.ApplymentID, r.BusinessCode)
}

type QueryMerchantRequest struct {
	MerchantLocator
	OutChannelSpID string
}

func (r QueryMerchantRequest) Validate() error {
	if err := r.MerchantLocator.Validate(); err != nil {
		return err
	}
	if r.IndirectMerchantID != "" && r.OutChannelSpID != "" {
		return errors.New("out_channel_sp_id is only supported with external merchant locator")
	}
	return nil
}

type QueryMerchantResponse struct {
	OutIndirectMerchantID string                 `json:"out_indirect_merchant_id"`
	OutPaymentInstID      string                 `json:"out_payment_inst_id"`
	ApplymentID           string                 `json:"applyment_id"`
	IndirectMerchantID    string                 `json:"indirect_merchant_id"`
	SubjectInfo           map[string]interface{} `json:"subject_info"`
	IdentificationInfo    map[string]interface{} `json:"identification_info"`
	BusinessInfo          map[string]interface{} `json:"business_info"`
}

type ModifyMerchantRequest struct {
	MerchantLocator
	MerchantShortName string `json:"merchant_short_name,omitempty"`
	ServicePhone      string `json:"service_phone,omitempty"`
}

func (r ModifyMerchantRequest) Validate() error {
	return r.MerchantLocator.Validate()
}

type BindAppIDRequest struct {
	MerchantLocator
	AppID string `json:"app_id"`
}

func (r BindAppIDRequest) Validate() error {
	if err := r.MerchantLocator.Validate(); err != nil {
		return err
	}
	if r.AppID == "" {
		return errors.New("app_id is required")
	}
	return nil
}

type BindAppIDResponse struct {
	AppInfo map[string]interface{} `json:"app_info"`
}

type QueryAppIDRequest struct {
	MerchantLocator
	AppID string
}

func (r QueryAppIDRequest) Validate() error {
	return r.MerchantLocator.Validate()
}

type QueryAppIDResponse struct {
	AppInfoList []map[string]interface{} `json:"app_info_list"`
}

type UploadRequest struct {
	FileName    string
	ContentType string
	FileContent []byte
}

func (r UploadRequest) Validate() error {
	if len(r.FileContent) == 0 {
		return errors.New("file content is required")
	}
	if len(r.FileContent) > maxUploadFileSize {
		return fmt.Errorf("file content must not exceed %d bytes", maxUploadFileSize)
	}
	if len(r.FileName) == 0 || len(r.FileName) > 128 {
		return errors.New("filename must contain 1-128 characters")
	}
	switch {
	case strings.HasSuffix(strings.ToLower(r.FileName), ".jpg"),
		strings.HasSuffix(strings.ToLower(r.FileName), ".jpeg"),
		strings.HasSuffix(strings.ToLower(r.FileName), ".bmp"),
		strings.HasSuffix(strings.ToLower(r.FileName), ".png"):
		return nil
	default:
		return errors.New("filename must use a jpg, jpeg, bmp, or png extension")
	}
}

type UploadResponse struct {
	MediaID string `json:"media_id"`
}

// ApplymentStatusNotification is the decrypted content of an indirect
// merchant application status notification.
type ApplymentStatusNotification struct {
	SpMchid            string `json:"sp_mchid"`
	ApplymentID        string `json:"applyment_id"`
	BusinessCode       string `json:"business_code"`
	IndirectMerchantID string `json:"indirect_merchant_id"`
	ApplymentState     string `json:"applyment_state"`
	FailReason         string `json:"fail_reason"`
}

func validateApplymentReference(applymentID, businessCode string) error {
	if (applymentID == "") == (businessCode == "") {
		return errors.New("specify exactly one of applyment_id or business_code")
	}
	return nil
}
