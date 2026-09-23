const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const { join } = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const base = process.env.CAREER_QUEST_TEST_URL;

async function rolePage(role, email, script, entrypoint, expectedSelector) {
  const login = await fetch(`${base}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password: "demo" }),
    signal: AbortSignal.timeout(10000),
  });
  assert.equal(login.status, 200, `${role} login`);
  const cookie = login.headers.get("set-cookie").split(";")[0];
  try {
    const elements = new Map();
    function element(selector) {
      if (!elements.has(selector)) elements.set(selector, {
        innerHTML: "", textContent: "", value: "", scrollHeight: 0,
        classList: { add() {}, remove() {}, toggle() {} },
        addEventListener() {}, insertAdjacentHTML(position, value) { this.innerHTML += value; },
      });
      return elements.get(selector);
    }
    const context = vm.createContext({
      document: { querySelector: element, querySelectorAll: () => [], addEventListener() {} },
      window: { addEventListener() {}, scrollTo() {} },
      location: { hash: "", replace() {} },
      history: { replaceState() {} },
      localStorage: { getItem() { return null; }, setItem() {} },
      setTimeout: () => 0, clearTimeout() {}, URLSearchParams,
      scrollTo() {},
      fetch: (path, options = {}) => fetch(`${base}${path}`, {
        ...options,
        headers: { ...options.headers, Cookie: cookie },
        signal: AbortSignal.timeout(15000),
      }),
    });
    vm.runInContext(readFileSync(join(__dirname, "..", script), "utf8"), context);
    await vm.runInContext(`${entrypoint}()`, context);
    assert.ok(elements.get(expectedSelector)?.innerHTML.length > 0, `${role} dashboard did not render`);
    const toast = elements.get(role === "ld" ? "#ld-toast" : "#employee-toast");
    assert.equal(toast?.textContent || "", "", `${role} dashboard reported an error`);
  } finally {
    await fetch(`${base}/auth/logout`, {
      method: "POST", headers: { Cookie: cookie }, signal: AbortSignal.timeout(10000),
    });
  }
}

test("employee dashboard starts with live data", { skip: !base }, async () => {
  await rolePage("employee", "employee@careerquest.demo", "employee.js", "employeeInit", "#personal-hero");
});

test("L&D dashboard starts with live data", { skip: !base }, async () => {
  await rolePage("ld", "ld@careerquest.demo", "ld.js", "ldInit", "#ld-event-table");
});
