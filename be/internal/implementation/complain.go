package implementation

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// maxComplaintComment — комментарий к жалобе, символов (CHECK в таблице complaints).
const maxComplaintComment = 500

// Complain — rpc Complain: жалоба на фото фиксации, статус «новая».
func (s *CarSearch) Complain(ctx context.Context, req *pb.ComplainRequest) (*pb.Complaint, error) {
	if s.complaints == nil {
		return nil, status.Error(codes.Unimplemented, "жалобы не включены")
	}
	comment := strings.TrimSpace(req.GetComment())
	reason, ok := complaintReasonModel[req.GetReason()]
	switch {
	case uuid.Validate(req.GetSightingId()) != nil:
		return nil, status.Error(codes.InvalidArgument, "sighting_id: нужен uuid фиксации")
	case !ok:
		return nil, status.Error(codes.InvalidArgument, "reason: укажите причину")
	case utf8.RuneCountInString(comment) > maxComplaintComment:
		return nil, status.Errorf(codes.InvalidArgument, "comment: не длиннее %d символов", maxComplaintComment)
	}
	c, err := s.complaints.Complain(ctx, models.ComplainOpts{
		SightingID: req.GetSightingId(), Reason: reason, Comment: comment, IncludeTestData: includeTestData(ctx),
	})
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "фиксация не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось сохранить жалобу")
	}
	return complaintPB(c), nil
}

func complaintPB(c models.Complaint) *pb.Complaint {
	return &pb.Complaint{
		Id:         c.ID,
		SightingId: c.SightingID,
		Reason:     complaintReasonPB[c.Reason],
		Comment:    c.Comment,
		Status:     complaintStatusPB[c.Status],
		CreatedAt:  timestamppb.New(c.CreatedAt),
	}
}

var complaintReasonModel = map[pb.ComplaintReason]models.ComplaintReason{
	pb.ComplaintReason_COMPLAINT_REASON_WRONG_PLATE:   models.ComplaintWrongPlate,
	pb.ComplaintReason_COMPLAINT_REASON_BAD_PHOTO:     models.ComplaintBadPhoto,
	pb.ComplaintReason_COMPLAINT_REASON_PERSONAL_DATA: models.ComplaintPersonalData,
	pb.ComplaintReason_COMPLAINT_REASON_OTHER:         models.ComplaintOther,
}

var complaintReasonPB = map[models.ComplaintReason]pb.ComplaintReason{
	models.ComplaintWrongPlate:   pb.ComplaintReason_COMPLAINT_REASON_WRONG_PLATE,
	models.ComplaintBadPhoto:     pb.ComplaintReason_COMPLAINT_REASON_BAD_PHOTO,
	models.ComplaintPersonalData: pb.ComplaintReason_COMPLAINT_REASON_PERSONAL_DATA,
	models.ComplaintOther:        pb.ComplaintReason_COMPLAINT_REASON_OTHER,
}

var complaintStatusPB = map[models.ComplaintStatus]pb.ComplaintStatus{
	models.ComplaintNew:      pb.ComplaintStatus_COMPLAINT_STATUS_NEW,
	models.ComplaintResolved: pb.ComplaintStatus_COMPLAINT_STATUS_RESOLVED,
}
