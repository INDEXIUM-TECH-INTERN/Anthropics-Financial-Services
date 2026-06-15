// ═══════════════════════════════════════════════════
// Shared — DOM Helpers
// Replaces src/utils/dom.ts
// ═══════════════════════════════════════════════════

export function $<T extends HTMLElement>(id: string): T | null {
  return document.getElementById(id) as T | null;
}

export function escHtml(str: string): string {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

export function formatTime(dateStr?: string): string {
  if (!dateStr) return "";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return "";
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  if (diff < 60000) return "Vừa xong";
  if (diff < 3600000) return Math.floor(diff / 60000) + " phút trước";
  if (diff < 86400000) return Math.floor(diff / 3600000) + " giờ trước";
  return d.toLocaleDateString("vi-VN");
}

export function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      resolve(result.split(",")[1] ?? "");
    };
    reader.onerror = () => reject(new Error("Failed to read file"));
    reader.readAsDataURL(file);
  });
}