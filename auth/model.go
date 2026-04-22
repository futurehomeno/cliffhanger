package auth

import (
	"github.com/futurehomeno/cliffhanger/hub"
)

type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        any    `json:"scope"`
}

type OAuth2RefreshProxyRequest struct {
	RefreshToken string `json:"refreshToken"`
	PartnerCode  string `json:"partnerCode"`
}

type OAuth2AuthCodeProxyRequest struct {
	AuthCode    string `json:"code"`
	PartnerCode string `json:"partnerCode"`
}

type OAuth2PasswordProxyRequest struct {
	PartnerCode string `json:"partnerCode"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

func ProxyURL(environment hub.Environment) string {
	if environment == hub.EnvBeta {
		return "https://partners-beta.futurehome.io"
	}

	return "https://partners.futurehome.io"
}

func ProxyCallbackURL(environment hub.Environment) string {
	if environment == hub.EnvBeta {
		return "https://app-static-beta.futurehome.io/playground_oauth_callback"
	}

	return "https://app-static.futurehome.io/playground_oauth_callback"
}
