package authgrpc

import (
	"context"
	ssov1 "github.com/grihezz/grpcv3/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const emptyValue = 0

type Auth interface {
	Login(ctx context.Context, email string, password string, app_id int) (token string, err error)
	RegisterNewUser(ctx context.Context, email string, password string) (int64, error)
	IsAdmin(ctx context.Context, user_id int64) (bool, error)
}

type serverAPI struct {
	ssov1.UnimplementedAuthServer
	auth Auth
}

func Register(gRPC *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPC, &serverAPI{auth: auth})
}

func (s *serverAPI) Login(ctx context.Context, req *ssov1.LoginRequest) (*ssov1.LoginResponce, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if req.GetAppId() == emptyValue {
		return nil, status.Error(codes.InvalidArgument, "app id is required")
	}

	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), int(req.GetAppId()))

	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.LoginResponce{
		Token: token,
	}, nil
}

func (s *serverAPI) Register(ctx context.Context, req *ssov1.RegisterRequest) (*ssov1.RegisterResponse, error) {
	if err := valideteRegiser(req); err != nil {
		return nil, err
	}

	user_id, err := s.auth.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword())

	if err != nil {
		// todo
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.RegisterResponse{
		UserId: user_id,
	}, nil
}

func (s *serverAPI) IsAdmin(ctx context.Context, req *ssov1.IsAdminRequest) (*ssov1.IsAdminResponce, error) {
	if err := valideteIsAdmin(req); err != nil {
		return nil, err
	}

	isAdmin, err := s.auth.IsAdmin(ctx, req.GetUserId())

	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &ssov1.IsAdminResponce{
		IsAdmin: isAdmin,
	}, nil
}

func valideteRegiser(req *ssov1.RegisterRequest) error {
	if req.GetEmail() == "" {
		return status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		return status.Error(codes.InvalidArgument, "password is required")
	}
	return nil
}

func valideteIsAdmin(req *ssov1.IsAdminRequest) error {
	if req.GetUserId() == emptyValue {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	return nil
}
