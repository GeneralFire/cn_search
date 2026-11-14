package implementation

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// ListAdminAudit — rpc ListAdminAudit (админ): журнал действий админов, от новых к старым.
func (s *CarSearch) ListAdminAudit(ctx context.Context, req *pb.ListAdminAuditRequest) (*pb.ListAdminAuditResponse, error) {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return nil, err
	}
	if s.admin == nil {
		return nil, status.Error(codes.Unimplemented, "админские методы не включены")
	}
	if len(req.GetAction()) > 64 || len(req.GetTargetId()) > 64 {
		return nil, status.Error(codes.InvalidArgument, "action/target_id: слишком длинное значение")
	}
	if id := req.GetTargetId(); id != "" && uuid.Validate(id) != nil {
		return nil, status.Error(codes.InvalidArgument, "target_id: нужен uuid объекта")
	}
	offset, limit, err := pageArgs(req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, err
	}
	page, err := s.admin.ListAudit(ctx, models.ListAuditOpts{Action: req.GetAction(), TargetID: req.GetTargetId(), Offset: offset, Limit: limit})
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось получить журнал")
	}
	resp := &pb.ListAdminAuditResponse{Items: make([]*pb.AdminAuditEntry, len(page.Items)), Total: int32(page.Total), NextOffset: nextOffsetPB(page.NextOffset)}
	for i, e := range page.Items {
		resp.Items[i] = &pb.AdminAuditEntry{
			Id: e.ID, Action: e.Action, Actor: e.Actor, TargetType: e.TargetType, TargetId: e.TargetID,
			Details: string(e.Details), RequestId: e.RequestID, CreatedAt: timestamppb.New(e.CreatedAt), TargetName: e.TargetName,
		}
	}
	return resp, nil
}
