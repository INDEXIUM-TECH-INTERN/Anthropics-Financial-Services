export interface ImageLightboxController {
  open: (src: string, caption?: string) => void;
  close: () => void;
  isOpen: () => boolean;
  destroy: () => void;
}

export function createImageLightbox(): ImageLightboxController {
  const overlay = document.createElement('div');
  overlay.className = 'image-lightbox hidden';
  overlay.setAttribute('role', 'dialog');
  overlay.setAttribute('aria-modal', 'true');
  overlay.setAttribute('aria-label', 'Xem ảnh');

  const panel = document.createElement('div');
  panel.className = 'image-lightbox-panel';

  const closeBtn = document.createElement('button');
  closeBtn.type = 'button';
  closeBtn.className = 'image-lightbox-close';
  closeBtn.setAttribute('aria-label', 'Đóng');
  closeBtn.textContent = '×';

  const img = document.createElement('img');
  img.className = 'image-lightbox-img';
  img.alt = '';

  const caption = document.createElement('p');
  caption.className = 'image-lightbox-caption';

  panel.appendChild(closeBtn);
  panel.appendChild(img);
  panel.appendChild(caption);
  overlay.appendChild(panel);
  document.body.appendChild(overlay);

  let open = false;

  function close() {
    open = false;
    overlay.classList.add('hidden');
    img.removeAttribute('src');
    caption.textContent = '';
    document.body.style.overflow = '';
  }

  function openLightbox(src: string, label = '') {
    open = true;
    img.src = src;
    img.alt = label;
    caption.textContent = label;
    caption.style.display = label ? '' : 'none';
    overlay.classList.remove('hidden');
    document.body.style.overflow = 'hidden';
    closeBtn.focus();
  }

  closeBtn.addEventListener('click', close);
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay || e.target === panel) close();
  });

  return {
    open: openLightbox,
    close,
    isOpen: () => open,
    destroy: () => {
      close();
      overlay.remove();
    },
  };
}