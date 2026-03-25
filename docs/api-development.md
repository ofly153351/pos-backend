# API Development Guide

เอกสารนี้เป็นแนวทางเพิ่ม API ใหม่ในโปรเจกต์ `pos-backend` แบบมาตรฐานเดียวกับโค้ดที่มีอยู่

## 1) เข้าใจเลเยอร์ก่อน

โครงสร้างหลักของแต่ละ feature จะอยู่ที่ `internal/modules/<feature>/`

- `model.go`: struct ของ entity / request / response
- `errors.go`: business errors ของ feature
- `repository.go`: query กับ DB (GORM/SQL), ไม่ใส่ business logic
- `service.go`: business rules, validation, permission, transaction orchestration
- `handler.go`: รับ HTTP request/response, parse input, map error เป็น status code
- `storage.go` (optional): upload file หรือ external storage เช่น MinIO
- `util.go` (optional): helper function เฉพาะ feature

## 2) ลำดับการสร้าง API (แนะนำ)

1. ออกแบบ endpoint และ role ที่เข้าถึงได้
2. เพิ่ม request/response model ใน `model.go`
3. เพิ่ม error constants ใน `errors.go`
4. เพิ่ม repository interface + implementation
5. เพิ่ม service method และ business validation
6. เพิ่ม handler method
7. register route ใน `internal/app/<feature>.go`
8. ถ้าต้องเปลี่ยน schema ให้เพิ่ม migration ใหม่ใน `init-db/`
9. อัปเดตเอกสารใน `docs/`
10. รัน test/build

## 3) ตัวอย่าง pattern (Create/Update)

### Handler

- รับค่า form/json
- เรียก service
- ส่ง response ผ่าน `httpx.Success` / `httpx.Error`

### Service

- ตรวจ required fields
- ตรวจสิทธิ์ผ่าน `UserCanManageStore` หรือ method ที่เกี่ยวข้อง
- ประมวลผล business logic
- เรียก repository เขียน/อ่านข้อมูล

### Repository

- ทำงานกับตารางโดยตรง
- คืน `Err...NotFound` เมื่อไม่พบข้อมูล
- ไม่ควรมี logic เงื่อนไข business ซับซ้อน

## 4) ถ้ามี upload file

แนวทางที่ใช้ในโปรเจกต์นี้:

- ใน handler ใช้ `c.FormFile("field_name")`
- ส่ง `*multipart.FileHeader` เข้า service
- service เรียก storage (`SaveStoreLogo`, `SaveProductImage`, etc.)
- บันทึกเฉพาะ URL/path ลง DB

## 5) Route Registration

เพิ่ม route ในไฟล์ `internal/app/<feature>.go` เช่น:

```go
protected.Post("/stores", handler.Create)
protected.Get("/stores/:storeID", handler.GetByID)
protected.Put("/stores/:storeID", handler.Update)
```

โปรเจกต์รองรับทั้ง prefix:

- `/api/v1/*`
- `/api/*` (compatibility)

## 6) Migration Rules

- เพิ่มไฟล์ใหม่แบบ forward-only ใน `init-db/` เช่น `020_add_xxx.sql`
- ไม่แก้ migration เก่าโดยตรง (ยกเว้นจำเป็นจริง)
- ใส่ `IF NOT EXISTS` / `ON CONFLICT` เมื่อเหมาะสม เพื่อลดปัญหา rerun

## 7) Definition of Done (Checklist)

- [ ] มี model/request/response ครบ
- [ ] มี validation และ permission check ใน service
- [ ] map error -> HTTP status ถูกต้อง
- [ ] query repository ตรงกับ schema ปัจจุบัน
- [ ] route ถูก register แล้ว
- [ ] docs ถูกอัปเดต
- [ ] รัน `go test ./...` ผ่าน

## 8) คำสั่งที่ใช้บ่อย

Run API:

```bash
go run ./cmd/api.go
```

Run tests:

```bash
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./...
```
