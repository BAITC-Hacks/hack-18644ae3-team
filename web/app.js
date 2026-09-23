const state = {
  user: null,
  employees: [],
  directoryEmployees: [],
  registrations: null,
  catalog: null,
  events: [],
  profile: null,
  careerPath: null,
  assessment: null,
  recommendations: [],
  mandatory: [],
  activities: [],
  selectedEventId: "",
  activeView: "overview",
};

const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];

document.addEventListener("DOMContentLoaded", init);

async function init() {
  bindNavigation();
  bindActions();
  try {
    const session = await api("/auth/me");
    if (session.user.role !== "hr") {
      location.replace(session.redirect);
      return;
    }
    state.user = session.user;
    loadRegistrations();
    const [health, employeeResult, catalog, eventResult] = await Promise.all([
      api("/health"),
      api("/employees"),
      api("/catalog"),
      api("/events"),
    ]);
    state.employees = employeeResult.employees || [];
    state.directoryEmployees = state.employees;
    state.catalog = catalog;
    state.events = (eventResult.events || []).map(normalizeHREvent);
    $("#snapshot-date").textContent = formatDate(health.as_of_date);
    populateEmployeeSelect();
    populateGoalRoles();
    populateHRFilters();
    if (state.registrations !== null) renderRegistrations();
    showView(location.hash.slice(1) || "overview", false);
    const remembered = localStorage.getItem("careerQuestEmployee");
    const firstID = state.employees.some((employee) => employee.employee_id === remembered)
      ? remembered
      : (state.employees.find((employee) => employee.employee_id === "E0001")?.employee_id || state.employees[0]?.employee_id);
    if (firstID) {
      $("#employee-select").value = firstID;
      await loadEmployee(firstID);
    }
  } catch (error) {
    showToast(error.message, true);
    $("#loading-screen p").textContent = "Could not load Career Quest.";
  } finally {
    setLoading(false);
  }
}

function bindNavigation() {
  $$(".nav-item[data-view]").forEach((button) => {
    button.addEventListener("click", () => showView(button.dataset.view));
  });
  document.addEventListener("click", (event) => {
    const viewButton = event.target.closest("[data-go-view]");
    if (viewButton) showView(viewButton.dataset.goView);
  });
  window.addEventListener("hashchange", () => {
    const view = location.hash.slice(1);
    if ($(`#view-${view}`)) showView(view, false);
  });
}

function bindActions() {
  $("#employee-select").addEventListener("change", (event) => loadEmployee(event.target.value));
  $("#refresh-button").addEventListener("click", () => {
    if (state.activeView === "registrations") return loadRegistrations();
    if (state.profile) loadEmployee(state.profile.employee.employee_id, true);
  });
  $("#refresh-registrations").addEventListener("click", loadRegistrations);
  window.addEventListener("focus", () => {
    if (state.user && state.activeView === "registrations") loadRegistrations();
  });
  $("#quest-format-filter").addEventListener("change", renderQuestBoard);
  const searchEmployees = debounce(loadEmployeeDirectory, 250);
  $("#employee-search").addEventListener("input", searchEmployees);
  $("#employee-department-filter").addEventListener("input", searchEmployees);
  $("#employee-team-filter").addEventListener("input", searchEmployees);
  $("#employee-role-filter").addEventListener("change", loadEmployeeDirectory);
  $("#employee-grade-filter").addEventListener("change", loadEmployeeDirectory);
  const searchEvents = debounce(loadHREvents, 250);
  $("#hr-event-search").addEventListener("input", searchEvents);
  $("#hr-event-type").addEventListener("change", loadHREvents);
  $("#open-employee-create").addEventListener("click", () => $("#employee-dialog").showModal());
  $("#close-employee-dialog").addEventListener("click", () => $("#employee-dialog").close());
  $("#cancel-employee-create").addEventListener("click", () => $("#employee-dialog").close());
  $("#employee-form").addEventListener("submit", createEmployee);
  $("#registration-table").addEventListener("click", handleRegistrationDecision);
  $("#employee-table").addEventListener("click", (event) => {
    const button = event.target.closest("[data-open-employee]");
    if (!button) return;
    $("#employee-select").value = button.dataset.openEmployee;
    loadEmployee(button.dataset.openEmployee).then(() => showView("overview"));
  });
  $("#hr-event-grid").addEventListener("click", (event) => {
    const button = event.target.closest("[data-event-candidates]");
    if (button) showCandidates(button.dataset.eventCandidates);
  });
  $("#close-candidates").addEventListener("click", () => $("#candidate-dialog").close());
  $("#candidate-list").addEventListener("click", (event) => {
    const row = event.target.closest("[data-candidate-employee]");
    if (!row) return;
    $("#candidate-dialog").close();
    $("#employee-select").value = row.dataset.candidateEmployee;
    loadEmployee(row.dataset.candidateEmployee).then(() => showView("overview"));
  });
  $$('[data-logout]').forEach((button) => button.addEventListener("click", logout));
  document.addEventListener("click", (event) => {
    if (event.target.closest("[data-open-goal]")) openGoalDialog();
    const explain = event.target.closest("[data-explain-event]");
    if (explain) {
      state.selectedEventId = explain.dataset.explainEvent;
      showView("navigator");
      askNavigator("Why is this quest recommended for me?", state.selectedEventId);
    }
  });
  $("#goal-form").addEventListener("submit", saveGoal);
  $("#cancel-goal").addEventListener("click", () => $("#goal-dialog").close());
  $("#clear-goal").addEventListener("click", clearGoal);
  $("#chat-form").addEventListener("submit", (event) => {
    event.preventDefault();
    const input = $("#chat-input");
    const question = input.value.trim();
    if (!question) return;
    input.value = "";
    askNavigator(question, state.selectedEventId);
  });
  $("#prompt-chips").addEventListener("click", (event) => {
    const chip = event.target.closest("[data-prompt]");
    if (!chip) return;
    state.selectedEventId = chip.dataset.event || "";
    askNavigator(chip.dataset.prompt, state.selectedEventId);
  });
}

