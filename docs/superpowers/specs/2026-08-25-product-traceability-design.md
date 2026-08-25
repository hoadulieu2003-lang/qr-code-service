# MVP truy xuất nguồn gốc sản phẩm bằng QR

## Trạng thái và mục tiêu

Thiết kế đã được duyệt ngày 25-08-2026. Mục tiêu của MVP là cho phép một hệ thống nội bộ tạo một sản phẩm qua API, nhận lại QR PNG, sau đó người dùng dùng camera điện thoại quét QR để xem một trang truy xuất nguồn gốc công khai.

MVP phải hiển thị sáu trường đã thống nhất: tên sản phẩm, mã sản phẩm/lô (cùng một dòng), ngày sản xuất, hạn dùng, nơi sản xuất và trạng thái xác thực.

## Phạm vi

Bao gồm:

- Lưu một danh mục sản phẩm nhỏ trong tệp JSON cục bộ.
- API có xác thực để tạo sản phẩm và lấy lại QR PNG.
- URL công khai cố định theo mã truy xuất.
- Trang HTML tối ưu cho màn hình điện thoại khi quét QR.
- Một sản phẩm demo để kiểm tra trực tiếp bằng điện thoại trên cùng mạng Wi-Fi.

Không bao gồm analytics lượt quét, tài khoản người dùng, chỉnh sửa/xóa sản phẩm, nhiều tenant, cơ sở dữ liệu bên ngoài, hoặc chứng nhận pháp lý. Những phần đó là giai đoạn sau khi MVP quét được thật.

## Lựa chọn kiến trúc

QR chứa một URL công khai, không chứa toàn bộ dữ liệu truy xuất. Ví dụ:

```text
http://192.168.1.20:18080/trace/SP-DEMO-001
```

Điện thoại mở URL này và server trả trang thông tin. Vì QR chỉ giữ URL, thông tin hiển thị có thể cập nhật trong tương lai mà không cần in lại mã. URL phải dùng `PUBLIC_BASE_URL` trong cấu hình; không bao giờ đưa `127.0.0.1` vào QR vì điện thoại không truy cập được loopback của máy tính.

MVP mở rộng Bond thay vì thay thế engine QR hiện có:

```text
Hệ thống nội bộ --POST /api/products--> API quản trị --ghi--> products.json
                                               |                 |
                                               v                 v
                                           QR PNG          dữ liệu đã xác thực
                                               |
Điện thoại --quét QR--> GET /trace/{trace_code} --trả--> trang truy xuất công khai
```

## Thành phần và trách nhiệm

| Thành phần | Trách nhiệm |
| --- | --- |
| `main.go` | Nạp cấu hình, khai báo router, health check và khởi động HTTP server. |
| `product.go` | Kiểu dữ liệu sản phẩm, chuẩn hóa và xác thực đầu vào. |
| `store.go` | Đọc/ghi `data/products.json`, khóa ghi trong tiến trình, kiểm tra mã truy xuất là duy nhất và ghi nguyên tử. |
| `trace_page.go` | API quản trị, tạo PNG QR và render trang truy xuất bằng `html/template`. |
| `data/products.json` | Dữ liệu MVP cục bộ; không commit dữ liệu thật hoặc secret. |

## Dữ liệu sản phẩm

Mỗi sản phẩm có cấu trúc sau:

```json
{
  "trace_code": "SP-DEMO-001",
  "product_code": "SP-001",
  "name": "Cà phê rang xay Demo",
  "batch_code": "LO-2026-001",
  "manufactured_at": "2026-08-25",
  "expires_at": "2027-08-25",
  "origin": "Đắk Lắk, Việt Nam",
  "verification_status": "verified"
}
```

Quy tắc:

- `trace_code` là duy nhất và chỉ gồm chữ, số, dấu gạch ngang hoặc gạch dưới.
- `name`, `product_code`, `batch_code` và `origin` là bắt buộc, sau khi trim không được rỗng.
- Hai ngày dùng chuẩn `YYYY-MM-DD`; hạn dùng không được trước ngày sản xuất.
- `verification_status` chỉ nhận `verified` hoặc `unverified`.
- Trang công khai hiển thị `product_code` và `batch_code` chung thành trường “Mã sản phẩm / lô”; với năm trường còn lại tạo thành đúng sáu trường hiển thị.
- Chỉ các trường trên được công khai trên trang truy xuất. Không lưu hay hiển thị secret, giá nhập, thông tin nhân viên hoặc dữ liệu khách hàng.

