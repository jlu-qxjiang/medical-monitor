package service

import (
	"context"
	"fmt"
	"time"

	"github.com/medical-monitor/internal/model"
	"github.com/medical-monitor/internal/pkg/utils"
	"github.com/medical-monitor/internal/repository"
	"gorm.io/gorm"
)

// FamilyService 家庭组业务逻辑：创建、加入、退出、成员角色管理、邀请码刷新。
// 角色体系：owner(创建者)→admin(管理员)→member(普通成员)→elder(老人)
// 只有 owner 和 admin 能管理其他成员。
type FamilyService struct {
	familyRepo *repository.FamilyRepository // 家庭组及成员持久化
	userRepo   *repository.UserRepository   // 用户查询
	auditRepo  *repository.AuditRepository  // 审计日志
}

func NewFamilyService(
	familyRepo *repository.FamilyRepository,
	userRepo *repository.UserRepository,
	auditRepo *repository.AuditRepository,
) *FamilyService {
	return &FamilyService{familyRepo: familyRepo, userRepo: userRepo, auditRepo: auditRepo}
}

// ====== 创建家庭组 ======

// Create 创建家庭组，生成 6 位邀请码（10 分钟过期），创建者自动成为 owner。
func (s *FamilyService) Create(ctx context.Context, userID uint64, name string) (*FamilyCreateResponse, error) {
	// 1. 生成 6 位随机数字邀请码
	inviteCode, err := utils.GenerateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("生成邀请码失败: %w", err)
	}

	// 2. 邀请码 10 分钟有效期
	expiresAt := time.Now().Add(10 * time.Minute)

	// 3. 创建家庭组记录
	family := &model.Family{
		Name:                name,
		InviteCode:          inviteCode,
		InviteCodeExpiresAt: &expiresAt,
		CreatedBy:           userID,
	}
	if err := s.familyRepo.CreateFamily(ctx, family); err != nil {
		return nil, fmt.Errorf("创建家庭组失败: %w", err)
	}

	// 4. 创建者成为 owner（最高权限）
	member := &model.FamilyMember{FamilyID: family.ID, UserID: userID, Role: model.RoleOwner}
	if err := s.familyRepo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("添加创建者为成员失败: %w", err)
	}

	// 5. 审计日志
	uid := userID
	fid := family.ID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionFamilyJoin, model.AuditTargetFamily, &family.ID, fmt.Sprintf(`{"name":"%s"}`, name), "")

	return &FamilyCreateResponse{
		ID: family.ID, Name: family.Name, InviteCode: family.InviteCode,
		InviteCodeExpiresAt: family.InviteCodeExpiresAt,
		CreatedBy: family.CreatedBy, CreatedAt: family.CreatedAt,
	}, nil
}

// List 返回当前用户所在的所有家庭组列表（含成员数+老人列表+邀请码状态）。
func (s *FamilyService) List(ctx context.Context, userID uint64) (*FamilyListResponse, error) {
	memberships, err := s.familyRepo.FindFamiliesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询家庭组失败: %w", err)
	}

	list := make([]FamilyListItem, 0, len(memberships))
	for _, m := range memberships {
		family, err := s.familyRepo.FindFamilyByID(ctx, m.FamilyID)
		if err != nil {
			continue // 跳过已删除的家庭组
		}
		memberCount, _ := s.familyRepo.CountMembers(ctx, m.FamilyID)
		// 查家庭中的老人列表
		elders, _ := s.familyRepo.FindEldersByFamilyID(ctx, m.FamilyID)
		elderInfos := make([]ElderInfo, 0, len(elders))
		for _, e := range elders {
			if e.User != nil {
				elderInfos = append(elderInfos, ElderInfo{UserID: e.UserID, Nickname: e.User.Nickname, AvatarURL: e.User.AvatarURL})
			}
		}
		list = append(list, FamilyListItem{
			ID: family.ID, Name: family.Name,
			InviteCode: family.InviteCode, InviteCodeExpiresAt: family.InviteCodeExpiresAt,
			MyRole: m.Role, MemberCount: int(memberCount), Elders: elderInfos,
		})
	}
	return &FamilyListResponse{List: list}, nil
}

// Detail 返回指定家庭组的完整信息（含所有成员列表+昵称+角色+加入时间）。
func (s *FamilyService) Detail(ctx context.Context, familyID uint64) (*FamilyDetailResponse, error) {
	family, err := s.familyRepo.FindFamilyByID(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("家庭组不存在")
	}
	members, err := s.familyRepo.FindMembersByFamilyID(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("查询成员失败: %w", err)
	}
	memberInfos := make([]MemberInfo, 0, len(members))
	for _, m := range members {
		info := MemberInfo{UserID: m.UserID, Role: m.Role, JoinedAt: m.JoinedAt}
		if m.User != nil {
			info.Nickname = m.User.Nickname
			info.AvatarURL = m.User.AvatarURL
		}
		memberInfos = append(memberInfos, info)
	}
	return &FamilyDetailResponse{
		ID: family.ID, Name: family.Name,
		InviteCode: family.InviteCode, InviteCodeExpiresAt: family.InviteCodeExpiresAt,
		Members: memberInfos,
	}, nil
}

