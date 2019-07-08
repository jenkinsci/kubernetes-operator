package client

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type userTokenResponseData struct {
	Name  string `json:"tokenName"`
	UUID  string `json:"tokenUuid"`
	Value string `json:"tokenValue"`
}

type userTokenResponse struct {
	Status string                `json:"status"`
	Data   userTokenResponseData `json:"data"`
}

// UserToken defines user token for Jenkins API communication
type UserToken struct {
	raw  *userTokenResponse
	base string
}

// GetToken returns user token
func (token *UserToken) GetToken() string {
	return token.raw.Data.Value
}

func (jenkins *jenkins) GenerateToken(userName, tokenName string) (*UserToken, error) {
	token := &UserToken{raw: new(userTokenResponse),
		base: fmt.Sprintf("/user/%s/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken", userName)}
	endpoint := token.base
	data := map[string]string{"newTokenName": tokenName}
	r, err := jenkins.Requester.Post(endpoint, nil, token.raw, data)

	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate API token")
	}

	if r.StatusCode == http.StatusOK {
		if token.raw.Status == "ok" {
			return token, nil
		}

		return nil, errors.New(token.raw.Status)
	}

	return nil, errors.Errorf("couldn't generate API token: %d", r.StatusCode)
}
