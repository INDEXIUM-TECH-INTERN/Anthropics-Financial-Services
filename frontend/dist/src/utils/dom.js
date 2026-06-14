export function $(id) {
    return document.getElementById(id);
}
export function escHtml(s) {
    const d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
}
export function formatTime(iso) {
    if (!iso)
        return '';
    try {
        const d = new Date(iso);
        const diff = (Date.now() - d.getTime()) / 1000;
        if (diff < 60)
            return 'vừa xong';
        if (diff < 3600)
            return `${Math.floor(diff / 60)}p`;
        if (diff < 86400)
            return `${Math.floor(diff / 3600)}h`;
        return d.toLocaleDateString('vi-VN', { day: 'numeric', month: 'short' });
    }
    catch {
        return '';
    }
}
export function readFileAsBase64(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => { resolve(reader.result.split(',')[1] ?? ''); };
        reader.onerror = reject;
        reader.readAsDataURL(file);
    });
}
