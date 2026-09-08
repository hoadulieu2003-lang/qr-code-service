# Khởi động nhanh module QR truy xuất nguồn gốc

Tài liệu này dành cho người vận hành trên Windows. Mục tiêu là mở form nhập một sản phẩm, sinh QR, rồi dùng điện thoại quét ra trang truy xuất công khai.

## 1. Điều kiện trước khi chạy

- Windows có PowerShell 5.1 trở lên.
- Có Go 1.21 trở lên. Nếu `go` chưa nằm trong `PATH`, vẫn có thể truyền đường dẫn đầy đủ đến `go.exe` khi chạy script.
- Máy tính và điện thoại cùng một mạng Wi-Fi riêng tư.
- Windows Firewall cho phép `bond.exe` trên mạng **Private** khi Windows hỏi. Không mở cổng ra Internet khi chỉ đang thử nội bộ.

## 2. Tạo cấu hình cục bộ

Tại thư mục gốc dự án, chỉ thực hiện lần đầu:

```powershell
Copy-Item .\sample.env .\.env
notepad .\.env
```

Trong `.env`, chỉnh ba giá trị sau:

```dotenv
PORT=18080
API_KEY="mot-chuoi-ngau-nhien-chi-he-thong-biet"
PUBLIC_BASE_URL="http://192.168.1.22:18080"
```

`API_KEY` là bí mật dùng cho API tích hợp, không nhập vào form web cục bộ và không gửi cho điện thoại. `PUBLIC_BASE_URL` phải là địa chỉ IPv4 của máy đang chạy module, không dùng `localhost` hoặc `127.0.0.1` vì các địa chỉ đó trên điện thoại sẽ trỏ về chính điện thoại.

Lấy IPv4 Wi-Fi bằng một trong hai lệnh sau, rồi thay giá trị `PUBLIC_BASE_URL`:

```powershell
Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.InterfaceAlias -eq 'Wi-Fi' }
ipconfig
```

Giữ `DATA_FILE="data/products.json"` nếu muốn dữ liệu thử nghiệm được lưu cục bộ. Xem toàn bộ các biến mẫu trong [sample.env](../../sample.env).

## 3. Khởi động và dừng dịch vụ

Mở PowerShell tại thư mục gốc dự án, sau đó chạy ở **cửa sổ foreground**:

```powershell
.\scripts\start-local.ps1
```

Nếu Go chưa có trong `PATH`, chỉ rõ đường dẫn `go.exe`:

```powershell
.\scripts\start-local.ps1 -GoPath 'C:\duong-dan\den\go.exe'
```

Ví dụ trong môi trường Codex hiện tại dùng Go đóng gói:

```powershell
.\scripts\start-local.ps1 -GoPath 'C:\Users\game\AppData\Local\Codex\runtimes\go1.21.4\go\bin\go.exe'
```

Script sẽ kiểm tra `.env`, build `bond.exe`, rồi chạy server ngay trong cửa sổ đó. Nhấn `Ctrl+C` trong cùng cửa sổ để dừng server. Không dùng `Start-Process` hoặc đóng cửa sổ nếu cần xem log lỗi.

## 4. Kiểm tra máy chủ và tạo QR từ form

Sau khi server khởi động, mở cửa sổ PowerShell thứ hai:

```powershell
.\scripts\check-health.ps1 -Port 18080
```

Kết quả hợp lệ là `PASS: health and local admin are ready on port 18080.`

Trên chính máy chạy server, mở [http://localhost:18080/admin](http://localhost:18080/admin). Form này:

- Không có ô API key.
- Chỉ hoạt động từ `localhost`, `127.0.0.1` hoặc `::1` để phục vụ kiểm thử cục bộ.
- Yêu cầu nhập đủ tám trường sản phẩm. `trace_code` chỉ gồm chữ, số, `_` và `-`.
- Trả về một ảnh QR và liên kết truy xuất công khai sau khi bấm **Tạo QR**.

Điện thoại phải quét QR (không mở `/admin`) khi đang cùng Wi-Fi. QR sẽ dẫn đến `http://<IPv4-may-chay>:18080/trace/<trace_code>`.

## 5. Các file theo từng máy triển khai

Các file dưới đây là dữ liệu/vận hành theo từng máy và không được đưa vào Git:

| File hoặc thư mục | Vai trò | Cần giữ/backup |
| --- | --- | --- |
| `.env` | Cấu hình và `API_KEY` riêng | Lưu an toàn, không gửi qua chat/Git |
| `data/products.json` | Danh mục sản phẩm cục bộ | Backup trước khi đổi máy hoặc cập nhật |
| `bond.exe`, `bond-verify.exe` | File build cho Windows | Có thể build lại từ source |

Source Go ở thư mục gốc, script vận hành ở `scripts/`, và hướng dẫn bàn giao ở `docs/handover/`. Không di chuyển các file Go khi vận hành hoặc bàn giao.
