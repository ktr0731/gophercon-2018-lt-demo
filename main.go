package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/ktr0731/gophercon-2018-lt-demo/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes/empty"

	"net/http"
	_ "net/http/pprof"
)

var store = sync.Map{}

type UserService struct{}

func (s *UserService) CreateUsers(ctx context.Context, req *api.CreateUsersRequest) (*api.CreateUsersResponse, error) {
	users := make([]*api.User, 0, len(req.GetUsers()))
	for _, user := range req.GetUsers() {
		u := &api.User{
			Name:      fmt.Sprintf("%s_%s", user.GetFirstName(), user.GetLastName()),
			FirstName: user.GetFirstName(),
			LastName:  user.GetLastName(),
			Language:  user.GetLanguage(),
		}
		store.Store(u.Name, u)
		users = append(users, u)
	}
	return &api.CreateUsersResponse{
		Users: users,
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *api.ListUsersRequest) (*api.ListUsersResponse, error) {
	var users []*api.User
	store.Range(func(_, v interface{}) bool {
		users = append(users, v.(*api.User))
		return true
	})
	return &api.ListUsersResponse{
		Users: users,
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *api.GetUserRequest) (*api.User, error) {
	var u *api.User
	store.Range(func(k, v interface{}) bool {
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
	store.Delete(req.GetName())
	return &empty.Empty{}, nil
}

type GreeterService struct {
	api.GreeterServiceServer
}

func (s *GreeterService) SayHello(ctx context.Context, req *api.SayHelloRequest) (*api.SayHelloResponse, error) {
	user, err := findUser(req.GetGreeterName())
	if err != nil {
		return nil, err
	}

	return &api.SayHelloResponse{
		Message: sayHello(user.LastName, user.Language),
	}, nil
}

func (s *GreeterService) SayHelloClientStream(stream api.GreeterService_SayHelloClientStreamServer) error {
	var greeters []string
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&api.SayHelloResponse{
				// TODO
				Message: sayHello(strings.Join(greeters, ", "), api.Language_ENGLISH),
			})
		}
		if err != nil {
			return err
		}

		user, err := findUser(req.GetGreeterName())
		if err != nil {
			return err
		}

		greeters = append(greeters, user.LastName)
	}
}

func (s *GreeterService) SayHelloServerStream(req *api.SayHelloRequest, stream api.GreeterService_SayHelloServerStreamServer) error {
	n := rand.Intn(5) + 1
	user, err := findUser(req.GetGreeterName())
	if err != nil {
		return err
	}

	message := sayHello(user.LastName, user.Language)
	for i := 0; i < n; i++ {
		if err := stream.Send(&api.SayHelloResponse{
			Message: fmt.Sprintf("%s. I greet %d times.", message, i+1),
		}); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func (s *GreeterService) SayHelloBidiStream(stream api.GreeterService_SayHelloBidiStreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		user, err := findUser(req.GetGreeterName())
		if err != nil {
			return err
		}

		if err := stream.Send(&api.SayHelloResponse{
			Message: sayHello(user.LastName, user.Language),
		}); err != nil {
			return err
		}
	}
}

func sayHello(name string, language api.Language) string {
	var format string
	switch language {
	case api.Language_ENGLISH:
		format = "Hello, %s!"
	case api.Language_JAPANESE:
		format = "こんにちは、%s！"
	default:
		format = "Hello, %s!"
	}
	return fmt.Sprintf(format, name)
}

func findUser(key string) (*api.User, error) {
	user, ok := store.Load(key)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no such user: %s", key)
	}
	return user.(*api.User), nil
}

func main() {
	addr := "localhost:50051"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	// for pprof
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	server := grpc.NewServer()
	api.RegisterUserServiceServer(server, &UserService{})
	api.RegisterGreeterServiceServer(server, &GreeterService{})
	log.Printf("Listen at %s\n", addr)
	if err := server.Serve(l); err != nil {
		log.Fatal(err)
	}
}
