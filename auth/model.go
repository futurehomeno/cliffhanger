package auth

import (
	"github.com/futurehomeno/cliffhanger/hub"
)

// OAuth2TokenResponse is an object representing credentials for the app to log into a third-party service.
type OAuth2TokenResponse struct {
	AccessToken  string      `json:"access_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int64       `json:"expires_in"`
	RefreshToken string      `json:"refresh_token"`
	Scope        interface{} `json:"scope"`
}

// OAuth2RefreshProxyRequest is an object representing request to partners API to exchange refresh token for access token.
type OAuth2RefreshProxyRequest struct {
	RefreshToken string `json:"refreshToken"`
	PartnerCode  string `json:"partnerCode"`
}

// OAuth2AuthCodeProxyRequest is an object representing request to partners API to exchange authorization code for access token.
type OAuth2AuthCodeProxyRequest struct {
	AuthCode    string `json:"code"`
	PartnerCode string `json:"partnerCode"`
}

// OAuth2PasswordProxyRequest is an object representing request to partners API to exchange login and password for access token.
type OAuth2PasswordProxyRequest struct {
	PartnerCode string `json:"partnerCode"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

// ProxyURL is a helper method returning proxy url for the given environment.
func ProxyURL(environment hub.Environment) string {
	if environment == hub.EnvBeta {
		return "https://partners-beta.futurehome.io"
	}

	return "https://partners.futurehome.io"
}

// ProxyCallbackURL is a helper method returning proxy callback url for the given environment.
func ProxyCallbackURL(environment hub.Environment) string {
	if environment == hub.EnvBeta {
		return "https://app-static-beta.futurehome.io/playground_oauth_callback"
	}

	return "https://app-static.futurehome.io/playground_oauth_callback"
}