async function loadEmployee(employeeID, refreshed = false) {
  setLoading(true);
  try {
    const encoded = encodeURIComponent(employeeID);
    const [profile, careerPath, assessment, recommendationResult, mandatoryResult, activityResult] = await Promise.all([
      api(`/employees/${encoded}`),
      api(`/employees/${encoded}/career-path`),
      api(`/employees/${encoded}/skill-gaps`),
      api(`/employees/${encoded}/recommendations`),
      api(`/employees/${encoded}/mandatory-quests`),
      api(`/employees/${encoded}/activities`),
    ]);
    state.profile = profile;
    state.careerPath = careerPath;
    state.assessment = assessment;
    state.recommendations = recommendationResult.recommendations || [];
    state.mandatory = mandatoryResult.quests || [];
    state.activities = activityResult.activities || [];
    state.selectedEventId = state.recommendations[0]?.event_id || "";
    localStorage.setItem("careerQuestEmployee", employeeID);
    renderAll();
    if (refreshed) showToast("Employee data refreshed");
  } catch (error) {
    showToast(error.message, true);
  } finally {
    setLoading(false);
  }
}

function renderAll() {
  renderIdentity();
  renderMainQuest();
  renderReadiness();
  renderQuestPreviews();
  renderSkillFocus();
  renderMandatory();
  renderQuestBoard();
  renderSkills();
  resetNavigator();
  renderEmployeeDirectory();
  renderHREvents();
  renderHRAnalytics();
  renderHRAccount();
  renderHRActivityStatus();
  $("#quest-nav-count").textContent = state.recommendations.length;
  $("#employee-nav-count").textContent = state.employees.length;
}

function renderIdentity() {
  const employee = state.profile.employee;
  const firstName = employee.full_name.split(" ")[0];
  $("#top-name").textContent = state.user.name;
  $("#top-role").textContent = "HR · Organization access";
  $("#top-avatar").textContent = initials(state.user.name);
  $("#overview-title").textContent = `${firstName}’s development profile`;
  $("#overview-subtitle").textContent = employee.career_goal
    ? `You’re building momentum toward ${employee.career_goal.target_role} · ${employee.career_goal.target_grade}.`
    : "Set a career goal to turn development activities into a focused path.";
}

function renderMainQuest() {
  const employee = state.profile.employee;
  const goal = employee.career_goal;
  if (!goal) {
    $("#main-quest-card").innerHTML = `
      <span class="quest-label">Main quest not selected</span>
      <h2>Where do you want to go next?</h2>
      <p>Choose a role and grade to unlock career readiness and targeted recommendations.</p>
      <div class="main-quest-actions"><button class="button" data-open-goal>Choose career goal →</button></div>`;
    return;
  }
  $("#main-quest-card").innerHTML = `
    <span class="quest-label">Main quest</span>
    <h2>${escapeHTML(goal.target_role)} · ${escapeHTML(goal.target_grade)}</h2>
    <p>Your recommended path from today’s role to your selected destination.</p>
    <div class="path-line">
      <div class="path-node"><span>Now</span><strong>${escapeHTML(employee.role)} · ${escapeHTML(employee.grade)}</strong></div>
      <span class="path-arrow">→</span>
      <div class="path-node target"><span>Target</span><strong>${escapeHTML(goal.target_role)} · ${escapeHTML(goal.target_grade)}</strong></div>
    </div>
    <div class="main-quest-actions"><button class="button" data-go-view="skills">View blockers →</button></div>`;
}

