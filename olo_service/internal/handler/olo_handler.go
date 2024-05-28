// Package handler provides gRPC handler functions for OLO service endpoints.
//
// This package includes handler functions for handling gRPC requests related to articles and widgets.
package handler

import (
	"OLO-backend/olo_service/generated"
	"OLO-backend/olo_service/internal/entity"
	"OLO-backend/olo_service/internal/mapper"
	"OLO-backend/olo_service/internal/service"
	"OLO-backend/pkg/utils/jwt"
	"context"
	"fmt"
	jwtgo "github.com/golang-jwt/jwt/v5"
)

// ArticleToArticleResponse converts an Article entity to a generated.Article.
func ArticleToArticleResponse(article entity.Article) *generated.Article {
	return &generated.Article{
		Id:     uint64(article.ID),
		Header: article.Header,
	}
}

// WidgetToWidgetResponse converts a Widget entity to a generated.Widget.
func WidgetToWidgetResponse(widget entity.Widget) *generated.Widget {
	return &generated.Widget{
		Id:          uint64(widget.ID),
		Description: widget.Description,
		Data:        widget.Data,
	}
}

// OloHandler represents the gRPC handler for OLO service endpoints.
type OloHandler struct {
	service   *service.OloService
	validator *jwt.Validator

	mapperWidget  mapper.MapFunc[entity.Widget, *generated.Widget]
	mapperArticle mapper.MapFunc[entity.Article, *generated.Article]

	generated.UnimplementedOLOServer
}

func NewOloHandler(service *service.OloService, validator *jwt.Validator) *OloHandler {
	return &OloHandler{
		service:   service,
		validator: validator,

		mapperWidget:  WidgetToWidgetResponse,
		mapperArticle: ArticleToArticleResponse,
	}
}

// getToken retrieves and validates the JWT token from the context.
func (h *OloHandler) getToken(ctx context.Context) (*jwtgo.Token, error) {
	token, err := h.validator.TokenFromContextMetadata(ctx, "Authorization")
	if err != nil {
		// todo wrap error
		return nil, fmt.Errorf("can't get token")
	}
	return token, nil
}

// newEntityUseToken extracts user information from the JWT token.
func (h *OloHandler) newEntityUseToken(token *jwtgo.Token) entity.User {
	id := int64(token.Claims.(jwtgo.MapClaims)["uid"].(float64))
	email := token.Claims.(jwtgo.MapClaims)["email"].(string)
	role := token.Claims.(jwtgo.MapClaims)["role"].(string)

	return entity.User{
		ID:    id,
		Email: email,
		Role:  role,
	}
}

func (h *OloHandler) HelloUser(ctx context.Context, _ *generated.HelloUserRequest) (*generated.HelloUserResponse, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	user := h.newEntityUseToken(token)
	return &generated.HelloUserResponse{
		Message: fmt.Sprintf(
			"Hello %s (%d)! I am the olo service. You have role %s",
			user.Email, user.ID, user.Role),
	}, nil
}

// todo add widget and article if user has role admin

func (h *OloHandler) GetAllWidgets(ctx context.Context, _ *generated.GetWidgetsRequest) (*generated.GetWidgetsResponse, error) {
	_, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	widgets, err := h.service.GetAllWidgets()
	if err != nil {
		return nil, err
	}
	return &generated.GetWidgetsResponse{
		Widgets: h.mapperWidget.MapEach(widgets),
	}, nil
}

func (h *OloHandler) GetUserWidgets(ctx context.Context, _ *generated.GetWidgetsRequest) (*generated.GetWidgetsResponse, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	user := h.newEntityUseToken(token)
	widgets, err := h.service.GetUserWidgets(user.ID)
	if err != nil {
		return nil, err
	}

	return &generated.GetWidgetsResponse{
		Widgets: h.mapperWidget.MapEach(widgets),
	}, nil
}

func (h *OloHandler) AddWidgetForUser(ctx context.Context, req *generated.WidgetForUserRequest) (*generated.WidgetForUserResponse, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	user := h.newEntityUseToken(token)
	err = h.service.AddWidgetForUser(req.WidgetId, user.ID)
	if err != nil {
		return nil, err
	}
	return &generated.WidgetForUserResponse{
		Response: fmt.Sprintf(
			"Successfully add widget (%d) for user (%d)!",
			req.WidgetId, user.ID),
	}, nil
}

func (h *OloHandler) GetUsersArticles(ctx context.Context, _ *generated.GetAllArticlesRequest) (*generated.GetAllArticlesResponse, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	user := h.newEntityUseToken(token)
	articles, err := h.service.GetUsersArticles(user.ID)
	if err != nil {
		return nil, err
	}

	return &generated.GetAllArticlesResponse{
		Articles: h.mapperArticle.MapEach(articles),
	}, nil
}

func (h *OloHandler) GetAllArticles(ctx context.Context, _ *generated.GetAllArticlesRequest) (*generated.GetAllArticlesResponse, error) {
	_, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	articles, err := h.service.GetAllArticles()
	if err != nil {
		return nil, err
	}

	return &generated.GetAllArticlesResponse{
		Articles: h.mapperArticle.MapEach(articles),
	}, nil
}

func (h *OloHandler) AddArticleForUser(ctx context.Context, req *generated.ArticleForUserRequest) (*generated.ArticleForUserResponse, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	user := h.newEntityUseToken(token)
	err = h.service.AddArticleForUser(req.ArticleId, user.ID)
	if err != nil {
		return nil, err
	}
	return &generated.ArticleForUserResponse{
		Response: fmt.Sprintf(
			"Successfully add article (%d) for user (%d)!",
			req.ArticleId, user.ID),
	}, nil
}

func (h *OloHandler) DeleteArticleForUser(ctx context.Context, req *generated.ArticleForUserRequest) (*generated.ArticleForUserResponse, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	user := h.newEntityUseToken(token)
	err = h.service.DeleteArticleForUser(req.ArticleId, user.ID)
	if err != nil {
		return nil, err
	}
	return &generated.ArticleForUserResponse{
		Response: fmt.Sprintf(
			"Successfully delete article (%d) for user (%d)!",
			req.ArticleId, user.ID),
	}, nil
}

func (h *OloHandler) DeleteWidgetForUser(ctx context.Context, req *generated.WidgetForUserRequest) (*generated.WidgetForUserResponse, error) {
	token, err := h.getToken(ctx)
	if err != nil {
		return nil, err
	}
	user := h.newEntityUseToken(token)
	err = h.service.DeleteWidgetForUser(req.WidgetId, user.ID)
	if err != nil {
		return nil, err
	}
	return &generated.WidgetForUserResponse{
		Response: fmt.Sprintf(
			"Successfully delete widget (%d) for user (%d)!",
			req.WidgetId, user.ID),
	}, nil
}
