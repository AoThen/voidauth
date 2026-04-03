package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"goauth/internal/model"
)

var ErrClientNotFound = errors.New("client not found")

type ClientRepo struct {
	db *sqlx.DB
}

func NewClientRepo(db *sqlx.DB) *ClientRepo {
	return &ClientRepo{db: db}
}

// ClientResponse 客户端响应
type ClientResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	HasSecret     bool     `json:"hasSecret"`
	RedirectURIs  []string `json:"redirectUris"`
	PostLogoutURIs []string `json:"postLogoutUris"`
	Scopes        []string `json:"scopes"`
	GrantTypes    []string `json:"grantTypes"`
	ResponseTypes []string `json:"responseTypes"`
	Trusted       bool     `json:"trusted"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

// Create 创建客户端
func (r *ClientRepo) Create(ctx context.Context, client *model.Client) error {
	if client.ID == "" {
		client.ID = uuid.NewString()
	}

	// 哈希密钥
	if client.Secret != nil && *client.Secret != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*client.Secret), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		hashedSecret := string(hashed)
		client.Secret = &hashedSecret
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO clients (id, name, secret, redirectUris, postLogoutUris, scopes, grantTypes, responseTypes, tokenEndpointAuth, trusted, createdBy, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
	`, client.ID, client.Name, client.Secret, client.RedirectURIs, client.PostLogoutURIs, client.Scopes, client.GrantTypes, client.ResponseTypes, client.TokenEndpointAuth, client.Trusted, client.CreatedBy)

	return err
}

// FindByID 根据 ID 查找客户端
func (r *ClientRepo) FindByID(ctx context.Context, id string) (*model.Client, error) {
	var client model.Client
	err := r.db.GetContext(ctx, &client, `SELECT * FROM clients WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

// List 列出客户端
func (r *ClientRepo) List(ctx context.Context) ([]*ClientResponse, error) {
	var clients []struct {
		ID                string  `db:"id"`
		Name              string  `db:"name"`
		Secret            *string `db:"secret"`
		RedirectURIs      string  `db:"redirectUris"`
		PostLogoutURIs    string  `db:"postLogoutUris"`
		Scopes            string  `db:"scopes"`
		GrantTypes        string  `db:"grantTypes"`
		ResponseTypes     string  `db:"responseTypes"`
		TokenEndpointAuth string  `db:"tokenEndpointAuth"`
		Trusted           int     `db:"trusted"` // SQLite 使用 INTEGER 存储
		CreatedAt         string  `db:"createdAt"`
		UpdatedAt         string  `db:"updatedAt"`
	}

	err := r.db.SelectContext(ctx, &clients, `SELECT id, name, secret, redirectUris, postLogoutUris, scopes, grantTypes, responseTypes, tokenEndpointAuth, trusted, createdAt, updatedAt FROM clients ORDER BY createdAt DESC`)
	if err != nil {
		return nil, err
	}

	responses := make([]*ClientResponse, len(clients))
	for i, c := range clients {
		var redirectURIs, postLogoutURIs, scopes, grantTypes, responseTypes []string
		json.Unmarshal([]byte(c.RedirectURIs), &redirectURIs)
		json.Unmarshal([]byte(c.PostLogoutURIs), &postLogoutURIs)
		json.Unmarshal([]byte(c.Scopes), &scopes)
		json.Unmarshal([]byte(c.GrantTypes), &grantTypes)
		json.Unmarshal([]byte(c.ResponseTypes), &responseTypes)

		responses[i] = &ClientResponse{
			ID:             c.ID,
			Name:           c.Name,
			HasSecret:      c.Secret != nil && *c.Secret != "",
			RedirectURIs:   redirectURIs,
			PostLogoutURIs: postLogoutURIs,
			Scopes:         scopes,
			GrantTypes:     grantTypes,
			ResponseTypes:  responseTypes,
			Trusted:        c.Trusted == 1, // 转换为 bool
			CreatedAt:      c.CreatedAt,
			UpdatedAt:      c.UpdatedAt,
		}
	}

	return responses, nil
}

// Update 更新客户端
func (r *ClientRepo) Update(ctx context.Context, client *model.Client) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE clients SET name = ?, redirectUris = ?, postLogoutUris = ?, scopes = ?, grantTypes = ?, responseTypes = ?, tokenEndpointAuth = ?, trusted = ?, updatedAt = datetime('now')
		WHERE id = ?
	`, client.Name, client.RedirectURIs, client.PostLogoutURIs, client.Scopes, client.GrantTypes, client.ResponseTypes, client.TokenEndpointAuth, client.Trusted, client.ID)
	return err
}

// Delete 删除客户端
func (r *ClientRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM clients WHERE id = ?`, id)
	return err
}

// ValidateSecret 验证客户端密钥
func (r *ClientRepo) ValidateSecret(ctx context.Context, id, secret string) (bool, error) {
	var hashedSecret string
	err := r.db.GetContext(ctx, &hashedSecret, `SELECT secret FROM clients WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrClientNotFound
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedSecret), []byte(secret))
	return err == nil, nil
}
