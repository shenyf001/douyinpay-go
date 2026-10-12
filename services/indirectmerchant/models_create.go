package indirectmerchant

// SubjectInfo 表示入驻申请的主体资料，不用于查询响应。
type SubjectInfo struct {
	// 字段含义：主体类型。
	// 格式规则：字符串。
	// 业务规则：【主体类型】主体类型需与营业执照/登记证书上一致。枚举：SUBJECT_TYPE_INDIVIDUAL：营业执照上的主体类型一般为个体户、个体工商户、个体经营；SUBJECT_TYPE_ENTERPRISE：营业执照上的主体类型一般为有限公司、有限责任公司；SUBJECT_TYPE_GOVERNMENT：包括各级、各类政府机关，如机关党委、税务、民政、人社、工商、商务、市监等；SUBJECT_TYPE_INSTITUTIONS：包括国内各类事业单位，如：医疗、教育、学校等单位；SUBJECT_TYPE_OTHERS：包括社会团体、民办非企业、基金会、基层群众性自治组织、农村集体经济组织等组织；SUBJECT_TYPE_MICRO：小微商户。
	// 示例：SUBJECT_TYPE_ENTERPRISE
	SubjectType string `json:"subject_type,omitempty"`
	// 字段含义：主体证件信息。
	// 格式规则：对象。
	// 业务规则：【主体证件信息】主体为企业/个体户/政府机关/事业单位/其他组织时，请填写主体证件信息；小微商户不需要填写主体证件信息，只需要提交“经营者/法人身份证件”信息。
	// 示例：{"register_address":"湖南省长沙市xx区xx路xx号","period_begin":"2019-08-01","period_end":"2019-08-02"}
	CertificateInfo *CertificateInfo `json:"certificate_info,omitempty"`
}

// CertificateInfo 表示入驻申请的营业执照/登记证书信息。
type CertificateInfo struct {
	// 字段含义：证件编号。
	// 格式规则：字符串。
	// 业务规则：【证件编号】1、请填写营业执照/登记证书上的统一社会信用代码。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	CertNo string `json:"cert_no,omitempty"`
	// 字段含义：商户名称。
	// 格式规则：字符串，长度为2-128个字符；不能仅含特殊字符和数字；仅支持utf-8格式。
	// 业务规则：【商户名称】对应其证件的主体名称。1、长度为2-128个字符；2、不能仅含特殊字符和数字；3、仅支持utf-8格式；4、小微商户使用法人名称。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	MerchantName string `json:"merchant_name,omitempty"`
	// 字段含义：证件图片。
	// 格式规则：字符串，填写通过图片上传接口预先上传图片生成好的MediaID，可上传1张图片。
	// 业务规则：【证件图片】1、照片应正面拍摄、清晰、四角完整、无反光或遮挡；不得翻拍、截图、镜像、PS；2、上传彩色照片、彩色扫描件，复印件需加盖公章鲜章；3、水印仅限于抖音支付业务相关；4、可上传1张图片，请填写通过图片上传接口预先上传图片生成好的MediaID。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	CertCopy string `json:"cert_copy,omitempty"`
	// 字段含义：注册地址。
	// 格式规则：字符串，长度为4-128个字符；前后不能有空格、制表符、换行符；不能仅含数字、特殊字符；仅能填写数字、英文字母、汉字及特殊字符；仅支持utf-8格式。
	// 业务规则：【注册地址】营业执照/登记证书的注册地址。1、长度为4-128个字符；2、前后不能有空格、制表符、换行符；3、不能仅含数字、特殊字符；4、仅能填写数字、英文字母、汉字及特殊字符；5、仅支持utf-8格式。
	// 示例：湖南省长沙市xx区xx路xx号
	RegisterAddress string `json:"register_address,omitempty"`
	// 字段含义：经营者/法人姓名。
	// 格式规则：字符串，长度为2-100个字符；前后不能有空格、制表符、换行符；不能仅含特殊字符；仅能填写数字、英文字母、汉字及特殊字符。
	// 业务规则：【经营者/法人姓名】请填写营业执照的经营者/法定代表人姓名。1、长度为2-100个字符；2、前后不能有空格、制表符、换行符；3、不能仅含特殊字符；4、仅能填写数字、英文字母、汉字及特殊字符。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	LegalPersonName string `json:"legal_person_name,omitempty"`
	// 字段含义：有效期限开始日期。
	// 格式规则：字符串，日期格式应满足合法的YYYY-MM-DD格式。
	// 业务规则：【有效期限开始日期】请填写营业执照/登记证书的有效期限开始时间。1、日期格式应满足合法的YYYY-MM-DD格式；3、开始时间不能大于等于当前日期。
	// 示例：2019-08-01
	PeriodBegin string `json:"period_begin,omitempty"`
	// 字段含义：有效期限结束日期。
	// 格式规则：字符串，日期格式应满足合法的YYYY-MM-DD格式或长期。
	// 业务规则：【有效期限结束日期】请填写营业执照/登记证书的有效期限结束时间。1、日期格式应满足合法的YYYY-MM-DD格式或长期；2、结束时间需大于开始时间；3、若证件有效期为长期，请填写：长期。
	// 示例：2019-08-02
	PeriodEnd string `json:"period_end,omitempty"`
}

