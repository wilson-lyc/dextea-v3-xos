package rpc

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strconv"

	pb "github.com/wilson-lyc/dextea-v3-proto/gen/go/xos/v1"
	"github.com/wilson-lyc/dextea-v3-xos/internal/ecode"
	"github.com/wilson-lyc/dextea-v3-xos/internal/provider"
	"github.com/wilson-lyc/dextea-v3-xos/internal/service"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedXOSServiceServer
	upload        service.UploadService
	gallery       service.GalleryService
	maxUploadSize int64
}

func NewServer(upload service.UploadService, gallery service.GalleryService, maxUploadSize int64) *Server {
	return &Server{upload: upload, gallery: gallery, maxUploadSize: maxUploadSize}
}
func (s *Server) Upload(ctx context.Context, r *pb.UploadRequest) (*pb.UploadResponse, error) {
	if r.GetSource() == "" || r.GetFileName() == "" || len(r.GetContent()) == 0 {
		return nil, ecode.InvalidParam
	}
	if int64(len(r.Content)) > s.maxUploadSize {
		return nil, ecode.FileTooLarge
	}
	out, err := s.upload.Upload(ctx, r.Source, r.Bucket, r.ObjectKey, provider.UploadInput{FileName: r.FileName, Reader: bytes.NewReader(r.Content), Size: int64(len(r.Content))})
	if err != nil {
		return nil, err
	}
	return &pb.UploadResponse{Bucket: out.Bucket, ObjectKey: out.ObjectKey, Size: out.Size, Etag: out.ETag}, nil
}
func (s *Server) ListPage(ctx context.Context, r *pb.ListPageRequest) (*pb.ListPageResponse, error) {
	if r.GetPage() < 1 || r.GetPageSize() < 1 {
		return nil, ecode.InvalidParam
	}
	out, err := s.gallery.ListPage(ctx, int(r.Page), int(r.PageSize))
	if err != nil {
		return nil, err
	}
	resp := &pb.ListPageResponse{Total: out.Total, Page: int32(out.Page), PageSize: int32(out.PageSize), TotalPages: out.TotalPages}
	for _, g := range out.List {
		resp.List = append(resp.List, &pb.Gallery{Id: g.ID, Source: g.Source, Url: g.URL, ObjectKey: g.ObjectKey, Name: g.Name, CreatedAt: timestamppb.New(g.CreatedAt)})
	}
	return resp, nil
}
func (s *Server) Delete(ctx context.Context, r *pb.IDRequest) (*emptypb.Empty, error) {
	if r.GetId() < 1 {
		return nil, ecode.InvalidParam
	}
	if err := s.gallery.Delete(ctx, r.Id); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
func (s *Server) ValidateID(ctx context.Context, r *pb.IDRequest) (*pb.ValidateIDResponse, error) {
	if r.GetId() < 1 {
		return nil, ecode.InvalidParam
	}
	valid, err := s.gallery.ValidateID(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return &pb.ValidateIDResponse{Id: r.Id, Valid: valid}, nil
}
func (s *Server) GetURLs(ctx context.Context, r *pb.GetURLsRequest) (*pb.GetURLsResponse, error) {
	if len(r.GetIds()) == 0 || len(r.GetIds()) > 100 {
		return nil, ecode.InvalidParam
	}
	for _, id := range r.Ids {
		if id < 1 {
			return nil, ecode.InvalidParam
		}
	}
	urls, err := s.gallery.GetURLsByIDs(ctx, r.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.GetURLsResponse{Urls: urls}, nil
}

// UnaryInterceptor converts domain errors without exposing internal causes and recovers panics.
func UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if v := recover(); v != nil {
			log.Printf("RPC %s panic: %v", info.FullMethod, v)
			resp = nil
			err = status.Error(codes.Internal, "internal server error")
		}
	}()
	resp, err = handler(ctx, req)
	if err == nil {
		return resp, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, status.FromContextError(err).Err()
	}
	if _, ok := status.FromError(err); ok {
		return nil, err
	}
	log.Printf("RPC %s: %v", info.FullMethod, err)
	be := ecode.From(err)
	st := status.New(be.RPCCode, be.Message)
	if detailed, e := st.WithDetails(&errdetails.ErrorInfo{Reason: strconv.Itoa(be.Code), Domain: "xos"}); e == nil {
		st = detailed
	}
	return nil, st.Err()
}
