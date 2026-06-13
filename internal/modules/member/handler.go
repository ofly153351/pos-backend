package member

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func (h Handler) List(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	result, err := h.service.ListMembers(c.UserContext(), middleware.ClaimsFromContext(c), storeID)
	if err != nil {
		return writeMemberError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "members fetched", result)
}

func (h Handler) Add(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	var req AddMemberRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.AddMember(c.UserContext(), middleware.ClaimsFromContext(c), storeID, req)
	if err != nil {
		return writeMemberError(c, err)
	}
	return httpx.Success(c, fiber.StatusCreated, "member added", result)
}

func (h Handler) Update(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	userID := c.Params("userID")
	var req UpdateMemberRequest
	if err := httpx.DecodeJSON(c, &req); err != nil {
		return httpx.Error(c, fiber.StatusBadRequest, "invalid request body", err.Error())
	}
	result, err := h.service.UpdateMember(c.UserContext(), middleware.ClaimsFromContext(c), storeID, userID, req)
	if err != nil {
		return writeMemberError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "member updated", result)
}

func (h Handler) Remove(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	userID := c.Params("userID")
	if err := h.service.RemoveMember(c.UserContext(), middleware.ClaimsFromContext(c), storeID, userID); err != nil {
		return writeMemberError(c, err)
	}
	return httpx.Success(c, fiber.StatusOK, "member removed", nil)
}

func writeMemberError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidRole):
		return httpx.Err422(c, "role", err.Error())
	case errors.Is(err, ErrInvalidStatus):
		return httpx.Err422(c, "status", err.Error())
	case errors.Is(err, ErrNameRequired):
		return httpx.Err422(c, "name", err.Error())
	case errors.Is(err, ErrEmailRequired):
		return httpx.Err422(c, "email", err.Error())
	case errors.Is(err, ErrPasswordTooShort):
		return httpx.Err422(c, "password", err.Error())
	case errors.Is(err, ErrStoreIDRequired), errors.Is(err, ErrNothingToUpdate):
		return httpx.ErrBadRequest(c, err.Error())
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrOwnerOnly):
		return httpx.ErrForbidden(c, err.Error())
	case errors.Is(err, ErrMemberNotFound):
		return httpx.ErrNotFound(c, err.Error())
	case errors.Is(err, ErrAlreadyMember),
		errors.Is(err, ErrLastOwner),
		errors.Is(err, ErrCannotSelfRemove),
		errors.Is(err, ErrCannotSelfSuspend):
		return httpx.ErrConflict(c, err.Error())
	default:
		log.Printf("[member] internal error: %v", err)
		return httpx.ErrInternal(c)
	}
}