function renderReadiness() {
  const readiness = state.assessment.readiness_percent;
  const criticalCount = state.assessment.skill_gaps.filter((gap) => gap.critical).length;
  const regularCount = state.assessment.skill_gaps.length - criticalCount;
  const displayValue = readiness == null ? 0 : readiness;
  $("#readiness-card").innerHTML = `
    <div class="readiness-ring" style="--value:${clamp(displayValue, 0, 100)}">
      <div class="readiness-number">${readiness == null ? "—" : Math.round(readiness)}${readiness == null ? "" : "<small>%</small>"}</div>
    </div>
    <div class="readiness-copy">
      <h3>${readiness == null ? "Goal needed" : "Career readiness"}</h3>
      <p>${readiness == null ? "Set a target to calculate readiness." : `Measured against ${escapeHTML(state.assessment.target_role)} · ${escapeHTML(state.assessment.target_grade)}.`}</p>
      <div class="mini-stat"><span></span><strong>${criticalCount}</strong> critical blocker${criticalCount === 1 ? "" : "s"}</div>
      <div class="mini-stat success"><span></span><strong>${regularCount}</strong> growth skill${regularCount === 1 ? "" : "s"}</div>
    </div>`;
}

function renderQuestPreviews() {
  const root = $("#quest-preview-grid");
  const quests = state.recommendations.slice(0, 3);
  root.innerHTML = quests.length
    ? quests.map((quest) => questCard(quest, true)).join("")
    : emptyState("No eligible quests", "Try selecting a different career goal or check back when new sessions are announced.");
}

function questCard(quest, compact = false) {
  const tags = quest.skills_covered.slice(0, compact ? 2 : 4).map((skill) =>
    `<span class="skill-tag ${skill.critical ? "critical" : ""}">${escapeHTML(skill.skill_name)} ${skill.before_level}→${skill.projected_level}</span>`
  ).join("");
  const session = quest.format === "self_paced"
    ? "Start anytime"
    : quest.upcoming_sessions?.length ? formatDate(quest.upcoming_sessions[0]) : "No session";
  const impact = quest.readiness_impact == null ? "" : `<span class="quest-impact">+${number(quest.readiness_impact)}% ready</span>`;
  return `
    <article class="quest-card" data-format="${escapeHTML(quest.format)}">
      <div class="quest-top"><span class="quest-type">${escapeHTML(titleCase(quest.type))}</span><span class="score-pill">${number(quest.match_score)} match</span></div>
      <h3>${escapeHTML(quest.title)}</h3>
      <p>${quest.alignment === "career_goal" ? "Directly aligned with your career goal." : "Builds a gap in your current role."}</p>
      <div class="skill-tags">${tags}</div>
      <div class="quest-meta"><span>◷ ${formatHours(quest.duration_hours)}</span><span>◉ ${escapeHTML(session)}</span>${impact}</div>
      <button class="explain-button" data-explain-event="${escapeHTML(quest.event_id)}" title="Ask Navigator why">✦</button>
    </article>`;
}

function renderSkillFocus() {
  const gaps = state.assessment.skill_gaps.slice(0, 4);
  $("#skill-focus").innerHTML = gaps.length ? gaps.map((gap) => `
    <div class="focus-row">
      <div class="focus-name">${escapeHTML(gap.skill_name)}${gap.critical ? "<span>Critical requirement</span>" : ""}</div>
      ${levelTrack(gap.current_level, gap.required_level)}
      <div class="level-label">Level ${gap.current_level} / ${gap.required_level}</div>
    </div>`).join("") : emptyState("Target met", "There are no remaining skill gaps for this target.");
}

