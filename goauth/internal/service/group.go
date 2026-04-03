package service

import (
	"context"

	"github.com/jmoiron/sqlx"

	"goauth/internal/model"
	"goauth/internal/repo"
)

// GroupService 分组服务
type GroupService struct {
	groupRepo *repo.GroupRepo
	db        *sqlx.DB
}

// NewGroupService 创建分组服务
func NewGroupService(groupRepo *repo.GroupRepo, db *sqlx.DB) *GroupService {
	return &GroupService{
		groupRepo: groupRepo,
		db:        db,
	}
}

// List 列出分组
func (s *GroupService) List(ctx context.Context) ([]*model.Group, error) {
	return s.groupRepo.List(ctx)
}

// Create 创建分组
func (s *GroupService) Create(ctx context.Context, group *model.Group) error {
	return s.groupRepo.Create(ctx, group)
}

// Update 更新分组
func (s *GroupService) Update(ctx context.Context, group *model.Group) error {
	return s.groupRepo.Update(ctx, group)
}

// Delete 删除分组
func (s *GroupService) Delete(ctx context.Context, id string) error {
	return s.groupRepo.Delete(ctx, id)
}

// AddMember 添加成员
func (s *GroupService) AddMember(ctx context.Context, groupID, userID string) error {
	return s.groupRepo.AddUserToGroup(ctx, userID, groupID)
}

// RemoveMember 移除成员
func (s *GroupService) RemoveMember(ctx context.Context, groupID, userID string) error {
	return s.groupRepo.RemoveUserFromGroup(ctx, userID, groupID)
}

// GetMembers 获取成员
func (s *GroupService) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	return s.groupRepo.GetGroupMembers(ctx, groupID)
}

// GetUserGroups 获取用户分组
func (s *GroupService) GetUserGroups(ctx context.Context, userID string) ([]*model.GroupRef, error) {
	return s.groupRepo.GetUserGroups(ctx, userID)
}
