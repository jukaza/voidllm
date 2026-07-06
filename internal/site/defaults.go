package site

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

func defaultFooter(name string) string {
	return name + " — OpenAI-compatible API marketplace"
}

func defaultUserAgreement(name string) string {
	return strings.ReplaceAll(userAgreementTemplate, "{{SYSTEM_NAME}}", name)
}

func defaultPrivacyPolicy(name string) string {
	return strings.ReplaceAll(privacyPolicyTemplate, "{{SYSTEM_NAME}}", name)
}

func defaultHomePageContent(name string) string {
	return strings.ReplaceAll(homePageContentTemplate, "{{SYSTEM_NAME}}", name)
}

func defaultAnnouncements(name string) []Announcement {
	zaloURL := "https://zalo.me/g/tavo-hotro"
	qrURL := "https://api.qrserver.com/v1/create-qr-code/?size=240x240&data=" + url.QueryEscape(zaloURL)
	content := fmt.Sprintf(`## 🎉 Khai trường — Chào mừng đến với **%s**

Chúng tôi chính thức mở cổng API marketplace. Ưu đãi dành cho khách hàng mới:

- **Tặng 50.000đ** khi đăng ký và nạp lần đầu
- Giảm *10%%* phí API trong **7 ngày** đầu

---

**Cần hỗ trợ?** Tham gia nhóm Zalo:

👉 [Nhóm Zalo hỗ trợ khách hàng](%s)

Quét mã QR bên dưới để vào nhóm nhanh:

![Quét Zalo để được hỗ trợ 24/7](%s)`, name, zaloURL, qrURL)

	return []Announcement{{
		ID:          uuid.NewString(),
		Content:     content,
		PublishDate: time.Now().UTC().Format(time.RFC3339),
		Type:        "success",
		Extra:       "*Hỗ trợ trong giờ hành chính: 8h–22h mỗi ngày.*",
	}}
}

func defaultAnnouncementsJSON(name string) (string, error) {
	return marshalAnnouncements(defaultAnnouncements(name))
}

const userAgreementTemplate = `# THỎA THUẬN DỊCH VỤ NGƯỜI DÙNG

Chào mừng bạn đến với {{SYSTEM_NAME}}. Trước khi sử dụng dịch vụ, vui lòng đọc kỹ các điều khoản dưới đây. Việc đăng ký tài khoản và sử dụng dịch vụ đồng nghĩa với việc bạn đồng ý tuân thủ các điều khoản này.

### 1. Quy định về Tài khoản
* **Đăng ký**: Người dùng cần cung cấp địa chỉ email hợp lệ. Bạn chịu trách nhiệm về tính chính xác của thông tin đã cung cấp.
* **Bảo mật**: Bạn có trách nhiệm bảo mật tài khoản, mật khẩu và các API key do hệ thống cấp. Chúng tôi không chịu trách nhiệm cho tổn thất phát sinh từ việc lộ thông tin tài khoản hoặc API key.
* **Hành vi bị cấm**: Nghiêm cấm sử dụng tài khoản để tấn công DDoS, dò quét lỗ hổng, khai thác hệ thống hoặc các hành vi phá hoại khác.

### 2. Quy định về Số dư và Nạp tiền
* **Cơ chế tính phí**: Hệ thống trừ số dư theo mức tiêu hao thực tế (pay-as-you-go) dựa trên token hoặc số lần gọi API, theo bảng giá công bố.
* **Chính sách hoàn tiền**: Giao dịch nạp tiền thành công **không được hoàn lại** thành tiền mặt, trừ lỗi phát sinh từ phía hệ thống mà không thể khắc phục.
* **Số dư**: Số dư không có thời hạn hết hạn trừ khi tài khoản bị khóa do vi phạm điều khoản.

### 3. Quy định Sử dụng API
Bạn cam kết không sử dụng API để gửi nội dung:
* Vi phạm pháp luật Việt Nam.
* Kích động bạo lực, khiêu dâm, thông tin giả mạo hoặc vi phạm thuần phong mỹ tục.
* Phục vụ lừa đảo, tấn công mạng hoặc phần mềm độc hại.
* Vi phạm chính sách nội dung của nhà cung cấp gốc (OpenAI, Anthropic, Google, v.v.).

### 4. Giới hạn Trách nhiệm
* {{SYSTEM_NAME}} là cổng kết nối (gateway) trung gian. Chúng tôi không chịu trách nhiệm về chất lượng câu trả lời hoặc thông tin sai lệch do mô hình AI tạo ra.
* Chúng tôi được miễn trừ trách nhiệm trong trường hợp gián đoạn dịch vụ bất khả kháng (sự cố nhà cung cấp gốc, hạ tầng mạng nằm ngoài tầm kiểm soát).

### 5. Đình chỉ Dịch vụ
Chúng tôi có quyền tạm ngừng hoặc khóa tài khoản nếu phát hiện vi phạm chính sách, gian lận thanh toán, tấn công hệ thống hoặc số dư âm vượt giới hạn cho phép.
`

