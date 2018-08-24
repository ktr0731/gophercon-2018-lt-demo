package main

import (
	"log"
	"net"
	"sync"

	"golang.org/x/net/context"

	"github.com/ktr0731/gophercon-2018-lt-demo/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes/empty"
)

type UserService struct {
	store sync.Map
}

func (s *UserService) CreateUsers(ctx context.Context, req *api.CreateUsersRequest) (*api.CreateUsersResponse, error) {
	users := make([]*api.User, 0, len(req.GetUsers()))
	for _, user := range req.GetUsers() {
		u := &api.User{
			Name:        user.GetDisplayName(),
			FirstName:   user.GetFirstName(),
			LastName:    user.GetLastName(),
			DisplayName: user.GetDisplayName(),
			Gender:      user.GetGender(),
		}
		s.store.Store(user.GetDisplayName(), u)
		users = append(users, u)
	}
	return &api.CreateUsersResponse{
		Users: users,
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *api.ListUsersRequest) (*api.ListUsersResponse, error) {
	var users []*api.User
	s.store.Range(func(_, v interface{}) bool {
		users = append(users, v.(*api.User))
		return true
	})
	return &api.ListUsersResponse{
		Users: users,
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *api.GetUserRequest) (*api.User, error) {
	var u *api.User
	s.store.Range(func(k, v interface{}) bool {
		if k.(string) == req.GetName() {
			u = v.(*api.User)
			return false
		}
		return true
	})
	if u != nil {
		return u, nil
	}
	return nil, status.Errorf(codes.NotFound, "no such user: %s", req.GetName())
}

func (s *UserService) DeleteUser(ctx context.Context, req *api.DeleteUserRequest) (*empty.Empty, error) {
	s.store.Delete(req.GetName())
	return &empty.Empty{}, nil
}

func main() {
	addr := "localhost:50051"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()
	api.RegisterUserServiceServer(server, &UserService{
		store: sync.Map{},
	})
	log.Printf("Listen at %s\n", addr)
	if err := server.Serve(l); err != nil {
		log.Fatal(err)
	}
}
