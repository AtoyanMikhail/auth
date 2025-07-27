package models

type GetTokensReq struct {
	GUID string `json:"guid"`
}

type GetTokensRes struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokensReq struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokensRes struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type MeReq struct {
	GUID string `json:"guid"`
}