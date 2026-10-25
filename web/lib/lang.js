// The language of the games: Polish (the default) or English, chosen with
// two flags and kept in a cookie (in localStorage too, as a page opened from
// disk can't keep cookies). Loaded in <head>, before a game's own script,
// which reads GryLang.lang once: changing the language reloads the page.
//
//   GryLang.lang      "pl" or "en"
//   GryLang.flags()   the HTML of the two flags, e.g. for a game's menu
//   GryLang.set(l)    save another language and reload
//
// Also in the HTML: <div data-gry-flags></div> gets the flags, and an element
// with data-en="..." gets that text in English (data-en-title,
// data-en-aria-label and data-en-placeholder its attributes). ?lang=en in
// the address sets the language too.
(function () {
  "use strict";
  const LANGS = ["pl", "en"];
  const KEY = "gry_lang";

  function stored() {
    const m = document.cookie.match(/(?:^|;\s*)gry_lang=(pl|en)\b/);
    if (m) return m[1];
    try {
      const v = localStorage.getItem(KEY);
      if (LANGS.includes(v)) return v;
    } catch (e) { /* storage unavailable */ }
    return null;
  }
  function save(lang) {
    document.cookie = KEY + "=" + lang + "; path=/; max-age=31536000; SameSite=Lax";
    try { localStorage.setItem(KEY, lang); } catch (e) { /* storage unavailable */ }
  }

  let lang = stored() || "pl";
  const param = new URLSearchParams(location.search).get("lang");
  if (LANGS.includes(param)) {
    lang = param;
    save(lang);
  }
  document.documentElement.lang = lang;

  // Pixel-art flags, crisp at any size like the rest of the games.
  // Both 2:1, so they fill their buttons alike.
  const FLAG_PL = '<svg viewBox="0 0 10 5" aria-hidden="true"><rect width="10" height="5" fill="#fff"/><rect y="2.5" width="10" height="2.5" fill="#dc143c"/></svg>';
  const FLAG_EN = '<svg viewBox="0 0 60 30" aria-hidden="true"><clipPath id="gry-uk"><path d="M30,15h30v15zv15h-30zh-30v-15zv-15h30z"/></clipPath>' +
    '<path d="M0,0v30h60v-30z" fill="#012169"/><path d="M0,0L60,30M60,0L0,30" stroke="#fff" stroke-width="6"/>' +
    '<path d="M0,0L60,30M60,0L0,30" clip-path="url(#gry-uk)" stroke="#c8102e" stroke-width="4"/>' +
    '<path d="M30,0v30M0,15h60" stroke="#fff" stroke-width="10"/><path d="M30,0v30M0,15h60" stroke="#c8102e" stroke-width="6"/></svg>';

  function flags() {
    const button = (l, svg, label) =>
      '<button type="button" data-gry-lang="' + l + '" aria-pressed="' + (l === lang) + '" title="' + label + '" aria-label="' + label + '">' + svg + "</button>";
    return '<div class="gry-lang">' + button("pl", FLAG_PL, "Polski") + button("en", FLAG_EN, "English") + "</div>";
  }

  function set(l) {
    if (!LANGS.includes(l)) return;
    save(l);
    if (l === lang) return;
    // drop ?lang= so the saved choice wins after the reload
    const url = new URL(location.href);
    url.searchParams.delete("lang");
    location.replace(url.href);
  }

  // The games handle clicks, touches and keys on their whole screen (e.g. any
  // key starts a game), so a press on a flag is caught first and goes no
  // further.
  const onFlag = e => e.target instanceof Element && e.target.closest("[data-gry-lang]");
  for (const type of ["pointerdown", "mousedown", "touchstart", "keydown", "keyup"]) {
    window.addEventListener(type, e => {
      if (onFlag(e)) e.stopPropagation();
    }, true);
  }
  window.addEventListener("click", e => {
    const b = onFlag(e);
    if (!b) return;
    e.stopPropagation();
    e.preventDefault();
    set(b.dataset.gryLang);
  }, true);

  // [data-lang] is a text in one language, which a game hides when the page is in the other one - also
  // where the game's own CSS gives it a display
  const css = "[data-lang][hidden]{display:none!important}" +
    ".gry-lang{display:flex;justify-content:center;gap:12px;margin:10px 0 0}" +
    ".gry-lang button{width:48px;height:30px;padding:3px;margin:0;border:3px solid transparent;background:#000;cursor:pointer;opacity:.55;line-height:0}" +
    ".gry-lang button[aria-pressed=true]{border-color:#ffd800;opacity:1}" +
    ".gry-lang button:hover,.gry-lang button:focus-visible{opacity:1;outline:none;border-color:#fff}" +
    ".gry-lang svg{width:100%;height:100%;display:block}";

  function apply() {
    const style = document.createElement("style");
    style.textContent = css;
    document.head.appendChild(style);
    for (const el of document.querySelectorAll("[data-gry-flags]")) el.innerHTML = flags();
    if (lang !== "en") return;
    for (const el of document.querySelectorAll("[data-en]")) el.textContent = el.dataset.en;
    for (const attr of ["title", "aria-label", "placeholder"]) {
      for (const el of document.querySelectorAll("[data-en-" + attr + "]")) el.setAttribute(attr, el.getAttribute("data-en-" + attr));
    }
    const title = document.querySelector("title[data-en]");
    if (title) document.title = title.dataset.en;
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", apply);
  else apply();

  window.GryLang = { lang, flags, set };
})();
