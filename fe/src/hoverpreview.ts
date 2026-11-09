// Полное фото по наведению: элемент с data-full="<url>" показывает миниатюру, а при наведении
// рядом всплывает полное фото (пока оно грузится — растянутая миниатюра). На тач-устройствах выключено.

const DELAY_MS = 250;
const GAP = 16;

export function initHoverPreview(): void {
  if (!matchMedia("(hover: hover) and (pointer: fine)").matches) return;

  const $pop = document.createElement("div");
  $pop.className = "hp";
  $pop.hidden = true;
  $pop.innerHTML = `<img class="hp__img" alt="" />`;
  document.body.append($pop);
  const $img = $pop.querySelector<HTMLImageElement>(".hp__img")!;

  let timer: number | undefined;
  let current: HTMLElement | null = null;

  const hide = () => {
    clearTimeout(timer);
    current = null;
    $pop.hidden = true;
    $pop.classList.remove("is-shown");
  };

  function show(el: HTMLElement): void {
    const full = el.dataset.full!;
    const thumb = el.querySelector("img")?.currentSrc ?? "";
    $pop.style.backgroundImage = thumb ? `url("${thumb}")` : "";
    $img.classList.remove("is-loaded");
    $img.onload = () => $img.classList.add("is-loaded");
    $img.src = full;

    // Место: справа от элемента, если не влезает — слева; по вертикали — в пределах окна.
    const r = el.getBoundingClientRect();
    const w = Math.min(720, Math.max(r.left, innerWidth - r.right) - GAP * 2);
    const h = Math.min(w / 1.5, innerHeight - GAP * 2);
    const width = h * 1.5;
    const left = innerWidth - r.right >= width + GAP * 2 ? r.right + GAP : Math.max(GAP, r.left - GAP - width);
    const top = Math.min(Math.max(GAP, r.top + r.height / 2 - h / 2), innerHeight - h - GAP);
    Object.assign($pop.style, { left: `${left}px`, top: `${top}px`, width: `${width}px`, height: `${h}px` });
    $pop.hidden = false;
    requestAnimationFrame(() => $pop.classList.add("is-shown"));
  }

  document.addEventListener("mouseover", (e) => {
    const el = (e.target as Element).closest<HTMLElement>("[data-full]");
    if (el === current) return;
    hide();
    if (!el?.dataset.full) return;
    current = el;
    timer = window.setTimeout(() => current === el && show(el), DELAY_MS);
  });
  document.addEventListener("scroll", hide, { passive: true, capture: true });
  addEventListener("blur", hide);
  addEventListener("popstate", hide);
  document.addEventListener("click", hide);
}
