## 🧹 Cleanup: GitHub Repository Housekeeping

### Mô tả PR này dọn dẹp repository để đảm bảo nó gọn gàng và chỉ chứa những file cần thiết cho GitHub.

### Thay đổi đã thực hiện:

#### 🗂️ Đã xóa các thư mục không cần thiết:
- `src/` - Duplicate frontend structure (đã có `frontend/src/`)
- `bin/` - Empty folder
- `scripts/` - Empty folder, contains temp scripts đã moved
- `temp-financial-services/` - Temporary project folder
- `OpenRouter/` - Duplicate provider implementation (đã có trong `Gemini/internal/providers/`)

#### 📄 Đã xóa các file không cần thiết:
- `test_gemini_keys.go` - Test file chứa sensitive data
- `C:UsersRabunoDocumentsAHihiTestAIFinancefrontendsrcsharedlibdom.ts` - Broken file path
- Các file `.tmp`, `.DS_Store`, `Thumbs.db` - System temporary files
- Các script eval Python (`eval_harness.py`, `report_generator.py`, etc.)
- Các kết quả eval cũ (`eval_results/`)

#### 📝 Cập nhật `.gitignore`:
- Thêm các patterns mới để tránh accidentally commit các file không cần thiết trong tương lai
- Bao gồm cả đường dẫn Windows absolute paths

### Files được GIU:
- ✅ `frontend/src/` - Source code chính
- ✅ `Gemini/` - Backend source code
- ✅ `docs/` - Documentation
- ✅ `.dockerignore`, `render.yaml` - Deployment configs
- ✅ `run-server.ps1` - Utility script
- ✅ Tất cả source code và configs liên quan

### ✅ Kiểm tra:
- [ ] Không ảnh hưởng đến functionality chính
- [ ] Repository trở nên gọn gàng hơn
- [ .gitignore đã cập nhật để prevent future issues

### Lợi ích:
- Repository size giảm đáng kể
- Dễ dàng navigate cho new contributors
- Giảm confusion với duplicate folders
- Security: loại bỏ sensitive test files

### Review checklist:
- [ ] Xem lại danh sách file đã xóa
- [ ] Kiểm tra functionality không bị ảnh hưởng
- [ ] Xác nhận repository sạch sẽ

---

*Cần review từ team trước khi merge*