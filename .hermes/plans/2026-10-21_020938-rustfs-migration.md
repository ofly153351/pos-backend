# เปลี่ยน Object Storage จาก MinIO เป็น RustFS

**Goal:** ใช้ RustFS สำหรับไฟล์ที่อัปโหลดใหม่ โดยไม่ย้ายหรือสนใจไฟล์เก่า

**แนวทาง:** ใช้ S3-compatible API เดิมผ่าน `minio-go` ต่อไป เปลี่ยนเฉพาะ storage configuration, Docker/deployment และตรวจ flow upload ใหม่

## Roadmap

1. **เปลี่ยน Infrastructure**
   - แก้ `docker-compose.yml` จาก MinIO เป็น RustFS
   - ตั้ง bucket และ public-read policy สำหรับไฟล์ใหม่
   - ปรับ healthcheck และ startup command

2. **เปลี่ยน Configuration**
   - เพิ่ม/เปลี่ยนค่า endpoint, access key, secret key, bucket, SSL และ public URL ให้ชี้ RustFS
   - แก้ `internal/config/config.go`, `.env`/`setup.sh`
   - จะเก็บชื่อ `MINIO_*` ไว้ชั่วคราวได้ เพื่อลดจำนวนจุดแก้ หรือเปลี่ยนเป็น `STORAGE_*` ทั้งระบบ

3. **ปรับ Deployment Scripts**
   - แก้ `start.sh` และ `golive.sh`
   - เปลี่ยนชื่อ service, healthcheck URL และข้อความสถานะจาก MinIO เป็น RustFS

4. **ตรวจ Storage Code**
   - ใช้ `PutObject`, `RemoveObject`, `BucketExists`, `MakeBucket` เดิมก่อน
   - ตรวจ 5 flow: product image, store logo, supplier logo, payment proof, warehouse attachment
   - ไม่ต้องแก้ frontend และไม่ต้องแก้ database schema

5. **ทดสอบไฟล์ใหม่**
   - Upload → เปิดดูผ่าน public URL → ลบไฟล์
   - ทดสอบ MIME type ของรูปและ PDF
   - ทดสอบ bucket ไม่มีอยู่แล้วให้ระบบสร้างได้
   - ยืนยันว่า URL ใหม่ไม่ชี้กลับไป MinIO

## ไม่ทำในงานนี้

- ไม่ migrate/copy ไฟล์เก่า
- ไม่แก้ URL เก่าใน database
- ไม่ลบไฟล์เก่าใน MinIO จนกว่าจะยืนยันว่าไม่ต้องใช้

## Verification

```text
go test ./...
ทดสอบ upload/delete จริงครบทั้ง 5 ประเภท
ตรวจ database ว่า URL ที่สร้างใหม่เป็น RustFS public URL
ตรวจ git diff และไม่เริ่ม frontend/backend server ระหว่างแก้ไข
```