function renderMandatory() {
  const latestByEvent = new Map();
  state.activities.forEach((activity) => {
    if (!latestByEvent.has(activity.event_id)) latestByEvent.set(activity.event_id, activity);
  });
  const sorted = [...latestByEvent.values()];
  $("#mandatory-list").innerHTML = sorted.length ? sorted.slice(0, 4).map((quest) => {
    const overdue = quest.status === "overdue";
    const meta = overdue && quest.due_date ? `Due ${formatDate(quest.due_date)}` : `${titleCase(quest.status)} · ${quest.completion_pct}%`;
    return `<div class="mandatory-item">
      <div class="mandatory-status ${overdue ? "overdue" : ""}">${overdue ? "!" : quest.status === "completed" ? "✓" : "↗"}</div>
      <div><strong>${escapeHTML(quest.title)}</strong><span class="${overdue ? "overdue-text" : ""}">${escapeHTML(meta)}</span></div>
    </div>`;
  }).join("") : emptyState("No activity yet", "There is no development history for this employee.");
}

function populateHRFilters() {
  $("#employee-role-filter").innerHTML += state.catalog.roles.map((role) => `<option>${escapeHTML(role)}</option>`).join("");
  $("#employee-grade-filter").innerHTML += state.catalog.grades.map((grade) => `<option>${escapeHTML(grade)}</option>`).join("");
  const types = [...new Set(state.events.map((event) => event.type))].sort();
  $("#hr-event-type").innerHTML += types.map((type) => `<option value="${escapeHTML(type)}">${escapeHTML(titleCase(type))}</option>`).join("");
  $("#new-employee-role").innerHTML = state.catalog.roles.map((role) => `<option>${escapeHTML(role)}</option>`).join("");
  $("#new-employee-grade").innerHTML = state.catalog.grades.map((grade) => `<option>${escapeHTML(grade)}</option>`).join("");
}

function renderEmployeeDirectory() {
  const employees = state.directoryEmployees;
  $("#employee-table").innerHTML = employees.length ? employees.map((employee) => `
    <tr>
      <td><div class="table-person"><span>${escapeHTML(initials(employee.full_name))}</span><div><strong>${escapeHTML(employee.full_name)}</strong><small>${escapeHTML(employee.employee_id)}</small></div></div></td>
      <td>${escapeHTML(employee.role)}</td><td><span class="grade-badge">${escapeHTML(employee.grade)}</span></td><td>${escapeHTML(employee.department)}</td>
      <td>${employee.career_goal ? `${escapeHTML(employee.career_goal.target_role)} · ${escapeHTML(employee.career_goal.target_grade)}` : '<span class="muted">Not set</span>'}</td>
      <td><button class="table-action" data-open-employee="${escapeHTML(employee.employee_id)}">Open →</button></td>
    </tr>`).join("") : `<tr><td colspan="6">${emptyState("No employees found", "Try changing the directory filters.")}</td></tr>`;
}

async function loadEmployeeDirectory() {
  const query = new URLSearchParams({ q: $("#employee-search").value.trim(), department: $("#employee-department-filter").value.trim(), team: $("#employee-team-filter").value.trim(), role: $("#employee-role-filter").value, grade: $("#employee-grade-filter").value });
  try { const result = await api(`/employees?${query}`); state.directoryEmployees = result.employees || []; renderEmployeeDirectory(); }
  catch (error) { showToast(error.message, true); }
}

function renderHREvents() {
  const events = state.searchedEvents || state.events;
  $("#hr-event-grid").innerHTML = events.length ? events.map((event) => `
    <article class="panel catalog-card">
      <div class="quest-top"><span class="quest-type">${escapeHTML(titleCase(event.type))}</span><span class="${event.mandatory ? "mandatory-badge" : "optional-badge"}">${event.mandatory ? "Mandatory" : "Optional"}</span></div>
      <h3>${escapeHTML(event.title)}</h3><p>${escapeHTML(event.description)}</p>
      <div class="skill-tags">${(event.develops_skills || []).slice(0, 3).map((skill) => `<span class="skill-tag">${escapeHTML(skill.skill_id.replace("SK_", "").replaceAll("_", " "))} +${skill.gain}</span>`).join("")}</div>
      <div class="quest-meta"><span>◷ ${formatHours(event.duration_hours)}</span><span>${escapeHTML(titleCase(event.format))}</span></div>
      <button class="button secondary full-button" data-event-candidates="${escapeHTML(event.event_id)}">View best candidates</button>
    </article>`).join("") : emptyState("No events found", "Try changing the catalog filters.");
}

