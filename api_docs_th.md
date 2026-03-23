# คู่มือการใช้งาน API สำหรับ go2rtc

เอกสารชุดนี้อธิบายวิธีการใช้งาน API สำหรับการเพิ่มและจัดการข้อมูลในระบบ go2rtc

## 1. การจัดการสตรีม (Streams Management)

ใช้สำหรับเพิ่ม แก้ไข หรือลบแหล่งข้อมูลวิดีโอ (กล้อง)

### เพิ่มหรืออัปเดตสตรีม
*   **Method:** `PUT`
*   **Endpoint:** `/api/streams`
*   **Query Parameters:**
    *   `src`: URL ของสตรีม (เช่น `rtsp://...`, `ffmpeg:...`, `http:...`)
    *   `name`: (ตัวเลือก) ชื่อเรียกสตรีม ถ้าไม่ใส่จะใช้ `src` เป็นชื่อ
    *   `type`: (ตัวเลือก) ประเภทกล้อง (เช่น `PTZ`, `Internal`)
*   **ตัวอย่างการยิง API (ต้องใช้ Token):**
    ```bash
    curl -X PUT "http://localhost:1984/api/streams?name=camera1&src=rtsp://...&type=PTZ" \
         -H "Authorization: Bearer <token>"
    ```

### ลบสตรีม
*   **Method:** `DELETE`
*   **Endpoint:** `/api/streams`
*   **Query Parameters:**
    *   `src`: ชื่อสตรีมที่ต้องการลบ
*   **ตัวอย่างการยิง API (ต้องใช้ Token):**
    ```bash
    curl -X DELETE "http://localhost:1984/api/streams?src=camera1" \
         -H "Authorization: Bearer <token>"
    ```

---

## 2. การจัดการโดเมนที่ได้รับอนุญาต (Allowed Origins / CORS)

ใช้สำหรับจัดการ Whitelist ของโดเมนที่จะอนุญาตให้ดึงสตรีมไปใช้ (CORS)

### เพิ่มโดเมน (Whitelist)
*   **Method:** `POST`
*   **Endpoint:** `/api/origins`
*   **Body (Form Data):**
    *   `origin`: ชื่อโดเมน (เช่น `http://example.com` หรือ `*`)
*   **ตัวอย่างการยิง API (ต้องใช้ Token):**
    ```bash
    curl -X POST http://localhost:1984/api/origins -d "origin=http://myweb.com" \
         -H "Authorization: Bearer <token>"
    ```

### ลบโดเมน
*   **Method:** `DELETE`
*   **Endpoint:** `/api/origins`
*   **Query Parameters:**
    *   `origin`: โดเมนที่ต้องการลบ
*   **ตัวอย่างการยิง API (ต้องใช้ Token):**
    ```bash
    curl -X DELETE "http://localhost:1984/api/origins?origin=http://myweb.com" \
         -H "Authorization: Bearer <token>"
    ```

---

## 3. การจัดการผู้ใช้งาน (User Management)

### เพิ่มผู้ใช้งานใหม่
*   **Method:** `POST`
*   **Endpoint:** `/api/users`
*   **Body (Form Data):**
    *   `username`: ชื่อผู้ใช้
    *   `password`: รหัสผ่าน
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X POST http://localhost:1984/api/users -d "username=user1&password=mypassword"
    ```

### ลบผู้ใช้งาน
*   **Method:** `DELETE`
*   **Endpoint:** `/api/users`
*   **Query Parameters:**
    *   `username`: ชื่อผู้ใช้ที่ต้องการลบ
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X DELETE "http://localhost:1984/api/users?username=user1"
    ```

---

## 4. การจัดการไฟล์ตั้งค่า (Config Management)

### บันทึกหรือแก้ไขไฟล์ go2rtc.yaml
*   **Method:** `POST` หรือ `PATCH`
*   **Endpoint:** `/api/config`
*   **Body:** เนื้อหาไฟล์ YAML
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X POST http://localhost:1984/api/config --data-binary @go2rtc.yaml \
         -H "Authorization: Bearer <token>"
    ```

---

## 5. การจัดการ API Token (Tokens Management)

ใช้สำหรับเพิ่ม/ลบ Token ที่อนุญาตให้เข้าถึง API

### ดูรายการ Token
*   **Method:** `GET`
*   **Endpoint:** `/api/tokens`
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X GET "http://localhost:1984/api/tokens" -H "Authorization: Bearer <token>"
    ```

### สร้าง Token ใหม่
*   **Method:** `POST`
*   **Endpoint:** `/api/tokens`
*   **Body (Form Data):**
    *   `name`: ชื่อหรือรายละเอียดของ Token (เช่น `HomeAssistant`)
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X POST "http://localhost:1984/api/tokens" -d "name=MyIntegration" -H "Authorization: Bearer <token>"
    ```
    ✅ **ผลลัพธ์:** จะส่งค่ากลับมาเป็น JSON มี property `token` (เช่น `{"status":"ok", "token":"<base64_string>"}`) ซึ่งใช้ดูได้ครั้งเดียว

### ปิด/เปิด การใช้งาน Token (Toggle Active)
*   **Method:** `PATCH`
*   **Endpoint:** `/api/tokens`
*   **Query/Form Data:**
    *   `id`: รหัสลำดับ ID ของ Token
    *   `is_active`: `true` หรือ `false`
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X PATCH "http://localhost:1984/api/tokens" -d "id=1&is_active=false" -H "Authorization: Bearer <token>"
    ```

### ลบ Token
*   **Method:** `DELETE`
*   **Endpoint:** `/api/tokens`
*   **Query/Form Data:**
    *   `id`: รหัสลำดับ ID ของ Token ที่ต้องการลบ
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X DELETE "http://localhost:1984/api/tokens?id=1" -H "Authorization: Bearer <token>"
    ```

---

## 6. การจัดการประเภทกล้อง (Camera Types Management)

ใช้สำหรับจัดการรายการประเภทกล้องที่มีให้เลือกในระบบ

### ดูรายการประเภทกล้อง
*   **Method:** `GET`
*   **Endpoint:** `/api/types`
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X GET "http://localhost:1984/api/types"
    ```

### เพิ่มประเภทกล้องใหม่
*   **Method:** `POST`
*   **Endpoint:** `/api/types`
*   **Body (Form Data):**
    *   `name`: ชื่อประเภทกล้อง (เช่น `PTZ`)
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X POST "http://localhost:1984/api/types" -d "name=External"
    ```

### ลบประเภทกล้อง
*   **Method:** `DELETE`
*   **Endpoint:** `/api/types`
*   **Query Parameters:**
    *   `id`: รหัส ID ของประเภทที่ต้องการลบ
*   **ตัวอย่างการยิง API:**
    ```bash
    curl -X DELETE "http://localhost:1984/api/types?id=1"
    ```

> [!NOTE]
> ระบบต้องการให้ทุกคนที่ยิง API (เช่น ขจัดการสตรีม, ค่าโดเมน, จัดการ Token) เพื่อเพิ่ม/ลบ ข้อมูล ต้องส่ง Header `Authorization: Bearer <token>` มาด้วยทุกครั้ง (หรือใช้ช่องทาง Basic Auth ร่วมกับผู้ใช้/รหัสผ่าน)
