// Package identidade e o adaptador do login (o Cognito) para a recuperacao do MFA (spec mfa-opcional, R27 e R32). Faz SO duas coisas:
// achar a pessoa pelo e-mail (AdminGetUser) e desligar o MFA dela (AdminSetUserMFAPreference). Nunca cria sessao, nunca devolve token
// e nunca altera a senha.
package identidade

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"financas/domain"
)

// API e o que se usa do Cognito; os testes trocam por um falso.
type API interface {
	AdminGetUser(context.Context, *cognitoidentityprovider.AdminGetUserInput, ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminGetUserOutput, error)
	AdminSetUserMFAPreference(context.Context, *cognitoidentityprovider.AdminSetUserMFAPreferenceInput, ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminSetUserMFAPreferenceOutput, error)
}

// Cognito implementa service.Identidade sobre o pool de login.
type Cognito struct {
	api    API
	poolID string
}

func NovoCognito(api API, poolID string) *Cognito { return &Cognito{api: api, poolID: poolID} }

const mfaTOTP = "SOFTWARE_TOKEN_MFA"

// BuscarPessoa acha a pessoa pelo e-mail (o pool usa o e-mail como nome de usuario). achou=false se nao ha conta.
func (c *Cognito) BuscarPessoa(ctx context.Context, email string) (domain.PessoaMFA, bool, error) {
	out, err := c.api.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{UserPoolId: aws.String(c.poolID), Username: aws.String(email)})
	if err != nil {
		var naoAchou *types.UserNotFoundException
		if errors.As(err, &naoAchou) {
			return domain.PessoaMFA{}, false, nil
		}
		return domain.PessoaMFA{}, false, err
	}
	p := domain.PessoaMFA{Habilitada: out.Enabled}
	for _, a := range out.UserAttributes {
		if aws.ToString(a.Name) == "sub" {
			p.Sub = aws.ToString(a.Value)
		}
	}
	if p.Sub == "" { // sem o atributo sub, o Username do pool (que e o sub quando o e-mail e o nome de usuario)
		p.Sub = aws.ToString(out.Username)
	}
	for _, m := range out.UserMFASettingList {
		if m == mfaTOTP {
			p.TemMFA = true
		}
	}
	return p, true, nil
}

// DesligarMFA desliga o aplicativo autenticador da pessoa. E a UNICA escrita no Cognito.
func (c *Cognito) DesligarMFA(ctx context.Context, sub string) error {
	_, err := c.api.AdminSetUserMFAPreference(ctx, &cognitoidentityprovider.AdminSetUserMFAPreferenceInput{
		UserPoolId:               aws.String(c.poolID),
		Username:                 aws.String(sub),
		SoftwareTokenMfaSettings: &types.SoftwareTokenMfaSettingsType{Enabled: false, PreferredMfa: false},
	})
	return err
}