async function loadHREvents() {
  const query = new URLSearchParams({ q: $("#hr-event-search").value.trim(), type: $("#hr-event-type").value });
  try { const result = await api(`/events?${query}`); state.searchedEvents = (result.events || []).map(normalizeHREvent); renderHREvents(); }
  catch (error) { showToast(error.message, true); }
}

let registrationsRequest = 0;
async function loadRegistrations() {
  const request = ++registrationsRequest;
  const refresh = $("#refresh-registrations");
  refresh.disabled = true;
  $("#registration-table").innerHTML = '<tr><td colspan="5" role="status">Loading registration requests...</td></tr>';
  try {
    const result = await api("/registrations", { cache: "no-store" });
    if (request !== registrationsRequest) return;
    state.registrations = result.registrations || [];
    renderRegistrations();
  } catch (error) {
    if (request !== registrationsRequest) return;
    state.registrations = null;
    $("#registration-nav-count").textContent = "!";
    $("#registration-table").innerHTML = `<tr><td colspan="5" role="alert">Could not load registration requests: ${escapeHTML(error.message)}. Use Refresh to try again.</td></tr>`;
  } finally {
    if (request === registrationsRequest) refresh.disabled = false;
  }
}

function renderRegistrations() {
  if (state.registrations === null) return;
  $("#registration-nav-count").textContent = state.registrations.length;
  $("#registration-table").innerHTML = state.registrations.length ? state.registrations.map((registration) => `<tr><td><strong>${escapeHTML(registration.name)}</strong></td><td>${escapeHTML(registration.email)}</td><td><small>Requested ID: ${escapeHTML(registration.requested_employee_id || "Not provided")}</small><select aria-label="Employee to link on approval" data-registration-employee="${escapeHTML(registration.id)}"><option value="">Choose employee</option>${state.employees.map((employee) => `<option value="${escapeHTML(employee.employee_id)}" ${employee.employee_id === registration.requested_employee_id ? "selected" : ""}>${escapeHTML(employee.full_name)} (${escapeHTML(employee.employee_id)})</option>`).join("")}</select></td><td><span class="grade-badge">${escapeHTML(registration.status)}</span></td><td><button class="table-action" data-approve-registration="${escapeHTML(registration.id)}">Approve</button> <button class="table-action danger-text" data-reject-registration="${escapeHTML(registration.id)}">Reject</button></td></tr>`).join("") : `<tr><td colspan="5">${emptyState("No pending registrations", "New account requests will appear here.")}</td></tr>`;
}

async function handleRegistrationDecision(event) {
  const approve = event.target.closest("[data-approve-registration]");
  const reject = event.target.closest("[data-reject-registration]");
  if (!approve && !reject) return;
  const id = approve?.dataset.approveRegistration || reject.dataset.rejectRegistration;
  try {
    const options = { method: "POST", body: reject ? "{}" : JSON.stringify({ employee_id: $(`[data-registration-employee="${CSS.escape(id)}"]`).value }) };
    await api(`/registrations/${encodeURIComponent(id)}/${approve ? "approve" : "reject"}`, options);
    await loadRegistrations(); showToast(approve ? "Account approved" : "Registration rejected");
  } catch (error) { showToast(error.message, true); }
}

async function createEmployee(event) {
  event.preventDefault();
  const payload = { employee_id: $("#new-employee-id").value.trim(), full_name: $("#new-employee-name").value.trim(), email: $("#new-employee-email").value.trim(), phone: $("#new-employee-phone").value.trim(), department: $("#new-employee-department").value.trim(), team: $("#new-employee-team").value.trim(), location: $("#new-employee-location").value.trim(), role: $("#new-employee-role").value, grade: $("#new-employee-grade").value };
  try { await api("/employees", { method: "POST", body: JSON.stringify(payload) }); const result = await api("/employees"); state.employees = result.employees || []; state.directoryEmployees = state.employees; populateEmployeeSelect(); renderEmployeeDirectory(); renderRegistrations(); event.currentTarget.reset(); $("#employee-dialog").close(); showToast("Employee created"); }
  catch (error) { showToast(error.message, true); }
}

