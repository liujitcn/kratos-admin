package utils

import (
	_http "github.com/liujitcn/go-utils/http"
)

// PhoneNumber 表示微信手机号授权接口返回结果
type PhoneNumber struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
		Watermark       struct {
			Timestamp int    `json:"timestamp"`
			AppID     string `json:"appid"`
		} `json:"watermark"`
	} `json:"phone_info"`
}

// GetPhoneNumber 调用微信接口换取手机号信息
func GetPhoneNumber(accessToken, code string) (*PhoneNumber, error) {
	// 微信接口要求请求体为 JSON，对外只传授权码即可
	query := make(map[string]string)
	query["access_token"] = accessToken

	body := make(map[string]string)
	body["code"] = code
	var res PhoneNumber
	err := _http.Post(
		"https://api.weixin.qq.com/wxa/business/getuserphonenumber",
		&res,
		_http.WithQueries(query),
		_http.WithJSONBody(body),
	)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