const privacyPolicyTemplate = `# CHÍNH SÁCH BẢO MẬT THÔNG TIN

Sự riêng tư và bảo mật dữ liệu của bạn là ưu tiên hàng đầu của {{SYSTEM_NAME}}.

### 1. Thông tin Thu thập
Chúng tôi chỉ thu thập thông tin tối thiểu cần thiết:
* **Tài khoản**: Email dùng để đăng ký và nhận diện tài khoản.
* **Giao dịch**: Lịch sử nạp tiền, số dư và mã giao dịch liên quan.
* **Metadata API**: Thời gian gọi, model sử dụng, số token, độ trễ và mã lỗi HTTP (nếu có) — phục vụ tính phí và vận hành.

### 2. Cam kết Bảo mật Dữ liệu API
* **Không lưu nội dung**: Hệ thống hoạt động theo cơ chế proxy/relay và **không lưu** nội dung prompt hoặc response trong cơ sở dữ liệu.
* **Truyền tải mã hóa**: Dữ liệu được truyền qua HTTPS từ máy khách tới máy chủ và chuyển tiếp tới nhà cung cấp gốc.
* **Không huấn luyện AI**: Dữ liệu gửi qua API không được dùng để huấn luyện mô hình AI.

### 3. Sử dụng và Chia sẻ
* Thông tin chỉ dùng để vận hành hệ thống, tính phí và hỗ trợ kỹ thuật khi bạn yêu cầu.
* Chúng tôi **không bán hoặc chia sẻ** email và thông tin tài khoản cho bên thứ ba vì mục đích thương mại.

### 4. Thay đổi Chính sách
Chúng tôi có thể cập nhật chính sách này theo thời gian. Phiên bản mới nhất luôn được hiển thị công khai trên trang này.
`

const homePageContentTemplate = `# HƯỚNG DẪN KẾT NỐI API

{{SYSTEM_NAME}} tương thích với định dạng API OpenAI. Chỉ cần thay **Base URL** và dùng **API Key** của bạn.

### Bước 1: Lấy API Key và nạp số dư
1. Đăng ký tài khoản bằng email.
2. Vào **API Keys** và tạo key mới.
3. Vào **Wallet** để nạp số dư.

### Bước 2: Thông tin kết nối
* **Base URL**: ` + "`https://your-domain.com/v1`" + ` (thay bằng tên miền của bạn)
* **API Key**: Token bạn vừa tạo.

### Python (OpenAI SDK)
` + "```python\nfrom openai import OpenAI\n\nclient = OpenAI(\n    base_url=\"https://your-domain.com/v1\",\n    api_key=\"sk-...\"\n)\n\ncompletion = client.chat.completions.create(\n    model=\"gpt-4o\",\n    messages=[{\"role\": \"user\", \"content\": \"Hello!\"}]\n)\nprint(completion.choices[0].message.content)\n```" + `
`