function renderHRAnalytics() {
  const withGoals = state.employees.filter((employee) => employee.career_goal).length;
  const mandatory = state.events.filter((event) => event.mandatory).length;
  const upcoming = state.events.reduce((sum, event) => sum + event.upcoming_sessions.length, 0);
  $("#hr-analytics").innerHTML = [
    [state.employees.length, "Employees"],
    [`${Math.round(withGoals / state.employees.length * 100)}%`, "Career goals set"],
    [state.events.length - mandatory, "Voluntary activities"],
    [upcoming, "Upcoming sessions"],
  ].map(([value, label]) => `<div class="panel metric-card"><strong>${value}</strong><span>${label}</span></div>`).join("");
  const counts = Object.entries(state.employees.reduce((acc, employee) => ({ ...acc, [employee.role]: (acc[employee.role] || 0) + 1 }), {})).sort((a, b) => b[1] - a[1]);
  const maxCount = Math.max(...counts.map(([, count]) => count), 1);
  $("#role-bars").innerHTML = counts.map(([role, count]) => `<div class="chart-row"><span>${escapeHTML(role)}</span><div><i style="width:${count / maxCount * 100}%"></i></div><strong>${count}</strong></div>`).join("");
}

function renderHRAccount() {
  $("#hr-account").innerHTML = `<div class="account-avatar">${escapeHTML(initials(state.user.name))}</div><div><span class="eyebrow">HR ACCOUNT</span><h2>${escapeHTML(state.user.name)}</h2><p>${escapeHTML(state.user.email)}</p><span class="access-badge">Organization development access</span></div>`;
}

function renderHRActivityStatus() {
  const sections = [
    ["completed", "Completed", state.activities.filter((activity) => !activity.mandatory && activity.status === "completed")],
    ["in_progress", "Planned / In progress", state.activities.filter((activity) => !activity.mandatory && ["planned", "enrolled", "in_progress"].includes(activity.status))],
    ["mandatory", "Mandatory", state.activities.filter((activity) => activity.mandatory)],
  ];
  $("#hr-activity-status").innerHTML = sections.map(([kind, label, activities]) => {
    const latest = activities.filter((activity, index, all) => all.findIndex((item) => item.event_id === activity.event_id) === index).slice(0, 3);
    return `<article class="panel activity-status-card"><div class="activity-status-head"><span class="title-icon ${kind === "mandatory" ? "danger" : "purple"}">${kind === "completed" ? "✓" : kind === "mandatory" ? "!" : "↗"}</span><div><strong>${escapeHTML(label)}</strong><small>${activities.length} records</small></div></div>${latest.length ? latest.map((activity) => `<div class="activity-mini-row"><span>${escapeHTML(activity.title)}</span><b>${escapeHTML(titleCase(activity.status))}</b></div>`).join("") : `<p class="muted">No activities in this status.</p>`}</article>`;
  }).join("");
}

async function showCandidates(eventID) {
  const event = state.events.find((item) => item.event_id === eventID);
  $("#candidate-title").textContent = `${event?.title || "Event"} candidates`;
  $("#candidate-list").innerHTML = `<div class="empty-state">Loading eligible candidates…</div>`;
  $("#candidate-dialog").showModal();
  try {
    const result = await api(`/events/${encodeURIComponent(eventID)}/candidates`);
    $("#candidate-list").innerHTML = result.candidates.length ? result.candidates.slice(0, 20).map((candidate, index) => `
      <button class="candidate-row" data-candidate-employee="${escapeHTML(candidate.employee_id)}">
        <span class="candidate-rank">${index + 1}</span><span class="candidate-avatar">${escapeHTML(initials(candidate.full_name))}</span><span><strong>${escapeHTML(candidate.full_name)}</strong><small>${escapeHTML(candidate.role)} · ${escapeHTML(candidate.grade)}</small></span><b>${number(candidate.match_score)}</b>
      </button>`).join("") : emptyState("No eligible candidates", "No employee currently meets the event filters and target gaps.");
  } catch (error) {
    $("#candidate-list").innerHTML = emptyState("Could not load candidates", error.message);
  }
}

async function logout() {
  await api("/auth/logout", { method: "POST", body: "{}" });
  location.replace("/");
}

function renderQuestBoard() {
  const filter = $("#quest-format-filter").value;
  const quests = filter === "all" ? state.recommendations : state.recommendations.filter((quest) => quest.format === filter);
  const goalAligned = state.recommendations.filter((quest) => quest.alignment === "career_goal").length;
  const selfPaced = state.recommendations.filter((quest) => quest.format === "self_paced").length;
  $("#quest-board-summary").innerHTML = `
    <span class="summary-pill"><strong>${state.recommendations.length}</strong> eligible</span>
    <span class="summary-pill"><strong>${goalAligned}</strong> goal-aligned</span>
    <span class="summary-pill"><strong>${selfPaced}</strong> self-paced</span>`;
  $("#quest-board").innerHTML = quests.length
    ? quests.map((quest) => questCard(quest)).join("")
    : emptyState("No quests in this format", "Choose another format to see available activities.");
}

