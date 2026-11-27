"use strict";

const localeButton = document.getElementById("language");
const chineseLanguage = {
  name: localeButton.textContent.trim(),
  action: localeButton.getAttribute("aria-label"),
};
const announcement = document.getElementById("announcement");
const originalTexts = new Map([...document.querySelectorAll("[data-zh]")]
  .map((element) => [element, element.dataset.en ?? element.textContent]));
const localizedAttributes = [
  ["aria-label", "data-zh-label"],
  ["content", "data-zh-content"],
].flatMap(([attribute, translation]) => [...document.querySelectorAll(`[${translation}]`)]
  .map((element) => [element, attribute, element.getAttribute(attribute), element.getAttribute(translation)]));
const browserLanguage = navigator.language.toLowerCase().startsWith("zh") ? "zh" : "en";
let language = readLanguage() || browserLanguage;

const localeListeners = [];
const text = (english, chinese) => language === "zh" ? chinese : english;

function readLanguage() {
  try {
    const stored = localStorage.getItem("goproxy-language");
    return stored === "en" || stored === "zh" ? stored : null;
  } catch {
    return null;
  }
}

function applyLanguage() {
  document.documentElement.lang = text("en", "zh-CN");
  for (const [element, english] of originalTexts) {
    element.textContent = text(english, element.dataset.zh);
  }
  for (const [element, attribute, english, chinese] of localizedAttributes) {
    element.setAttribute(attribute, text(english, chinese));
  }
  localeButton.textContent = text(chineseLanguage.name, "English");
  localeButton.setAttribute("aria-label", text(chineseLanguage.action, "Switch to English"));
  localeButton.lang = text("zh-CN", "en");
  for (const update of localeListeners) update();
}

localeButton.addEventListener("click", () => {
  language = text("zh", "en");
  try {
    localStorage.setItem("goproxy-language", language);
  } catch {
    // The language can still change for the current page.
  }
  applyLanguage();
});

window.addEventListener("pageshow", (event) => {
  if (!event.persisted) return;

  const preferredLanguage = readLanguage();
  if (!preferredLanguage || preferredLanguage === language) return;

  language = preferredLanguage;
  applyLanguage();
});

const currentNav = [...document.querySelectorAll("[data-nav]")]
  .find((link) => link.dataset.nav === document.body.dataset.page);
currentNav?.setAttribute("aria-current", "page");

async function copyValue(value, button) {
  const label = button.querySelector("span");
  let result;
  try {
    if (!navigator.clipboard) throw new Error("Clipboard unavailable");
    await navigator.clipboard.writeText(value);
    result = text("Copied", announcement.dataset.zhCopied);
  } catch {
    result = text("Couldn't copy", announcement.dataset.zhCopyFailed);
  }
  label.textContent = result;
  announcement.textContent = result;
  clearTimeout(button.copyTimer);
  button.copyTimer = setTimeout(() => {
    label.textContent = text(originalTexts.get(label), label.dataset.zh);
  }, 1800);
}

document.querySelectorAll("[data-copy]").forEach((button) => {
  button.addEventListener("click", () => {
    copyValue(document.getElementById(button.dataset.copy).textContent, button);
  });
});

function setupGuide() {
  const buttons = [...document.querySelectorAll("[data-setup]")];
  if (!buttons.length) return;

  const command = document.getElementById("setup-command");
  const note = document.getElementById("setup-note");
  const defaults = { command: command.textContent, note: note.textContent, zhNote: note.dataset.zhNote };
  let active = buttons.find((button) => button.getAttribute("aria-pressed") === "true");

  const render = () => {
    const { command: value, note: english, zhNote: chinese } = active.dataset;
    command.textContent = value ?? defaults.command;
    note.textContent = text(english ?? defaults.note, chinese ?? defaults.zhNote);
    for (const button of buttons) {
      button.setAttribute("aria-pressed", button === active);
    }
  };

  for (const button of buttons) {
    button.addEventListener("click", () => {
      active = button;
      render();
    });
  }
  localeListeners.push(render);
}

function diagnostics() {
  const button = document.getElementById("run-probe");
  if (!button) return;
  const messages = button.closest(".diagnostic-panel").dataset;
  const moduleVersion = "v0.3.0";
  const checks = Object.entries({
    module: `/golang.org/x/text/@v/${moduleVersion}.info`,
    checksum: "/sumdb/sum.golang.org/supported",
  }).map(([name, path]) => ({
    name,
    path,
    output: document.getElementById(`${name}-result`),
    dot: document.getElementById(`${name}-dot`),
    result: { state: "idle" },
  }));

  function render() {
    for (const check of checks) {
      const { state, http, time } = check.result;
      check.output.textContent = {
        idle: text("Not run", messages.zhIdle),
        running: text("Checking...", messages.zhRunning),
        ok: "HTTP " + http + " / " + time + " ms",
        http: "HTTP " + http,
        invalid: "HTTP " + http + " / " + text("Unexpected metadata", messages.zhInvalid),
        timeout: text("Timed out", messages.zhTimeout),
        failed: text("Could not read the response", messages.zhFailed),
      }[state];
      check.dot.dataset.state = ["http", "invalid", "timeout"].includes(state)
        ? "failed"
        : state;
    }
  }

  async function probe(check) {
    const start = performance.now();
    try {
      const response = await fetch(check.path, {
        cache: "no-store",
        credentials: "omit",
        signal: AbortSignal.timeout(6500),
      });
      let state = response.status === 200 ? "ok" : "http";
      if (state === "ok" && check.name === "module") {
        const metadata = await response.json().catch((error) => {
          if (error.name === "SyntaxError") return null;
          throw error;
        });
        if (metadata?.Version !== moduleVersion) state = "invalid";
      }
      check.result = {
        state,
        http: response.status,
        time: Math.round(performance.now() - start),
      };
    } catch (error) {
      check.result = { state: error.name === "TimeoutError" ? "timeout" : "failed" };
    }
    render();
  }

  button.addEventListener("click", async () => {
    if (button.getAttribute("aria-disabled") === "true") return;
    button.setAttribute("aria-disabled", "true");
    try {
      for (const check of checks) check.result = { state: "running" };
      render();
      await Promise.all(checks.map(probe));
    } finally {
      button.removeAttribute("aria-disabled");
    }
  });
  localeListeners.push(render);
}

setupGuide();
diagnostics();
applyLanguage();

for (const scroller of document.querySelectorAll(".terminal-command")) {
  const updateTabStop = () => {
    scroller.tabIndex = scroller.scrollWidth > scroller.clientWidth ? 0 : -1;
  };
  new ResizeObserver(updateTabStop).observe(scroller);
  new MutationObserver(updateTabStop).observe(scroller, {childList: true, subtree: true, characterData: true});
}
