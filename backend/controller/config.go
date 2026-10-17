package controller

import (
	"context"

	"financas/domain"
	"financas/service"
)

// Config liga as rotas de configuracao, aparencia, tags e contador ao service.
type Config struct {
	cfg  *service.Config
	tags *service.Tags
	cont *service.Contador
}

func NewConfig(cfg *service.Config, tags *service.Tags, cont *service.Contador) *Config {
	return &Config{cfg: cfg, tags: tags, cont: cont}
}

// GetCfg: GET /spaces/{sid}/cfg/{nome}
func (c *Config) GetCfg(ctx context.Context, sid, nome string) (Resp, error) {
	d, err := c.cfg.Ler(ctx, sid, nome)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PutCfg: PUT /spaces/{sid}/cfg/{nome} (so o dono). O corpo leva o "version" que o cliente leu.
func (c *Config) PutCfg(ctx context.Context, sid, nome string, body domain.Doc) (Resp, error) {
	d, err := c.cfg.Gravar(ctx, sid, nome, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// GetAparencia: GET /spaces/{sid}/aparencia (a da pessoa que fez a chamada)
func (c *Config) GetAparencia(ctx context.Context, sid, sub string) (Resp, error) {
	d, err := c.cfg.LerAparencia(ctx, sid, sub)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PutAparencia: PUT /spaces/{sid}/aparencia
func (c *Config) PutAparencia(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	d, err := c.cfg.GravarAparencia(ctx, sid, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}

// PostTags: POST /spaces/{sid}/tags
func (c *Config) PostTags(ctx context.Context, sid string, body domain.Doc) (Resp, error) {
	adicionadas, err := c.tags.Adicionar(ctx, sid, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"adicionadas": adicionadas})
}

// GetSeq: GET /spaces/{sid}/seq
func (c *Config) GetSeq(ctx context.Context, sid string) (Resp, error) {
	n, err := c.cont.Atual(ctx, sid)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, domain.Doc{"seq": n})
}

// Perfil liga PUT /spaces/{sid}/perfil ao service.
type Perfil struct{ svc *service.Perfil }

func NewPerfil(svc *service.Perfil) *Perfil { return &Perfil{svc: svc} }

// PutPerfil: PUT /spaces/{sid}/perfil. Altera o vinculo de quem fez a chamada.
func (c *Perfil) PutPerfil(ctx context.Context, sid, sub string, body domain.Doc) (Resp, error) {
	d, err := c.svc.Editar(ctx, sid, sub, body)
	if err != nil {
		return respostaErro(err)
	}
	return Out(200, d)
}