function renderSkills() {
  const gaps = state.assessment.skill_gaps;
  const critical = gaps.filter((gap) => gap.critical);
  const growth = gaps.filter((gap) => !gap.critical);
  const totalMissing = gaps.reduce((sum, gap) => sum + gap.missing_levels, 0);
  const readiness = state.assessment.readiness_percent;
  $("#skills-subtitle").textContent = `Measured against ${state.assessment.target_role} · ${state.assessment.target_grade}.`;
  $("#skill-summary").innerHTML = `
    ${summaryCard("◎", readiness == null ? "—" : `${number(readiness)}%`, "Career readiness")}
    ${summaryCard("!", critical.length, "Critical blockers")}
    ${summaryCard("↗", totalMissing, "Skill levels to gain")}`;
  $("#critical-skills").innerHTML = critical.length ? critical.map(skillRow).join("") : emptyState("No critical blockers", "All critical requirements are currently met.");
  $("#growth-skills").innerHTML = growth.length ? growth.map(skillRow).join("") : emptyState("No growth gaps", "All other target requirements are met.");
}

function skillRow(gap) {
  return `<div class="skill-row">
    <div class="skill-row-head"><strong>${escapeHTML(gap.skill_name)}</strong><span>Level ${gap.current_level} → ${gap.required_level}</span></div>
    <div class="skill-bar"><span style="width:${clamp(gap.current_level / 5 * 100, 0, 100)}%"></span><i style="left:${clamp(gap.required_level / 5 * 100, 0, 100)}%"></i></div>
    <div class="skill-bar-labels"><span>0 · No knowledge</span><span>5 · Expert</span></div>
  </div>`;
}

function resetNavigator() {
  const employee = state.profile.employee;
  $("#chat-messages").innerHTML = `<div class="message assistant">Hi ${escapeHTML(employee.full_name.split(" ")[0])}. I can explain your recommendations using your target, assessed skills, prerequisites, and activity history. What would you like to explore?</div>`;
  const top = state.recommendations.slice(0, 3);
  const chips = [
    { prompt: "What is blocking my career goal?", label: "My blockers", event: "" },
    ...top.map((quest) => ({ prompt: `Why should I take ${quest.title}?`, label: quest.title, event: quest.event_id })),
  ];
  $("#prompt-chips").innerHTML = chips.map((chip) => `<button class="prompt-chip" data-prompt="${escapeHTML(chip.prompt)}" data-event="${escapeHTML(chip.event)}">${escapeHTML(chip.label)}</button>`).join("");
  const best = state.recommendations[0];
  $("#navigator-context").innerHTML = `
    <h3>Current context</h3><p>The Navigator only explains backend-calculated results.</p>
    <div class="context-item"><span>Employee</span><strong>${escapeHTML(employee.full_name)}</strong></div>
    <div class="context-item"><span>Target</span><strong>${escapeHTML(state.assessment.target_role)} · ${escapeHTML(state.assessment.target_grade)}</strong></div>
    <div class="context-item"><span>Readiness</span><strong>${state.assessment.readiness_percent == null ? "Set a goal" : `${number(state.assessment.readiness_percent)}%`}</strong></div>
    <div class="context-item"><span>Top recommendation</span><strong>${best ? escapeHTML(best.title) : "No eligible quest"}</strong></div>`;
}

async function askNavigator(question, eventID = "") {
  const messages = $("#chat-messages");
  messages.insertAdjacentHTML("beforeend", `<div class="message user">${escapeHTML(question)}</div><div class="message assistant loading">Checking your career data</div>`);
  messages.scrollTop = messages.scrollHeight;
  try {
    const response = await api("/navigator/chat", {
      method: "POST",
      body: JSON.stringify({ employee_id: state.profile.employee.employee_id, event_id: eventID || undefined, question }),
    });
    $(".message.loading", messages)?.remove();
    messages.insertAdjacentHTML("beforeend", `<div class="message assistant">${escapeHTML(response.message)}</div>`);
  } catch (error) {
    $(".message.loading", messages)?.remove();
    messages.insertAdjacentHTML("beforeend", `<div class="message assistant">${escapeHTML(error.message)}</div>`);
  }
  messages.scrollTop = messages.scrollHeight;
}

