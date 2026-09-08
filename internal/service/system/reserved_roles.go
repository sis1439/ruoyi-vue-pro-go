package system

import (
	"context"
	"errors"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"strings"
)

const PlatformAdminRole = "platform_admin"

func reservedRole(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	return code == PlatformAdminRole || code == consts.RoleCodeSuperAdmin
}
func rejectReservedUser(ctx context.Context, q *query.Query, userID int64) error {
	links, err := q.SystemUserRole.WithContext(ctx).Where(q.SystemUserRole.UserID.Eq(userID)).Find()
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.RoleID)
	}
	if len(ids) == 0 {
		return nil
	}
	roles, err := q.SystemRole.WithContext(ctx).Where(q.SystemRole.ID.In(ids...)).Find()
	if err != nil {
		return err
	}
	for _, role := range roles {
		if reservedRole(role.Code) {
			return errors.New("reserved administrator accounts require offline management")
		}
	}
	return nil
}

func validateAssignableRoles(ctx context.Context, q *query.Query, ids []int64) error {
	for _, id := range ids {
		role, err := q.SystemRole.WithContext(ctx).Where(q.SystemRole.ID.Eq(id)).First()
		if err != nil {
			return err
		}
		if reservedRole(role.Code) {
			return errors.New("reserved roles require offline assignment")
		}
	}
	return nil
}