// IdentificationInfo 表示入驻申请的经营者/法人身份证件信息。
type IdentificationInfo struct {
	// 字段含义：证件类型。
	// 格式规则：字符串。
	// 业务规则：【证件类型】1、个体户/企业/事业单位/社会组织：可选择任一证件类型，政府机关、小微商户仅支持中国大陆居民-身份证类型。枚举：IDENTIFICATION_TYPE_IDCARD：中国大陆居民-身份证；IDENTIFICATION_TYPE_OVERSEA_PASSPORT：其他国家或地区居民-护照；IDENTIFICATION_TYPE_HONGKONG_PASSPORT：中国香港居民-来往内地通行证；IDENTIFICATION_TYPE_MACAO_PASSPORT：中国澳门居民-来往内地通行证；IDENTIFICATION_TYPE_TAIWAN_PASSPORT：中国台湾居民-来往大陆通行证；IDENTIFICATION_TYPE_FOREIGN_RESIDENT：外国人居留证；IDENTIFICATION_TYPE_HONGKONG_MACAO_RESIDENT：港澳居民证；IDENTIFICATION_TYPE_TAIWAN_RESIDENT：台湾居民证。
	// 示例：IDENTIFICATION_TYPE_IDCARD
	IdentificationType string `json:"identification_type,omitempty"`
	// 字段含义：证件持有人姓名。
	// 格式规则：字符串，明文长度为2-100个字符；前后不能有空格、制表符、换行符；不能仅含数字、特殊字符；仅能填写数字、英文字母、汉字及特殊字符。
	// 业务规则：【证件持有人姓名】1、请填写经营者/法定代表人对应证件的姓名；2、长度为2-100个字符；3、前后不能有空格、制表符、换行符；4、不能仅含数字、特殊字符；5、仅能填写数字、英文字母、汉字及特殊字符；6、该字段需进行加密处理，详见 https://partner.douyinpay.com/wiki/682561fbd3e93f04ff85a04f/682563b39faab1051f8f5abc。调用方传入明文，由 SDK 加密。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	IdentificationName string `json:"identification_name,omitempty"`
	// 字段含义：证件持有人证件号码。
	// 格式规则：字符串，调用方传入明文。
	// 业务规则：【证件持有人证件号码】1、请填写经营者/法定代表人对应证件的号码；2、该字段需进行加密处理，详见 https://partner.douyinpay.com/wiki/682561fbd3e93f04ff85a04f/682563b39faab1051f8f5abc。由 SDK 加密。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	IdentificationNumber string `json:"identification_number,omitempty"`
	// 字段含义：证件持有人有效期开始时间。
	// 格式规则：字符串，日期格式应满足合法的YYYY-MM-DD格式。
	// 业务规则：【证件持有人有效期开始时间】1、日期格式应满足合法的YYYY-MM-DD格式；2、开始时间不能小于1900-01-01；3、开始时间不能大于等于当前日期。
	// 示例：2019-08-01
	PeriodBegin string `json:"period_begin,omitempty"`
	// 字段含义：证件持有人有效期结束时间。
	// 格式规则：字符串，日期格式应满足合法的YYYY-MM-DD格式或长期。
	// 业务规则：【证件持有人有效期结束时间】1、日期格式应满足合法的YYYY-MM-DD格式或长期；2、结束时间需大于开始时间；3、若证件有效期为长期，请填写：长期。
	// 示例：2019-08-02
	PeriodEnd string `json:"period_end,omitempty"`
	// 字段含义：证件正面照片。
	// 格式规则：字符串，填写通过图片上传接口预先上传图片生成好的MediaID，可上传1张图片。
	// 业务规则：【证件正面照片】1、证件类型不为“身份证”时，上传证件正面照片；若法人证件类型为身份证，请上传人像面照片；2、正面拍摄、清晰、四角完整、无反光或遮挡；不得翻拍、截图、镜像、PS；3、请上传彩色照片or彩色扫描件，复印件需加盖公章鲜章，可添加“抖音支付”相关水印（如抖音支付认证）；4、可上传1张图片，请填写通过图片上传接口预先上传图片生成好的MediaID。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	IdentificationFrontCopy string `json:"identification_front_copy,omitempty"`
	// 字段含义：证件背面照片。
	// 格式规则：字符串，填写通过图片上传接口预先上传图片生成好的MediaID，可上传1张图片。
	// 业务规则：【证件背面照片】1、若法人证件类型为身份证，请上传国徽面照片；2、正面拍摄、清晰、四角完整、无反光或遮挡；不得翻拍、截图、镜像、PS；3、请上传彩色照片or彩色扫描件，复印件需加盖公章鲜章，可添加“抖音支付”相关水印（如抖音支付认证）；4、可上传1张图片，请填写通过图片上传接口预先上传图片生成好的MediaID。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	IdentificationBackCopy string `json:"identification_back_copy,omitempty"`
}