function openGoalDialog() {
  const goal = state.profile.employee.career_goal;
  $("#goal-role").value = goal?.target_role || state.profile.employee.role;
  $("#goal-grade").value = goal?.target_grade || nextGrade(state.profile.employee.grade);
  $("#clear-goal").hidden = !goal;
  $("#goal-dialog").showModal();
}

async function saveGoal(event) {
  event.preventDefault();
  const employeeID = state.profile.employee.employee_id;
  try {
    await api(`/employees/${encodeURIComponent(employeeID)}/career-goal`, {
      method: "PUT",
      body: JSON.stringify({ target_role: $("#goal-role").value, target_grade: $("#goal-grade").value }),
    });
    $("#goal-dialog").close();
    await loadEmployee(employeeID);
    showToast("Career goal updated");
  } catch (error) {
    showToast(error.message, true);
  }
}

async function clearGoal() {
  const employeeID = state.profile.employee.employee_id;
  try {
    await api(`/employees/${encodeURIComponent(employeeID)}/career-goal`, { method: "PUT", body: JSON.stringify({ clear: true }) });
    $("#goal-dialog").close();
    await loadEmployee(employeeID);
    showToast("Career goal cleared");
  } catch (error) {
    showToast(error.message, true);
  }
}

function populateEmployeeSelect() {
  $("#employee-select").innerHTML = state.employees.map((employee) =>
    `<option value="${escapeHTML(employee.employee_id)}">${escapeHTML(employee.full_name)} — ${escapeHTML(employee.role)}, ${escapeHTML(employee.grade)}</option>`
  ).join("");
}

function populateGoalRoles() {
  const roles = [...new Set(state.employees.map((employee) => employee.role))].sort();
  $("#goal-role").innerHTML = roles.map((role) => `<option>${escapeHTML(role)}</option>`).join("");
}

function showView(view, updateHash = true) {
  state.activeView = view;
  $$(".view").forEach((section) => section.classList.toggle("active", section.id === `view-${view}`));
  $$(".nav-item").forEach((button) => button.classList.toggle("active", button.dataset.view === view));
  if (updateHash) history.replaceState(null, "", `#${view}`);
  if (view === "registrations" && state.user) loadRegistrations();
  window.scrollTo({ top: 0, behavior: "smooth" });
}

function setLoading(loading) {
  $("#loading-screen").classList.toggle("hidden", !loading);
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
  return data;
}

let toastTimer;
function showToast(message, error = false) {
  const toast = $("#toast");
  toast.textContent = message;
  toast.classList.toggle("error", error);
  toast.classList.add("show");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.remove("show"), 2800);
}

function levelTrack(current, required) {
  return `<div class="level-track" aria-label="Current level ${current}, required level ${required}">${[1, 2, 3, 4, 5].map((level) => `<i class="${level <= current ? "filled" : level <= required ? "required" : ""}"></i>`).join("")}</div>`;
}

function summaryCard(icon, value, label) {
  return `<div class="skill-summary-card"><div class="summary-icon">${icon}</div><div><strong>${escapeHTML(value)}</strong><span>${escapeHTML(label)}</span></div></div>`;
}

function emptyState(title, description) {
  return `<div class="empty-state"><div><strong>${escapeHTML(title)}</strong>${escapeHTML(description)}</div></div>`;
}

function initials(name) {
  return name.split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
}

function nextGrade(grade) {
  const grades = ["Junior", "Middle", "Senior", "Lead"];
  return grades[Math.min(grades.indexOf(grade) + 1, grades.length - 1)] || "Middle";
}

function titleCase(value) {
  return String(value || "").replaceAll("_", " ").replace(/\b\w/g, (char) => char.toUpperCase());
}

function formatDate(value) {
  if (!value) return "—";
  const date = new Date(`${value}T00:00:00Z`);
  return new Intl.DateTimeFormat("en", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(date);
}

function formatHours(value) {
  return `${number(value)}h`;
}

function number(value) {
  return Number(value || 0).toLocaleString(undefined, { maximumFractionDigits: 1 });
}

function clamp(value, min, max) {
  return Math.min(Math.max(Number(value) || 0, min), max);
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>'"]/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "'": "&#39;",
    '"': "&quot;",
  })[character]);
}

function debounce(fn, delay) { let timer; return (...args) => { clearTimeout(timer); timer = setTimeout(() => fn(...args), delay); }; }
function normalizeHREvent(event) { return { ...event, target_roles: event.target_roles || [], target_grades: event.target_grades || [], develops_skills: event.develops_skills || [], upcoming_sessions: event.upcoming_sessions || [] }; }