## Cấu hình

Các biến môi trường mới trong `.env`:

| Biến | Ý nghĩa | Ví dụ local để quét bằng điện thoại |
| --- | --- | --- |
| `API_KEY` | Khóa cho các API quản trị, truyền bằng header. | Chuỗi ngẫu nhiên dài. |
| `PUBLIC_BASE_URL` | Gốc URL được nhúng vào QR. | `http://192.168.1.20:18080` |
| `DATA_FILE` | Đường dẫn JSON lưu dữ liệu. | `data/products.json` |

`SECRET` của Bond cũ giữ lại để tương thích endpoint sinh QR chung. Các API sản phẩm mới dùng `X-API-Key`; không đặt khóa trong query string.

Server từ chối khởi động nếu `API_KEY` rỗng. `DATA_FILE` mặc định là `data/products.json`. `PUBLIC_BASE_URL` phải là URL `http` hoặc `https` không có dấu gạch chéo cuối; nếu không hợp lệ hoặc rỗng, API tạo sản phẩm trả `503` thay vì phát hành QR không thể quét được.

## Hợp đồng API

### Tạo sản phẩm và nhận URL QR

`POST /api/products`

Header bắt buộc:

```text
X-API-Key: <API_KEY>
Content-Type: application/json
```

Body là dữ liệu sản phẩm ở phần trên. `trace_code` là bắt buộc trong MVP để hệ thống gọi API chủ động quản lý mã.

Phản hồi `201 Created`:

```json
{
  "trace_code": "SP-DEMO-001",
  "trace_url": "http://192.168.1.20:18080/trace/SP-DEMO-001",
  "qr_url": "http://192.168.1.20:18080/api/products/SP-DEMO-001/qr.png"
}
```

Trả `401` nếu khóa sai hoặc thiếu, `400` nếu JSON/trường không hợp lệ, `409` nếu `trace_code` đã tồn tại, và `503` nếu `PUBLIC_BASE_URL` chưa được cấu hình.

### Lấy PNG QR để in

`GET /api/products/{trace_code}/qr.png`

Header `X-API-Key` bắt buộc. Phản hồi `200` với `Content-Type: image/png`, error-correction mức High và kích thước mặc định 512 px. Mã QR luôn encode đúng `trace_url` đã trả khi tạo sản phẩm.

### Trang truy xuất công khai

`GET /trace/{trace_code}`

Không yêu cầu API key. Trả `200` HTML responsive có tiêu đề “Truy xuất nguồn gốc”, sáu trường dữ liệu và nhãn trạng thái dễ đọc. Mã không tồn tại trả `404` với trang HTML không làm lộ chi tiết hệ thống.

### Health check

`GET /health` tiếp tục trả `200` như hiện tại.

## An toàn và xử lý lỗi

- API quản trị chỉ chấp nhận `POST` JSON và so sánh `X-API-Key` bằng thời gian hằng.
- Trang công khai dùng `html/template` để escape nội dung sản phẩm, không nối dữ liệu thô vào HTML.
- Từ chối `trace_code` sai định dạng trước khi truy cập tệp.
- Ghi JSON qua tệp tạm rồi đổi tên để tránh một tệp nửa chừng khi ghi lỗi.
- API không ghi `X-API-Key` hoặc toàn bộ header vào log.
- Chạy LAN chỉ dùng dữ liệu demo. Khi đưa internet, bắt buộc dùng HTTPS, domain cố định và reverse proxy; không dùng địa chỉ IP LAN trong QR in thật.

## Kịch bản kiểm thử nghiệm thu

1. Cấu hình `PUBLIC_BASE_URL` bằng IPv4 LAN của máy đang chạy service và đặt điện thoại cùng Wi-Fi.
2. Gọi `POST /api/products` với sản phẩm demo `SP-DEMO-001`, nhận `201` và URL truy xuất.
3. Tải `GET /api/products/SP-DEMO-001/qr.png`, kiểm tra ảnh là PNG, rồi giải mã ngược để nội dung bằng chính `trace_url`.
4. Mở ảnh QR trên màn hình máy tính hoặc in ra; dùng camera điện thoại quét.
5. Điện thoại mở `/trace/SP-DEMO-001` và hiển thị đúng sáu trường cùng trạng thái `verified`.
6. Thử API key sai nhận `401`, mã trùng nhận `409`, mã công khai không tồn tại nhận `404`.

MVP được chấp nhận khi sáu bước trên hoàn tất với bằng chứng request/response và lần quét bằng điện thoại thật.
