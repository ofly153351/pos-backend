# API Development Guide

This document describes the standard approach for adding new APIs to the `pos-backend` project, consistent with the existing codebase patterns.

## 1) Understanding the Layers

The main structure of each feature lives in `internal/modules/<feature>/`:

- `model.go`: entity / request / response structs
- `errors.go`: business errors for the feature
- `repository.go`: database queries (GORM/SQL), no business logic
- `service.go`: business rules, validation, permission checks, transaction orchestration
- `handler.go`: handles HTTP request/response, parses input, maps errors to status codes
- `storage.go` (optional): file upload or external storage such as MinIO
- `util.go` (optional): feature-specific helper functions

## 2) Recommended API Creation Order

1. Design the endpoint and the roles that can access it
2. Add request/response models in `model.go`
3. Add error constants in `errors.go`
4. Add repository interface + implementation
5. Add service method and business validation
6. Add handler method
7. Register the route in `internal/app/<feature>.go`
8. If the schema needs to change, add a new migration in `init-db/`
9. Update documentation in `docs/`
10. Run tests/build

## 3) Pattern Examples (Create/Update)

### Handler

- Receives form/json values
- Calls the service
- Sends response via `httpx.Success` / `httpx.Error`

### Service

- Validates required fields
- Checks permissions via `UserCanManageStore` or the relevant method
- Runs business logic
- Calls the repository for reads/writes

### Repository

- Works directly with tables
- Returns `Err...NotFound` when a record is not found
- Should not contain complex business condition logic

## 4) File Uploads

The approach used in this project:

- In the handler, use `c.FormFile("field_name")`
- Pass `*multipart.FileHeader` into the service
- The service calls storage (`SaveStoreLogo`, `SaveProductImage`, etc.)
- Only the URL/path is stored in the database

## 5) Route Registration

Add routes in `internal/app/<feature>.go`, for example:

```go
protected.Post("/stores", handler.Create)
protected.Get("/stores/:storeID", handler.GetByID)
protected.Put("/stores/:storeID", handler.Update)
```

The project supports both prefixes:

- `/api/v1/*`
- `/api/*` (compatibility)

## 6) Migration Rules

- Add new files in a forward-only manner in `init-db/`, e.g. `020_add_xxx.sql`
- Do not edit existing migrations directly (unless strictly necessary)
- Use `IF NOT EXISTS` / `ON CONFLICT` where appropriate to reduce re-run issues

## 7) Definition of Done (Checklist)

- [ ] model/request/response structs are complete
- [ ] validation and permission checks exist in the service
- [ ] errors are correctly mapped to HTTP status codes
- [ ] repository queries match the current schema
- [ ] route is registered
- [ ] docs are updated
- [ ] `go test ./...` passes

## 8) Common Commands

Run API:

```bash
go run ./cmd/api.go
```

Run tests:

```bash
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./...
```
