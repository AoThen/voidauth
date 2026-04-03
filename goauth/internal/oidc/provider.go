package oidc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/crypto/bcrypt"

	"goauth/internal/config"
	"goauth/internal/model"
	"goauth/internal/repo"
)

// Provider OIDC Provider 封装
type Provider struct {
	provider *op.Provider
	storage  *Storage
	config   *config.Config
}

// NewProvider 创建 OIDC Provider
func NewProvider(cfg *config.Config, storage *Storage) (*Provider, error) {
	// 创建 provider 配置
	opConfig := &op.Config{
		DefaultLogoutRedirectURI: "/",
	}

	// 转换加密密钥
	var cryptoKey [32]byte
	copy(cryptoKey[:], cfg.Security.CryptoKey)
	opConfig.CryptoKey = cryptoKey

	// Provider 选项
	var opts []op.Option

	// 开发/测试模式允许 HTTP
	if cfg.Server.Environment == "development" || cfg.Server.Environment == "test" {
		opts = append(opts, op.WithAllowInsecure())
	}

	// 创建 provider
	provider, err := op.NewProvider(opConfig, storage, op.StaticIssuer(cfg.OIDC.Issuer), opts...)
	if err != nil {
		return nil, err
	}

	return &Provider{
		provider: provider,
		storage:  storage,
		config:   cfg,
	}, nil
}

// Handler 返回 OIDC HTTP Handler
func (p *Provider) Handler() http.Handler {
	return p.provider.HttpHandler()
}

// Storage 返回 Storage
func (p *Provider) Storage() *Storage {
	return p.storage
}

// OpProvider 返回底层 op.Provider
func (p *Provider) OpProvider() *op.Provider {
	return p.provider
}

// CreateClient 创建 OIDC 客户端
func CreateClient(ctx context.Context, oidcRepo *repo.OIDCRepo, clientID, clientSecret, name string, redirectURIs, scopes []string, trusted bool) error {
	// 使用 bcrypt 哈希客户端密钥
	var hashedSecret string
	if clientSecret != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		hashedSecret = string(hash)
	}

	client := &Client{
		ID_:                     clientID,
		Secret_:                 hashedSecret,
		Name_:                   name,
		RedirectURIs_:           redirectURIs,
		PostLogoutRedirectURIs_: []string{},
		ApplicationType_:        op.ApplicationTypeWeb,
		ResponseTypes_:          []oidc.ResponseType{oidc.ResponseTypeCode},
		GrantTypes_:             []oidc.GrantType{oidc.GrantTypeCode, oidc.GrantTypeRefreshToken},
		AccessTokenType_:        op.AccessTokenTypeBearer,
		IDTokenLifetime_:        30 * 60 * 1000000000, // 30 分钟
		DevMode_:                false,
		ClockSkew_:              0,
		Scopes_:                 scopes,
		SkipConsent_:            trusted,
		Trusted_:                trusted,
	}

	payload, err := json.Marshal(client)
	if err != nil {
		return err
	}

	oidcPayload := &model.OIDCPayload{
		ID:      clientID,
		Type:    "client",
		Payload: string(payload),
	}

	return oidcRepo.Create(ctx, oidcPayload)
}

// UpdateClient 更新 OIDC 客户端
func UpdateClient(ctx context.Context, oidcRepo *repo.OIDCRepo, clientID, name string, redirectURIs, scopes []string, trusted bool) error {
	payload, err := oidcRepo.FindByIDAndType(ctx, clientID, "client")
	if err != nil {
		return err
	}

	var client Client
	if err := json.Unmarshal([]byte(payload.Payload), &client); err != nil {
		return err
	}

	if name != "" {
		client.Name_ = name
	}
	if redirectURIs != nil {
		client.RedirectURIs_ = redirectURIs
	}
	if scopes != nil {
		client.Scopes_ = scopes
	}
	client.Trusted_ = trusted
	client.SkipConsent_ = trusted

	newPayload, err := json.Marshal(&client)
	if err != nil {
		return err
	}

	payload.Payload = string(newPayload)
	return oidcRepo.Update(ctx, payload)
}

// DeleteClient 删除客户端
func DeleteClient(ctx context.Context, oidcRepo *repo.OIDCRepo, clientID string) error {
	return oidcRepo.Delete(ctx, clientID, "client")
}

// GetClient 获取客户端
func GetClient(ctx context.Context, oidcRepo *repo.OIDCRepo, clientID string) (*Client, error) {
	payload, err := oidcRepo.FindByIDAndType(ctx, clientID, "client")
	if err != nil {
		return nil, err
	}

	var client Client
	if err := json.Unmarshal([]byte(payload.Payload), &client); err != nil {
		return nil, err
	}

	return &client, nil
}

// ListClients 列出客户端
func ListClients(ctx context.Context, oidcRepo *repo.OIDCRepo) ([]*Client, error) {
	payloads, err := oidcRepo.ListByType(ctx, "client", 100, 0)
	if err != nil {
		return nil, err
	}

	clients := make([]*Client, 0, len(payloads))
	for _, p := range payloads {
		var client Client
		if err := json.Unmarshal([]byte(p.Payload), &client); err != nil {
			continue
		}
		clients = append(clients, &client)
	}

	return clients, nil
}