// ====== 加入家庭组 ======

// Join 通过 6 位邀请码加入家庭组。默认 role=member（子女）。
// 两层校验：邀请码是否正确 → 邀请码是否过期 → 用户是否已在组中。
func (s *FamilyService) Join(ctx context.Context, userID uint64, inviteCode string) (string, error) {
	// 1. 查邀请码是否存在
	family, err := s.familyRepo.FindFamilyByInviteCode(ctx, inviteCode)
	if err != nil {
		return "", fmt.Errorf("邀请码无效或已过期")
	}

	// 2. 检查邀请码是否过期（10 分钟有效期）
	if family.InviteCodeExpiresAt != nil && time.Now().After(*family.InviteCodeExpiresAt) {
		return "", fmt.Errorf("邀请码已过期，请刷新后重试")
	}

	// 3. 防止重复加入
	member, err := s.familyRepo.FindMember(ctx, family.ID, userID)
	if err == nil && member != nil {
		return "", fmt.Errorf("您已在该家庭组中")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("查询成员状态失败: %w", err)
	}

	// 4. 加入，默认角色为 member
	newMember := &model.FamilyMember{FamilyID: family.ID, UserID: userID, Role: model.RoleMember}
	if err := s.familyRepo.AddMember(ctx, newMember); err != nil {
		return "", fmt.Errorf("加入家庭组失败: %w", err)
	}

	// 5. 审计日志
	uid := userID
	fid := family.ID
	_ = s.auditRepo.LogAction(ctx, &uid, &fid, model.AuditActionFamilyJoin, model.AuditTargetFamily, &family.ID, fmt.Sprintf(`{"invite_code":"%s"}`, inviteCode), "")

	return fmt.Sprintf("已加入「%s」", family.Name), nil
}

// ====== 成员管理 ======

// SetMemberRole 修改成员角色。仅 owner/admin 可操作，不能改 owner。
// newRole: elder=老人 / member=子女 / admin=管理员
func (s *FamilyService) SetMemberRole(ctx context.Context, familyID, operatorUserID, targetUserID uint64, newRole string) error {
	// 校验角色名是否合法
	if !model.IsValidRole(newRole) {
		return fmt.Errorf("无效的角色: %s", newRole)
	}
	// owner 不能通过此接口设置
	if newRole == model.RoleOwner {
		return fmt.Errorf("无法通过此接口设置owner角色")
	}

	// 操作者必须是 owner 或 admin
	operator, err := s.familyRepo.FindMember(ctx, familyID, operatorUserID)
	if err != nil {
		return fmt.Errorf("您不是该家庭组成员")
	}
	if operator.Role != model.RoleOwner && operator.Role != model.RoleAdmin {
		return fmt.Errorf("仅owner或admin可以修改成员角色")
	}

	// 目标用户必须在组内
	target, err := s.familyRepo.FindMember(ctx, familyID, targetUserID)
	if err != nil {
		return fmt.Errorf("目标用户不是该家庭组成员")
	}
	// 不能修改 owner 的角色
	if target.Role == model.RoleOwner {
		return fmt.Errorf("无法修改owner的角色")
	}

	if err := s.familyRepo.UpdateMemberRole(ctx, target.ID, newRole); err != nil {
		return fmt.Errorf("修改角色失败: %w", err)
	}

	// 审计日志：记录角色变更前后的值
	fid := familyID
	_ = s.auditRepo.LogAction(ctx, &operatorUserID, &fid, model.AuditActionRoleChange, model.AuditTargetUser, &targetUserID, fmt.Sprintf(`{"from":"%s","to":"%s"}`, target.Role, newRole), "")
	return nil
}

