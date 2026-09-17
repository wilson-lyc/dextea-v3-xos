package rpc

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	pb "github.com/wilson-lyc/dextea-v3-proto/gen/go/xos/v1"
	"github.com/wilson-lyc/dextea-v3-xos/internal/dto"
	"github.com/wilson-lyc/dextea-v3-xos/internal/ecode"
	"github.com/wilson-lyc/dextea-v3-xos/internal/provider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type uploadStub struct{ size int64 }

func (s *uploadStub) Upload(ctx context.Context, source, bucket, key string, in provider.UploadInput) (*dto.UploadResp, error) {
	b, err := io.ReadAll(in.Reader)
	if err != nil {
		return nil, err
	}
	s.size = int64(len(b))
	return &dto.UploadResp{Bucket: bucket, ObjectKey: key, Size: s.size}, nil
}
func TestUploadTransport(t *testing.T) {
	const limit = 6 << 20
	u := &uploadStub{}
	srv := grpc.NewServer(grpc.MaxRecvMsgSize(limit+(64<<10)), grpc.UnaryInterceptor(UnaryInterceptor))
	pb.RegisterXOSServiceServer(srv, NewServer(u, nil, limit))
	l := bufconn.Listen(1 << 20)
	go srv.Serve(l)
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///buf", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return l.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewXOSServiceClient(conn)
	out, err := client.Upload(context.Background(), &pb.UploadRequest{Source: "test", FileName: "a.png", Content: make([]byte, 5<<20)})
	if err != nil || out.GetSize() != 5<<20 || u.size != 5<<20 {
		t.Fatalf("large upload: %v %v", out, err)
	}
	_, err = client.Upload(context.Background(), &pb.UploadRequest{Source: "test", FileName: "a.png", Content: make([]byte, limit+1)})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("size limit: %v", err)
	}
	_, err = client.Upload(context.Background(), &pb.UploadRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("validation: %v", err)
	}
}
func TestErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code codes.Code
	}{{ecode.NotFound, codes.NotFound}, {ecode.FileTypeDenied, codes.InvalidArgument}, {ecode.FileTooLarge, codes.ResourceExhausted}, {context.Canceled, codes.Canceled}, {context.DeadlineExceeded, codes.DeadlineExceeded}, {errors.New("secret database password"), codes.Internal}} {
		_, err := UnaryInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) { return nil, tc.err })
		if status.Code(err) != tc.code {
			t.Fatalf("%v: %v", tc.err, err)
		}
		if tc.code == codes.Internal && status.Convert(err).Message() != "internal server error" {
			t.Fatal("internal error leaked")
		}
	}
	_, err := UnaryInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) { panic("secret") })
	if status.Code(err) != codes.Internal {
		t.Fatal(err)
	}
}
func TestInvalidGalleryRequests(t *testing.T) {
	s := NewServer(nil, nil, 10)
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { _, e := s.ListPage(ctx, &pb.ListPageRequest{Page: 1}); return e },
		func() error { _, e := s.Delete(ctx, &pb.IDRequest{}); return e },
		func() error { _, e := s.ValidateID(ctx, &pb.IDRequest{Id: -1}); return e },
		func() error { _, e := s.GetURLs(ctx, &pb.GetURLsRequest{}); return e },
		func() error { _, e := s.GetURLs(ctx, &pb.GetURLsRequest{Ids: []int64{1, -1}}); return e },
		func() error { _, e := s.GetURLs(ctx, &pb.GetURLsRequest{Ids: make([]int64, 101)}); return e },
	} {
		if !errors.Is(call(), ecode.InvalidParam) {
			t.Fatal("expected invalid parameter")
		}
	}
}
