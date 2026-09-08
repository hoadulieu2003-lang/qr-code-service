# Cẩm Nang Tích Hợp Đa Kịch Bản (QR Code Integration Playbook)

> **Tài liệu bàn giao kỹ thuật**: Hướng dẫn copy-paste nhanh dành cho lập trình viên để nhúng module **QR Code Service** vào các nền tảng công nghệ khác nhau (Web, Mobile, Backend, In ấn).

---

## Mục lục tra cứu nhanh theo kịch bản (Use Cases)

* [Case 1: Web Frontend (HTML, React, Next.js, Vue)](#case-1-web-frontend-html-react-nextjs-vue)
* [Case 2: Backend Node.js / Express / NestJS](#case-2-backend-nodejs--express--nestjs)
* [Case 3: Python (FastAPI, Django, Flask) & Lưu Cloud S3](#case-3-python-fastapi-django-flask--lưu-cloud-s3)
* [Case 4: PHP / Laravel (Thanh toán & Hóa đơn)](#case-4-php--laravel-thanh-toán--hóa-đơn)
* [Case 5: Mobile App (Flutter, React Native)](#case-5-mobile-app-flutter-react-native)
* [Case 6: Truy Xuất Nguồn Gốc Sản Phẩm (JSON REST API)](#case-6-truy-xuất-nguồn-gốc-sản-phẩm-json-rest-api)
* [Case 7: In Ấn Tem Nhãn Bao Bì & Máy In Nhiệt](#case-7-in-ấn-tem-nhãn-bao-bì--máy-in-nhiệt)
* [Bảng mã lỗi HTTP & Giải pháp khắc phục](#bảng-mã-lỗi-http--giải-pháp-khắc-phục)

---

## Case 1: Web Frontend (HTML, React, Next.js, Vue)

Phù hợp khi cần hiển thị mã QR trực tiếp trên giao diện người dùng mà không cần code xử lý logic backend phức tạp.

### 1.1. HTML Thuần
```html
<!-- Nhúng trực tiếp URL API vào thuộc tính src -->
<div class="qr-container">
  <h3>Quét mã để mở website</h3>
  <img 
    src="http://localhost:18080/?size=300&secret=replace-with-legacy-qr-secret&content=https%3A%2F%2Fsmartmarket-six.vercel.app%2F" 
    alt="Mã QR SmartMarket" 
    width="300" 
    height="300" 
  />
</div>
```

### 1.2. React / Next.js Component
```jsx
import React from 'react';

export default function QRCodeViewer({ targetUrl, size = 300 }) {
  const QR_SERVICE_URL = process.env.NEXT_PUBLIC_QR_SERVICE_URL || 'http://localhost:18080';
  const QR_SECRET = process.env.NEXT_PUBLIC_QR_SECRET || 'replace-with-legacy-qr-secret';

  // Luôn dùng encodeURIComponent để bảo vệ các ký tự ?, &, = trong URL
  const encodedContent = encodeURIComponent(targetUrl);
  const qrImageSrc = `${QR_SERVICE_URL}/?size=${size}&secret=${QR_SECRET}&content=${encodedContent}`;

  return (
    <div className="flex flex-col items-center p-4 bg-white rounded-lg shadow">
      <img src={qrImageSrc} alt="QR Code" width={size} height={size} className="rounded" />
      <p className="mt-2 text-sm text-gray-500">Giơ camera điện thoại để quét</p>
    </div>
  );
}
```

---

## Case 2: Backend Node.js / Express / NestJS

Phù hợp cho môi trường Production: Giữ bí mật mã `SECRET` ở phía máy chủ, backend đứng ra gọi module QR qua mạng nội bộ rồi trả ảnh về cho client hoặc mã hóa sang chuỗi **Base64**.

### 2.1. Proxy ảnh PNG trực tiếp qua Express
```javascript
import express from 'express';

const app = express();
const QR_SERVICE_HOST = process.env.QR_SERVICE_HOST || 'http://localhost:18080';
const QR_SECRET = process.env.QR_SECRET || 'replace-with-legacy-qr-secret';

app.get('/api/qr', async (req, res) => {
  try {
    const { url, size = 512 } = req.query;
    if (!url) return res.status(400).json({ error: 'Thiếu tham số url' });

    // Gọi nội bộ tới QR service bằng Header bảo mật X-Bond-Secret
    const qrResponse = await fetch(
      `${QR_SERVICE_HOST}/?size=${size}&content=${encodeURIComponent(url)}`,
      {
        method: 'GET',
        headers: { 'X-Bond-Secret': QR_SECRET }
      }
    );

    if (!qrResponse.ok) {
      return res.status(qrResponse.status).send('Không thể tạo mã QR');
    }

    const imageBuffer = Buffer.from(await qrResponse.arrayBuffer());
    res.setHeader('Content-Type', 'image/png');
    res.send(imageBuffer);
  } catch (error) {
    res.status(500).json({ error: 'Lỗi máy chủ nội bộ' });
  }
});
```

### 2.2. Nhận chuỗi Base64 Data URI để nhúng vào Email hoặc PDF
```javascript
async function getQRCodeBase64(content, size = 300) {
  const res = await fetch(
    `http://localhost:18080/?size=${size}&content=${encodeURIComponent(content)}`,
    { headers: { 'X-Bond-Secret': 'replace-with-legacy-qr-secret' } }
  );
  const arrayBuffer = await res.arrayBuffer();
  const base64 = Buffer.from(arrayBuffer).toString('base64');
  return `data:image/png;base64,${base64}`;
}
```

---

## Case 3: Python (FastAPI, Django, Flask) & Lưu Cloud S3

Tự động tạo mã QR khi thêm sản phẩm mới và đẩy thẳng ảnh lên kho lưu trữ đám mây (AWS S3, Google Cloud Storage).

```python
import urllib.request
import urllib.parse
import boto3

QR_ENDPOINT = "http://localhost:18080"
SECRET_KEY = "replace-with-legacy-qr-secret"

def generate_and_upload_qr_to_s3(product_id: str, target_url: str, bucket_name: str):
    # 1. Gọi API sinh mã QR
    params = urllib.parse.urlencode({"size": 512, "content": target_url})
    url = f"{QR_ENDPOINT}/?{params}"
    
    req = urllib.request.Request(url, headers={"X-Bond-Secret": SECRET_KEY})
    with urllib.request.urlopen(req) as response:
        png_bytes = response.read()

    # 2. Đẩy thẳng mảng byte ảnh lên AWS S3
    s3_client = boto3.client('s3')
    s3_key = f"qr-codes/product_{product_id}.png"
    s3_client.put_object(
        Bucket=bucket_name,
        Key=s3_key,
        Body=png_bytes,
        ContentType='image/png'
    )
    
    public_url = f"https://{bucket_name}.s3.amazonaws.com/{s3_key}"
    print(f"Đã lưu QR thành công lên S3: {public_url}")
    return public_url
```

---

## Case 4: PHP / Laravel (Thanh toán & Hóa đơn)

Nhúng mã QR vào hóa đơn thanh toán điện tử hoặc file PDF xuất đơn hàng.

```php
<?php
namespace App\Services;

use Illuminate\Support\Facades\Http;

class QRCodeService {
    protected $baseUrl = 'http://localhost:18080';
    protected $secret = 'replace-with-legacy-qr-secret';

    public function generateOrderQR($orderId, $paymentUrl) {
        $response = Http::withHeaders([
            'X-Bond-Secret' => $this->secret
        ])->get($this->baseUrl, [
            'size' => 400,
            'content' => $paymentUrl
        ]);

        if ($response->successful()) {
            // Lưu ảnh vào thư mục storage/app/public/qrcodes/
            $path = "qrcodes/order_{$orderId}.png";
            \Storage::disk('public')->put($path, $response->body());
            return \Storage::url($path);
        }

        throw new \Exception("Không thể tạo mã QR cho đơn hàng #{$orderId}");
    }
}
```

---

## Case 5: Mobile App (Flutter, React Native)

Hiển thị mã QR trực tiếp trong ứng dụng di động cho khách hàng quét hoặc chia sẻ.

### 5.1. Flutter (Dart)
```dart
import 'package:flutter/material.dart';

class QRCodeScreen extends StatelessWidget {
  final String orderUrl = "https://smartmarket-six.vercel.app/order/123";

  @override
  Widget build(BuildContext context) {
    final encodedUrl = Uri.encodeComponent(orderUrl);
    final qrEndpoint = "http://192.168.1.20:18080/?size=512&secret=replace-with-legacy-qr-secret&content=$encodedUrl";

    return Scaffold(
      appBar: AppBar(title: Text("Mã QR Thanh Toán")),
      body: Center(
        child: Image.network(
          qrEndpoint,
          width: 250,
          height: 250,
          loadingBuilder: (ctx, child, progress) {
            if (progress == null) return child;
            return CircularProgressIndicator();
          },
        ),
      ),
    );
  }
}
```

---

## Case 6: Truy Xuất Nguồn Gốc Sản Phẩm (JSON REST API)

Dành cho hệ thống ERP / Kho hàng: Tạo thông tin bảo hành, xuất xứ và sinh đồng thời tem QR truy xuất.

* **Endpoint**: `POST /api/products`
* **Header**: `X-API-Key: replace-with-random-admin-api-key`
* **Body mẫu (JSON)**:
```json
{
  "trace_code": "SP-SMARTMARKET-001",
  "product_code": "SM-2026-X",
  "name": "Nước mắm truyền thống Phú Quốc",
  "batch_code": "BATCH-09-2026",
  "manufactured_at": "2026-09-01",
  "expires_at": "2028-09-01",
  "origin": "Phú Quốc, Kiên Giang, Việt Nam",
  "verification_status": "verified"
}
```

* **Kết quả trả về**:
```json
{
  "trace_code": "SP-SMARTMARKET-001",
  "trace_url": "http://192.168.1.20:18080/trace/SP-SMARTMARKET-001",
  "qr_url": "http://192.168.1.20:18080/api/products/SP-SMARTMARKET-001/qr.png"
}
```
*(Người tiêu dùng quét tem QR này sẽ được chuyển thẳng tới trang hiển thị xuất xứ sản phẩm).*

---

## Case 7: In Ấn Tem Nhãn Bao Bì & Máy In Nhiệt

Module trả về định dạng **PNG Monochrome (Trắng đen chuẩn)** với tỷ lệ tương phản tuyệt đối, tương thích 100% với các phần mềm thiết kế tem nhãn (Bartender, NiceLabel) hoặc máy in hóa đơn nhiệt khổ 80mm/58mm.

Lệnh tải ảnh độ nét cao để gửi xưởng in ấn:
```bash
curl -H "X-Bond-Secret: replace-with-legacy-qr-secret" \
  -o "tem_in_bao_bi_1024px.png" \
  "http://localhost:18080/?size=1024&content=https%3A%2F%2Fsmartmarket-six.vercel.app%2F"
```

---

## Bảng Mã Lỗi HTTP & Giải Pháp Khắc Phục

| Mã lỗi HTTP | Nguyên nhân | Cách khắc phục |
| :---: | :--- | :--- |
| **`400 Bad Request`** | Thiếu tham số `content` hoặc `size`<br>`size` nhập chữ hoặc số âm<br>Nội dung quá dài (> 3.000 ký tự) | Kiểm tra lại URL query params; đảm bảo `size` là số nguyên từ 1 đến 1024. |
| **`403 Forbidden`** | Sai hoặc thiếu mã `SECRET`<br>Thiếu header `X-API-Key` khi tạo sản phẩm | Kiểm tra giá trị trong file `.env`; thêm header `X-Bond-Secret` hoặc `X-API-Key`. |
| **`404 Not Found`** | Mã truy xuất `traceCode` không tồn tại trong kho | Kiểm tra lại mã sản phẩm đã được tạo qua `/api/products` chưa. |
| **`500 Internal Error`**| Lỗi hệ thống ghi đĩa hoặc máy chủ | Kiểm tra log máy chủ qua terminal. |