// RemoveMember 将指定成员移出家庭组。不能移除自己（应用 Leave），不能移除 owner。
func (s *FamilyService) RemoveMember(ctx context.Context, familyID, operatorUserID, targetUserID uint64) error {
	if operatorUserID == targetUserID {
		return fmt.Errorf("无法移除自己，请使用退出功能")
	}
	operator, err := s.familyRepo.FindMember(ctx, familyID, operatorUserID)
	if err != nil {
		return fmt.Errorf("您不是该家庭组成员")
	}
	if operator.Role != model.RoleOwner && operator.Role != model.RoleAdmin {
		return fmt.Errorf("仅owner或admin可以移除成员")
	}
	target, err := s.familyRepo.FindMember(ctx, familyID, targetUserID)
	if err != nil {
		return fmt.Errorf("目标用户不是该家庭组成员")
	}
	if target.Role == model.RoleOwner {
		return fmt.Errorf("无法移除owner")
	}
	if err := s.familyRepo.RemoveMember(ctx, target.ID); err != nil {
		return fmt.Errorf("移除成员失败: %w", err)
	}
	fid := familyID
	_ = s.auditRepo.LogAction(ctx, &operatorUserID, &fid, model.AuditActionMemberRemove, model.AuditTargetUser, &targetUserID, "", "")
	return nil
}

// Leave 当前用户主动退出家庭组。owner 不能退出（需先解散或转让）。
func (s *FamilyService) Leave(ctx context.Context, familyID, userID uint64) error {
	member, err := s.familyRepo.FindMember(ctx, familyID, userID)
	if err != nil {
		return fmt.Errorf("您不是该家庭组成员")
	}
	if member.Role == model.RoleOwner {
		return fmt.Errorf("owner无法退出，请先转让ownership或解散家庭组")
	}
	if err := s.familyRepo.RemoveMember(ctx, member.ID); err != nil {
		return fmt.Errorf("退出家庭组失败: %w", err)
	}
	fid := familyID
	_ = s.auditRepo.LogAction(ctx, &userID, &fid, model.AuditActionFamilyLeave, model.AuditTargetFamily, &familyID, "", "")
	return nil
}

// ====== 邀请码 ======

// RefreshInviteCode 刷新邀请码。旧码立即失效，新码重置 10 分钟有效期。
func (s *FamilyService) RefreshInviteCode(ctx context.Context, familyID, userID uint64) (string, error) {
	// 仅 owner/admin 可刷新
	member, err := s.familyRepo.FindMember(ctx, familyID, userID)
	if err != nil || (member.Role != model.RoleOwner && member.Role != model.RoleAdmin) {
		return "", fmt.Errorf("仅owner或admin可以刷新邀请码")
	}
	newCode, err := utils.GenerateInviteCode()
	if err != nil {
		return "", fmt.Errorf("生成邀请码失败: %w", err)
	}
	expiresAt := time.Now().Add(10 * time.Minute)
	if err := s.familyRepo.UpdateInviteCode(ctx, familyID, newCode, &expiresAt); err != nil {
		return "", fmt.Errorf("更新邀请码失败: %w", err)
	}
	return newCode, nil
}

// ====== Request / Response 类型 ======

type CreateFamilyRequest struct {
	Name string `json:"name" binding:"required,max=64"` // 家庭组名称
}

type FamilyCreateResponse struct {
	ID                  uint64     `json:"id"`
	Name                string     `json:"name"`
	InviteCode          string     `json:"invite_code"`            // 6 位邀请码
	InviteCodeExpiresAt *time.Time `json:"invite_code_expires_at"` // 过期时间
	CreatedBy           uint64     `json:"created_by"`             // 创建者 ID
	CreatedAt           time.Time  `json:"created_at"`
}

type FamilyListResponse struct {
	List []FamilyListItem `json:"list"`
}

type FamilyListItem struct {
	ID                   uint64      `json:"id"`
	Name                 string      `json:"name"`
	InviteCode           string      `json:"invite_code"`
	InviteCodeExpiresAt  *time.Time  `json:"invite_code_expires_at"` // 邀请码过期时间
	IsPersonal           bool        `json:"is_personal"`            // 是否个人私有空间
	MyRole               string      `json:"my_role"`               // 当前用户在该组的角色
	MemberCount          int         `json:"member_count"`          // 成员总数
	Elders               []ElderInfo `json:"elders"`                // 组内老人列表
}

type ElderInfo struct {
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type FamilyDetailResponse struct {
	ID                  uint64       `json:"id"`
	Name                string       `json:"name"`
	InviteCode          string       `json:"invite_code"`
	InviteCodeExpiresAt *time.Time   `json:"invite_code_expires_at"`
	Members             []MemberInfo `json:"members"` // 完整成员列表
}

type MemberInfo struct {
	UserID    uint64    `json:"user_id"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
	Role      string    `json:"role"`      // owner/admin/member/elder
	JoinedAt  time.Time `json:"joined_at"` // 加入时间
}

type JoinFamilyRequest struct {
	InviteCode string `json:"invite_code" binding:"required,len=6"` // 6 位邀请码
}

type SetRoleRequest struct {
	Role string `json:"role" binding:"required"` // 目标角色：elder/member/admin
}
