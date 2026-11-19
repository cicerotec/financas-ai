package identidade

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// M-T26 (spec mfa-opcional): o adaptador SO acha a pessoa e desliga o MFA; nao cria sessao nem mexe na senha.

type apiFalsa struct {
	get     func(*cognitoidentityprovider.AdminGetUserInput) (*cognitoidentityprovider.AdminGetUserOutput, error)
	gets    []*cognitoidentityprovider.AdminGetUserInput
	desliga []*cognitoidentityprovider.AdminSetUserMFAPreferenceInput
	errDesl error
}

func (a *apiFalsa) AdminGetUser(_ context.Context, in *cognitoidentityprovider.AdminGetUserInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminGetUserOutput, error) {
	a.gets = append(a.gets, in)
	return a.get(in)
}

func (a *apiFalsa) AdminSetUserMFAPreference(_ context.Context, in *cognitoidentityprovider.AdminSetUserMFAPreferenceInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminSetUserMFAPreferenceOutput, error) {
	a.desliga = append(a.desliga, in)
	return &cognitoidentityprovider.AdminSetUserMFAPreferenceOutput{}, a.errDesl
}

func TestBuscarPessoaAchaPeloEmailComSubEMFA(t *testing.T) {
	api := &apiFalsa{get: func(*cognitoidentityprovider.AdminGetUserInput) (*cognitoidentityprovider.AdminGetUserOutput, error) {
		return &cognitoidentityprovider.AdminGetUserOutput{
			Username: aws.String("nome-de-usuario"), Enabled: true,
			UserAttributes:     []types.AttributeType{{Name: aws.String("email"), Value: aws.String("x")}, {Name: aws.String("sub"), Value: aws.String("sub-1")}},
			UserMFASettingList: []string{"SOFTWARE_TOKEN_MFA"},
		}, nil
	}}
	p, achou, err := NovoCognito(api, "pool-1").BuscarPessoa(context.Background(), "ana@exemplo.com")
	if err != nil || !achou || p.Sub != "sub-1" || !p.Habilitada || !p.TemMFA {
		t.Fatalf("BuscarPessoa = %+v %v %v", p, achou, err)
	}
	if len(api.gets) != 1 || aws.ToString(api.gets[0].Username) != "ana@exemplo.com" || aws.ToString(api.gets[0].UserPoolId) != "pool-1" {
		t.Errorf("a chamada ao Cognito saiu errada: %+v", api.gets)
	}
}

func TestBuscarPessoaSemMFAEDesabilitada(t *testing.T) {
	api := &apiFalsa{get: func(*cognitoidentityprovider.AdminGetUserInput) (*cognitoidentityprovider.AdminGetUserOutput, error) {
		return &cognitoidentityprovider.AdminGetUserOutput{Username: aws.String("so-username"), Enabled: false}, nil
	}}
	p, achou, err := NovoCognito(api, "p").BuscarPessoa(context.Background(), "a@b.com")
	if err != nil || !achou || p.Sub != "so-username" || p.Habilitada || p.TemMFA {
		t.Errorf("BuscarPessoa = %+v %v %v (sem o atributo sub, vale o Username)", p, achou, err)
	}
}

func TestBuscarPessoaInexistenteNaoEErro(t *testing.T) {
	api := &apiFalsa{get: func(*cognitoidentityprovider.AdminGetUserInput) (*cognitoidentityprovider.AdminGetUserOutput, error) {
		return nil, &types.UserNotFoundException{Message: aws.String("User does not exist.")}
	}}
	if _, achou, err := NovoCognito(api, "p").BuscarPessoa(context.Background(), "a@b.com"); achou || err != nil {
		t.Errorf("e-mail sem conta: achou=%v err=%v (esperado false, nil)", achou, err)
	}
	api.get = func(*cognitoidentityprovider.AdminGetUserInput) (*cognitoidentityprovider.AdminGetUserOutput, error) {
		return nil, errors.New("cognito fora do ar")
	}
	if _, _, err := NovoCognito(api, "p").BuscarPessoa(context.Background(), "a@b.com"); err == nil {
		t.Error("falha do servico precisa voltar como erro")
	}
}

func TestDesligarMFAChamaSoAdminSetUserMFAPreference(t *testing.T) {
	api := &apiFalsa{get: func(*cognitoidentityprovider.AdminGetUserInput) (*cognitoidentityprovider.AdminGetUserOutput, error) {
		t.Error("desligar nao deve consultar o usuario")
		return nil, nil
	}}
	if err := NovoCognito(api, "pool-1").DesligarMFA(context.Background(), "sub-1"); err != nil {
		t.Fatal(err)
	}
	if len(api.desliga) != 1 {
		t.Fatalf("esperava 1 chamada, vieram %d", len(api.desliga))
	}
	in := api.desliga[0]
	if aws.ToString(in.Username) != "sub-1" || aws.ToString(in.UserPoolId) != "pool-1" {
		t.Errorf("pessoa ou pool errados: %+v", in)
	}
	if in.SoftwareTokenMfaSettings == nil || in.SoftwareTokenMfaSettings.Enabled || in.SoftwareTokenMfaSettings.PreferredMfa {
		t.Errorf("o TOTP devia ser desligado (Enabled e PreferredMfa falsos): %+v", in.SoftwareTokenMfaSettings)
	}
	if in.SMSMfaSettings != nil || in.EmailMfaSettings != nil {
		t.Error("so o TOTP pode ser mexido")
	}
	api.errDesl = errors.New("falhou")
	if err := NovoCognito(api, "p").DesligarMFA(context.Background(), "s"); err == nil {
		t.Error("a falha do Cognito precisa voltar como erro")
	}
}
