package handler

import (
	"context"
	"errors"

	moviepb "my-go-app/api/proto"
	"my-go-app/internal/domain"
	"my-go-app/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	moviepb.UnimplementedMovieServiceServer
	service *usecase.Service
}

func NewGRPCServer(service *usecase.Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) LoadMovies(
	ctx context.Context,
	req *moviepb.LoadMoviesRequest,
) (*moviepb.LoadMoviesResponse, error) {
	if req == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"запрос не должен быть пустым",
		)
	}

	result, err := s.service.LoadMovies(
		ctx,
		int(req.GetFrom()),
		int(req.GetTo()),
	)
	if err != nil {
		return nil, toGRPCError(err)
	}

	response := &moviepb.LoadMoviesResponse{
		Movies: make([]*moviepb.Movie, 0, len(result.Movies)),
		Errors: make([]*moviepb.MovieError, 0, len(result.Errors)),
	}

	for _, item := range result.Movies {
		response.Movies = append(response.Movies, toProtoMovie(item))
	}

	for _, item := range result.Errors {
		response.Errors = append(response.Errors, &moviepb.MovieError{
			Id:      int32(item.ID),
			Message: item.Message,
		})
	}

	return response, nil
}

func (s *GRPCServer) GetLastMovie(
	ctx context.Context,
	req *moviepb.GetLastMovieRequest,
) (*moviepb.Movie, error) {
	if req == nil {
		return nil, status.Error(
			codes.InvalidArgument,
			"запрос не должен быть пустым",
		)
	}

	result, err := s.service.GetLastMovie(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return toProtoMovie(*result), nil
}

func toProtoMovie(item domain.Movie) *moviepb.Movie {
	return &moviepb.Movie{
		Id:       int32(item.ID),
		Title:    item.Title,
		Year:     int32(item.Year),
		Director: item.Director,
	}
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidRange):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, domain.ErrMovieNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "запрос отменён")

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(
			codes.DeadlineExceeded,
			"превышено время ожидания",
		)

	default:
		return status.Error(
			codes.Unavailable,
			"ошибка фильм-стенда: "+err.Error(),
		)
	}
}
