const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const { join } = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

function workspace() {
  const elements = new Map();
  function element(selector) {
    if (!elements.has(selector)) elements.set(selector, {
      innerHTML: "", textContent: "", classList: { add() {}, remove() {} },
    });
    return elements.get(selector);
  }
  const context = vm.createContext({ document: { querySelector: element, addEventListener() {} } });
  vm.runInContext(readFileSync(join(__dirname, "..", "employee.js"), "utf8"), context);
  return { run: (code) => vm.runInContext(code, context), element };
}

const plan = {
  total_duration_hours: 10, readiness_before_percent: 50, readiness_after_percent: 90,
  remaining_gap_count: 1, explanation: "Complete each step before the next.",
  steps: [
    { event_id: "FOUNDATION", title: "SQL Foundations <basics>", format: "self_paced", duration_hours: 2, skills_developed: [], explanation: "Unlocks Advanced Architecture.", readiness_after_percent: 50 },
    { event_id: "ADVANCED", title: "Advanced Architecture", format: "online", duration_hours: 8, skills_developed: [], explanation: "Closes critical gaps.", readiness_after_percent: 90 },
  ],
};

function loadPlan(page) {
  page.run(`Object.assign(employeeState, {
    profile: { employee: { role: "Backend Engineer", grade: "Junior", career_goal: {} } },
    assessment: { target_role: "Backend Engineer", target_grade: "Middle" },
    learningPlan: ${JSON.stringify(plan)}, recommendations: []
  })`);
}

test("a prerequisite course appears as the next step even without direct recommendations", () => {
  const page = workspace();
  loadPlan(page);
  page.run("renderEmployeeNext(); renderMyQuests(); renderEmployeePath()");
  assert.match(page.element("#employee-next-quest").innerHTML, /SQL Foundations &lt;basics&gt;/);
  assert.match(page.element("#employee-learning-plan").innerHTML, /10 study hours/);
  const path = page.element("#employee-career-path").innerHTML;
  assert.ok(path.indexOf("SQL Foundations &lt;basics&gt;") < path.indexOf("<h3>Advanced Architecture</h3>"));
  assert.match(path, /50% → 90%/);
  assert.match(path, /1 skill gaps projected to remain/);
});

test("switching to activity history hides the proposed learning plan", () => {
  const page = workspace();
  loadPlan(page);
  page.run('employeeState.questTab = "completed"; renderMyQuests()');
  assert.equal(page.element("#employee-learning-plan").innerHTML, "");
});

test("reloading after a goal change clears an obsolete plan and selected course", async () => {
  const page = workspace();
  loadPlan(page);
  page.run(`employeeState.user = { employee_id: "TEST" };
    employeeState.selectedEvent = "OLD";
    employeeAPI = async () => ({});
    renderEmployeeWorkspace = () => {};`);
  await page.run("loadMyCareer()");
  assert.equal(page.run("employeeState.learningPlan"), null);
  assert.equal(page.run("employeeState.selectedEvent"), "");
});