// BusinessInfo 表示入驻申请的经营信息。
type BusinessInfo struct {
	// 字段含义：商户简称。
	// 格式规则：字符串，长度为1-32个字符；前后不能有空格、制表符、换行符；不能仅含特殊字符；仅能填写数字、英文字母、汉字及特殊字符；仅支持utf-8格式。
	// 业务规则：【商户简称】在支付完成页向买家展示，需与抖音支付经营类目相关。1、请输入1-32个字符；2、前后不能有空格、制表符、换行符；3、不能仅含特殊字符；4、仅能填写数字、英文字母、汉字及特殊字符；5、仅支持utf-8格式。
	// 示例：个体户张三
	MerchantShortName string `json:"merchant_short_name,omitempty"`
	// 字段含义：客服电话。
	// 格式规则：字符串；前后不能有空格、制表符、换行符；需满足以下任一条件：11位数字的手机号码；5-20位数字、连字符“-”、加号“+”。
	// 业务规则：【客服电话】将在交易记录中向买家展示，请确保电话畅通以便平台回拨确认。1、前后不能有空格、制表符、换行符；2、需满足以下任一条件：11位数字的手机号码；5-20位数字、连字符“-”、加号“+”。
	// 示例：18042002458
	ServicePhone string `json:"service_phone,omitempty"`
	// 字段含义：所属行业。
	// 格式规则：字符串。
	// 业务规则：填写间连商户在清算机构侧登记的行业MCC编码。
	// 示例：7300
	MCC string `json:"mcc,omitempty"`
	// 字段含义：特殊资质图片。
	// 格式规则：数组，使用字符串列表表示图片，图片大小不超过10M，支持1-5张图片。
	// 业务规则：【特殊资质图片】图片大小不超过10M，支持1-5张图片。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	SpecialIndustryPics []string `json:"special_industry_pics,omitempty"`
}

// AdditionInfo 表示入驻申请的补充信息。
type AdditionInfo struct {
	// 字段含义：补充材料。
	// 格式规则：数组，使用字符串列表表示图片，图片大小不超过10M，最多不超过5张图片。
	// 业务规则：【补充材料】补充图片内容，图片大小不超过10M，最多不超过5张图片。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	AdditionPics []string `json:"addition_pics,omitempty"`
	// 字段含义：补充说明。
	// 格式规则：字符串，文案不超过100字。
	// 业务规则：【补充说明】补充说明，文案不超过100字。
	// 示例：
	// TODO: 官方文档未提供标准示例值，暂留空。
	AdditionRemark string `json:"addition_remark,omitempty"`
}
