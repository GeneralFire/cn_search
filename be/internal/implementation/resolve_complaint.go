package implementation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// ResolveComplaint — rpc ResolveComplaint (админ): отметить жалобу решённой.
func (s *CarSearch) ResolveComplaint(ctx context.Context, req *pb.ResolveComplaintRequest) (*pb.Complaint, error) {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return nil, err
	}
	if s.complaints == nil {
		return nil, status.Error(codes.Unimplemented, "жалобы не включены")
	}
	if uuid.Validate(req.GetId()) != nil {
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid жалобы")
	}
	c, err := s.complaints.ResolveComplaint(ctx, req.GetId())
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "жалоба не найдена")
	case err != nil:
		return nil, internalErr(ctx, err, "не удалось обновить жалобу")
	}
	return complaintPB(c), nil
}
