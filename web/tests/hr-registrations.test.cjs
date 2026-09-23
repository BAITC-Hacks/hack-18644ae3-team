const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const { join } = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const source = readFileSync(join(__dirname, "..", "app.js"), "utf8");

function workspace(respond) {
  const elements = new Map();
  function element(selector) {
    if (!elements.has(selector)) elements.set(selector, {
      innerHTML: "", textContent: "", value: "", disabled: false,
      classList: { toggle() {}, add() {}, remove() {} },
      listeners: {},
      addEventListener(name, handler) { this.listeners[name] = handler; },
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
    respond,
  });
  vm.runInContext(source, context);
  vm.runInContext("api = (path, options) => respond(path, options)", context);
  return { run: (code) => vm.runInContext(code, context), element };
}

const requests = [
  { id: "U1", name: "First applicant", email: "first@example.test", status: "PENDING", requested_employee_id: "E7777" },
  { id: "U2", name: "Second applicant", email: "second@example.test", status: "PENDING" },
];

test("courses without skill rewards do not crash HR rendering", () => {
  const page = workspace(() => {});
  page.run(`state.events = [{ event_id: "EV_001", title: "Compliance course", type: "compliance", mandatory: true, description: "Required activity", duration_hours: 1, format: "online", develops_skills: null }]; renderHREvents();`);
  assert.match(page.element("#hr-event-grid").innerHTML, /Compliance course/);
});

test("pending requests render without loading an employee profile", async () => {
  const page = workspace(async (path, options) => {
    assert.equal(path, "/registrations");
    assert.equal(options.cache, "no-store");
    return { registrations: requests };
  });
  await page.run("loadRegistrations()");
  assert.equal(page.run("state.profile"), null);
  assert.equal(page.element("#registration-nav-count").textContent, 2);
  assert.match(page.element("#registration-table").innerHTML, /First applicant/);
  assert.match(page.element("#registration-table").innerHTML, /Second applicant/);
  assert.match(page.element("#registration-table").innerHTML, /E7777/);
});

test("opening requests again fetches newly submitted applications", async () => {
  let calls = 0;
  const page = workspace(async () => ({ registrations: calls++ === 0 ? [] : requests }));
  page.run('state.user = { role: "hr" }; showView("registrations")');
  await new Promise(setImmediate);
  assert.match(page.element("#registration-table").innerHTML, /No pending registrations/);
  page.run('showView("overview"); showView("registrations")');
  await new Promise(setImmediate);
  assert.equal(calls, 2);
  assert.match(page.element("#registration-table").innerHTML, /Second applicant/);
});

test("request failures show an error and allow retry", async () => {
  let calls = 0;
  const page = workspace(async () => {
    if (calls++ === 0) throw new Error("Server unavailable");
    return { registrations: requests };
  });
  await page.run("loadRegistrations()");
  assert.match(page.element("#registration-table").innerHTML, /Could not load registration requests/);
  assert.doesNotMatch(page.element("#registration-table").innerHTML, /No pending registrations/);
  assert.equal(page.element("#refresh-registrations").disabled, false);
  await page.run("loadRegistrations()");
  assert.equal(page.element("#registration-nav-count").textContent, 2);
});

test("initial request load survives failure of unrelated dashboard data", async () => {
  const page = workspace(async (path) => {
    if (path === "/auth/me") return { user: { role: "hr" } };
    if (path === "/registrations") return { registrations: requests };
    throw new Error("Employee data unavailable");
  });
  await page.run("init()");
  assert.match(page.element("#registration-table").innerHTML, /First applicant/);
  assert.equal(page.element("#registration-nav-count").textContent, 2);
});

test("an older refresh cannot overwrite a newer request list", async () => {
  let finishOld;
  let calls = 0;
  const page = workspace(() => calls++ === 0
    ? new Promise(resolve => { finishOld = resolve; })
    : Promise.resolve({ registrations: requests }));
  const old = page.run("loadRegistrations()");
  await page.run("loadRegistrations()");
  finishOld({ registrations: [] });
  await old;
  assert.equal(page.element("#registration-nav-count").textContent, 2);
});

test("HR startup renders requests returned by the running application", {
  skip: !process.env.CAREER_QUEST_TEST_URL,
}, async () => {
  const base = process.env.CAREER_QUEST_TEST_URL;
  const login = await fetch(`${base}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: "hr@careerquest.demo", password: "demo" }),
    signal: AbortSignal.timeout(10000),
  });
  assert.equal(login.status, 200);
  const cookie = login.headers.get("set-cookie").split(";")[0];
  try {
    let returnedRequests;
    const page = workspace(async (path) => {
      const response = await fetch(`${base}${path}`, {
        headers: { Cookie: cookie }, signal: AbortSignal.timeout(15000),
      });
      assert.equal(response.status, 200, path);
      const data = await response.json();
      if (path === "/registrations") returnedRequests = data.registrations;
      return data;
    });
    await page.run("init()");
    // The request panel loads independently; wait for it explicitly as well.
    await page.run("loadRegistrations()");
    assert.equal(page.element("#toast").textContent, "", "HR startup reported a rendering error");
    assert.equal(page.element("#registration-nav-count").textContent, returnedRequests.length);
    for (const request of returnedRequests) {
      assert.ok(page.element("#registration-table").innerHTML.includes(request.id));
    }
  } finally {
    await fetch(`${base}/auth/logout`, {
      method: "POST", headers: { Cookie: cookie }, signal: AbortSignal.timeout(10000),
    });
  }
});
