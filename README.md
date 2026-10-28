# gocode — AI Coding Agent for Termux

وكيل ذكاء اصطناعي متخصص في هندسة البرمجيات، مكتوب بالكامل بلغة Go، يعمل حصراً في Termux (Android ARM64).

## الميزات الحالية

- Agent Loop مع Tool Calling
- قراءة وكتابة وتعديل الملفات (مع Sandbox وحماية الروابط الرمزية)
- بحث في المشروع + ملخص البنية (`project_summary`)
- تعديل جراحي (`edit_file`) مع معاينة ملونة
- تشغيل الاختبارات والبناء (`go_test`, `go_build`)
- تكامل Git آمن (`git_status`, `git_diff`, `git_add`, `git_commit`)
- ذاكرة وجلسات محلية (`remember`, `-continue`, `/sessions`)
- Self-Correction عند فشل الاختبارات
- بدون مكتبات خارجية (Pure Go)

## التثبيت على Termux

```bash
pkg install golang git
git clone https://github.com/amiraq1/gocode.git
cd gocode
chmod +x build.sh
go test ./...
./build.sh install

export GOCODE_API_KEY="your-key"
export GOCODE_BASE_URL="https://openrouter.ai/api/v1"
export GOCODE_MODEL="deepseek/deepseek-chat"

gocode
```

## الأوامر داخل الجلسة

- `/help` — المساعدة
- `/sessions` — عرض الجلسات المحفوظة
- `/load <id>` — استئناف جلسة
- `/memory` — عرض ذاكرة المشروع
- `/reset` — جلسة جديدة
- `/exit` — خروج

## ملاحظات

- الجلسات والذاكرة تُحفظ محلياً في `~/.config/gocode/` (تعمل بدون إنترنت).
- كل عملية كتابة أو تنفيذ تتطلب موافقة صريحة.
- لا يوجد `git push` افتراضياً.
