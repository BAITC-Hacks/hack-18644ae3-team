const employeeState = { user: null, profile: null, path: null, assessment: null, recommendations: [], mandatory: [], activities: [], catalog: null, questTab: "recommended", selectedEvent: "" };
const e$ = (selector, root = document) => root.querySelector(selector);
const e$$ = (selector, root = document) => [...root.querySelectorAll(selector)];

document.addEventListener("DOMContentLoaded", employeeInit);

async function employeeInit() {
  bindEmployeeUI();
  try {
    const session = await employeeAPI("/auth/me");
    if (session.user.role !== "employee") return location.replace(session.redirect);
    employeeState.user = session.user;
    employeeState.catalog = await employeeAPI("/catalog");
    populateEmployeeGoals();
    await loadMyCareer();
  } catch (error) {
    employeeToast(error.message, true);
    e$("#employee-loading p").textContent = "Could not load your workspace.";
  }
}

function bindEmployeeUI() {
  e$$('[data-employee-view]').forEach((button) => button.addEventListener("click", () => showEmployeeView(button.dataset.employeeView)));
  document.addEventListener("click", (event) => {
    const link = event.target.closest("[data-employee-view-link]");
    if (link) showEmployeeView(link.dataset.employeeViewLink);
    if (event.target.closest("[data-employee-goal]")) openEmployeeGoal();
    const why = event.target.closest("[data-employee-explain]");
    if (why) {
      employeeState.selectedEvent = why.dataset.employeeExplain;
      showEmployeeView("navigator");
      askEmployeeNavigator("Why is this course recommended for me?", employeeState.selectedEvent);
    }
  });
  e$("#employee-quest-tabs").addEventListener("click", (event) => {
    const tab = event.target.closest("[data-quest-tab]");
    if (!tab) return;
    employeeState.questTab = tab.dataset.questTab;
    e$$('[data-quest-tab]').forEach((button) => button.classList.toggle("active", button === tab));
    renderMyQuests();
  });
  e$("#employee-course-search").addEventListener("input", employeeDebounce(searchEmployeeCourses, 250));
  e$("#employee-chat-form").addEventListener("submit", (event) => {
    event.preventDefault();
    const input = e$("#employee-chat-input");
    if (!input.value.trim()) return;
    const question = input.value.trim(); input.value = "";
    askEmployeeNavigator(question, employeeState.selectedEvent);
  });
  e$("#employee-prompt-chips").addEventListener("click", (event) => {
    const chip = event.target.closest("[data-prompt]");
    if (chip) askEmployeeNavigator(chip.dataset.prompt, chip.dataset.event || "");
  });
  e$("#employee-goal-form").addEventListener("submit", saveEmployeeGoal);
  e$("#employee-cancel-goal").addEventListener("click", () => e$("#employee-goal-dialog").close());
  e$("#employee-clear-goal").addEventListener("click", clearEmployeeGoal);
  e$("#employee-logout").addEventListener("click", employeeLogout);
}

async function loadMyCareer() {
  e$("#employee-loading").classList.remove("hidden");
  const id = encodeURIComponent(employeeState.user.employee_id);
  try {
    const [profile, path, assessment, recommendations, mandatory, activities] = await Promise.all([
      employeeAPI(`/employees/${id}`), employeeAPI(`/employees/${id}/career-path`), employeeAPI(`/employees/${id}/skill-gaps`),
      employeeAPI(`/employees/${id}/recommendations`), employeeAPI(`/employees/${id}/mandatory-quests`), employeeAPI(`/employees/${id}/activities`),
    ]);
    Object.assign(employeeState, { profile, path, assessment, recommendations: recommendations.recommendations || [], mandatory: mandatory.quests || [], activities: activities.activities || [] });
    employeeState.selectedEvent = employeeState.recommendations[0]?.event_id || "";
    renderEmployeeWorkspace();
  } finally {
    e$("#employee-loading").classList.add("hidden");
  }
}

function renderEmployeeWorkspace() {
  const person = employeeState.profile.employee;
  e$("#employee-name").textContent = person.full_name;
  e$("#employee-position").textContent = `${person.role} · ${person.grade}`;
  e$("#employee-avatar").textContent = employeeInitials(person.full_name);
  e$("#employee-welcome").textContent = `Welcome back, ${person.full_name.split(" ")[0]}`;
  e$("#my-quest-count").textContent = employeeState.recommendations.length;
  renderPersonalHero(); renderEmployeeReadiness(); renderEmployeeNext(); renderEmployeeCriticalPreview(); renderEmployeeCurrent(); renderMyQuests(); renderEmployeeSkills(); renderEmployeePath(); resetEmployeeNavigator(); renderEmployeeProfile();
}

function renderPersonalHero() {
  const person = employeeState.profile.employee, goal = person.career_goal;
  e$("#personal-hero").innerHTML = goal ? `<span class="quest-label">My main quest</span><h2>${employeeEscape(goal.target_role)} · ${employeeEscape(goal.target_grade)}</h2><p>From ${employeeEscape(person.role)} · ${employeeEscape(person.grade)} to your selected destination.</p><div class="path-line"><div class="path-node"><span>Now</span><strong>${employeeEscape(person.grade)}</strong></div><span class="path-arrow">→</span><div class="path-node target"><span>Goal</span><strong>${employeeEscape(goal.target_grade)}</strong></div></div>` : `<span class="quest-label">Choose a destination</span><h2>Set your career goal</h2><p>Unlock readiness, focused gaps, and activities matched to where you want to go.</p><button class="button primary" data-employee-goal>Set my goal</button>`;
}

function renderEmployeeReadiness() {
  const value = employeeState.assessment.readiness_percent;
  const critical = employeeState.assessment.skill_gaps.filter((gap) => gap.critical).length;
  e$("#employee-readiness").innerHTML = `<div class="readiness-ring" style="--value:${value || 0}"><div class="readiness-number">${value == null ? "—" : Math.round(value) + "<small>%</small>"}</div></div><div class="readiness-copy"><h3>${value == null ? "Choose a goal" : "Career readiness"}</h3><p>${employeeEscape(employeeState.assessment.target_role)} · ${employeeEscape(employeeState.assessment.target_grade)}</p><div class="mini-stat"><span></span><strong>${critical}</strong> critical blockers</div><div class="mini-stat success"><span></span><strong>${employeeState.assessment.skill_gaps.length}</strong> gaps in total</div></div>`;
}

function renderEmployeeNext() {
  const quest = employeeState.recommendations[0];
  e$("#employee-next-quest").innerHTML = quest ? employeeQuestCard(quest, true) : employeeEmpty("No eligible quest right now", "Try updating your career goal or check back after new activities are published.");
}

function renderEmployeeCriticalPreview() {
  const gaps = employeeState.assessment.skill_gaps.filter((gap) => gap.critical).slice(0, 4);
  e$("#employee-critical-preview").innerHTML = gaps.length ? gaps.map((gap) => `<div class="focus-row"><div class="focus-name">${employeeEscape(gap.skill_name)}<span>Critical</span></div>${employeeLevelTrack(gap.current_level, gap.required_level)}<div class="level-label">${gap.current_level} / ${gap.required_level}</div></div>`).join("") : employeeEmpty("No critical blockers", "You meet all critical requirements for this target.");
}

function renderEmployeeCurrent() {
  const current = employeeState.activities.filter((activity) => ["in_progress", "planned", "enrolled"].includes(activity.status)).slice(0, 4);
  e$("#employee-current-preview").innerHTML = current.length ? current.map((activity) => `<div class="mandatory-item"><div class="mandatory-status">↗</div><div><strong>${employeeEscape(activity.title)}</strong><span>${employeeEscape(employeeTitle(activity.status))} · ${activity.completion_pct}%</span></div></div>`).join("") : employeeEmpty("Nothing in progress", "Choose a recommended activity when you are ready.");
}

function renderMyQuests() {
  let items;
  if (employeeState.questTab === "recommended") {
    e$("#employee-quest-list").innerHTML = employeeState.recommendations.length ? employeeState.recommendations.map((quest) => employeeQuestCard(quest)).join("") : employeeEmpty("No recommendations", "There are no eligible activities at the moment.");
    return;
  }
  items = employeeState.activities.filter((activity) => employeeState.questTab === "mandatory" ? activity.mandatory : !activity.mandatory && activity.category === employeeState.questTab);
  const latestByEvent = new Map();
  items.forEach((item) => { if (!latestByEvent.has(item.event_id)) latestByEvent.set(item.event_id, item); });
  const latest = [...latestByEvent.values()];
  e$("#employee-quest-list").innerHTML = latest.length ? latest.map(employeeActivityCard).join("") : employeeEmpty(`No ${employeeTitle(employeeState.questTab)} activities`, "Nothing is recorded in this section yet.");
}

async function searchEmployeeCourses() {
  const query = e$("#employee-course-search").value.trim();
  if (!query) { e$("#employee-course-results").innerHTML = ""; return; }
  try {
    const result = await employeeAPI(`/events?q=${encodeURIComponent(query)}`);
    const events = result.events || [];
    e$("#employee-course-results").innerHTML = events.length ? `<div class="quest-board">${events.map((event) => `<article class="quest-card"><div class="quest-top"><span class="quest-type">${employeeEscape(employeeTitle(event.type))}</span><span class="grade-badge">${event.mandatory ? "Mandatory" : "Optional"}</span></div><h3>${employeeEscape(event.title)}</h3><p>${employeeEscape(event.description)}</p><div class="quest-meta"><span>${event.duration_hours}h</span><span>${employeeEscape(employeeTitle(event.format))}</span></div>${event.learning_link ? `<a class="button primary" href="${employeeEscape(event.learning_link)}" target="_blank" rel="noopener noreferrer">Open training</a>` : ""}</article>`).join("")}</div>` : employeeEmpty("No courses found", "Try a different title, skill, role or grade.");
  } catch (error) { employeeToast(error.message, true); }
}

function employeeQuestCard(quest, featured = false) {
  const link = quest.learning_link ? `<a class="button primary" href="${employeeEscape(quest.learning_link)}" target="_blank" rel="noopener noreferrer">Open training ↗</a>` : `<span class="link-pending">Training link not added yet</span>`;
  return `<article class="quest-card ${featured ? "featured-quest" : ""}"><div class="quest-top"><span class="quest-type">${employeeEscape(employeeTitle(quest.type))}</span><span class="score-pill">${quest.match_score} match</span></div><h3>${employeeEscape(quest.title)}</h3><p>${employeeEscape(quest.explanation)}</p><div class="skill-tags">${quest.skills_covered.map((skill) => `<span class="skill-tag ${skill.critical ? "critical" : ""}">${employeeEscape(skill.skill_name)} ${skill.before_level}→${skill.projected_level}</span>`).join("")}</div><div class="quest-meta"><span>◷ ${quest.duration_hours}h</span>${quest.readiness_impact == null ? "" : `<span class="quest-impact">+${quest.readiness_impact}% ready</span>`}</div><div class="card-actions">${link}<button class="button secondary" data-employee-explain="${employeeEscape(quest.event_id)}">Why this quest?</button></div></article>`;
}

function employeeActivityCard(activity) {
  const link = activity.learning_link ? `<a class="button primary" href="${employeeEscape(activity.learning_link)}" target="_blank" rel="noopener noreferrer">Open training ↗</a>` : "";
  return `<article class="quest-card"><div class="quest-top"><span class="quest-type">${employeeEscape(employeeTitle(activity.event_type))}</span><span class="grade-badge">${employeeEscape(employeeTitle(activity.status))}</span></div><h3>${employeeEscape(activity.title)}</h3><p>${activity.completion_pct}% complete · ${employeeEscape(activity.assigned_by)} assigned</p><div class="progress-track"><i style="width:${activity.completion_pct}%"></i></div><div class="quest-meta"><span>${employeeEscape(activity.date)}</span><span>◷ ${activity.duration_hours}h</span></div>${link ? `<div class="card-actions">${link}</div>` : ""}</article>`;
}

function renderEmployeeSkills() {
  const gaps = employeeState.assessment.skill_gaps, critical = gaps.filter((gap) => gap.critical), growth = gaps.filter((gap) => !gap.critical);
  e$("#employee-skills-subtitle").textContent = `Compared with ${employeeState.assessment.target_role} · ${employeeState.assessment.target_grade}.`;
  e$("#employee-skill-summary").innerHTML = employeeSummary("◎", employeeState.assessment.readiness_percent == null ? "—" : `${employeeState.assessment.readiness_percent}%`, "Career readiness") + employeeSummary("!", critical.length, "Critical gaps") + employeeSummary("↗", gaps.reduce((sum, gap) => sum + gap.missing_levels, 0), "Levels to gain");
  e$("#employee-critical-skills").innerHTML = critical.length ? critical.map(employeeSkillRow).join("") : employeeEmpty("Critical requirements met", "No critical blockers remain.");
  e$("#employee-growth-skills").innerHTML = growth.length ? growth.map(employeeSkillRow).join("") : employeeEmpty("Growth requirements met", "No other gaps remain.");
}

function renderEmployeePath() {
  const person = employeeState.profile.employee, goal = person.career_goal, quest = employeeState.recommendations[0];
  if (!goal) { e$("#employee-career-path").innerHTML = employeeEmpty("Choose a career goal first", "Your visual path will appear here."); return; }
  const impacts = quest?.skills_covered || [];
  e$("#employee-career-path").innerHTML = `<div class="career-path-node current"><span>YOU ARE HERE</span><strong>${employeeEscape(person.role)}</strong><b>${employeeEscape(person.grade)}</b></div><div class="career-path-connector"><i></i><span>↓</span></div>${quest ? `<div class="career-path-node activity"><span>RECOMMENDED NEXT QUEST</span><strong>${employeeEscape(quest.title)}</strong><b>${quest.duration_hours} hours · ${quest.match_score} match</b>${quest.learning_link ? `<a href="${employeeEscape(quest.learning_link)}" target="_blank" rel="noopener noreferrer">Open training ↗</a>` : ""}</div><div class="career-path-connector"><i></i><span>↓</span></div><div class="path-impact-list">${impacts.map((skill) => `<div><span>${employeeEscape(skill.skill_name)}</span><strong>${skill.before_level} → ${skill.projected_level}</strong><small>Target ${skill.required_level}</small></div>`).join("")}</div><div class="career-path-connector"><i></i><span>↓</span></div>` : ""}<div class="career-path-node goal"><span>YOUR GOAL</span><strong>${employeeEscape(goal.target_role)}</strong><b>${employeeEscape(goal.target_grade)}</b><small>${employeeState.assessment.readiness_percent}% ready today</small></div>`;
}

function resetEmployeeNavigator() {
  const first = employeeState.profile.employee.full_name.split(" ")[0];
  e$("#employee-chat-messages").innerHTML = `<div class="message assistant">Hi ${employeeEscape(first)}. I can explain what blocks your goal, why each quest was selected, and which activity creates the biggest progress.</div>`;
  const prompts = [{ label: "What should I do next?", question: "What should I do next?", event: "" }, { label: "What blocks my promotion?", question: "What is blocking my promotion?", event: "" }, ...employeeState.recommendations.slice(0, 2).map((quest) => ({ label: `Why ${quest.title}?`, question: `Why is ${quest.title} recommended?`, event: quest.event_id }))];
  e$("#employee-prompt-chips").innerHTML = prompts.map((prompt) => `<button class="prompt-chip" data-prompt="${employeeEscape(prompt.question)}" data-event="${employeeEscape(prompt.event)}">${employeeEscape(prompt.label)}</button>`).join("");
  e$("#employee-navigator-context").innerHTML = `<h3>Your current context</h3><p>Private to your employee account.</p><div class="context-item"><span>Target</span><strong>${employeeEscape(employeeState.assessment.target_role)} · ${employeeEscape(employeeState.assessment.target_grade)}</strong></div><div class="context-item"><span>Readiness</span><strong>${employeeState.assessment.readiness_percent == null ? "Set a goal" : employeeState.assessment.readiness_percent + "%"}</strong></div><div class="context-item"><span>Top quest</span><strong>${employeeEscape(employeeState.recommendations[0]?.title || "None available")}</strong></div>`;
}

async function askEmployeeNavigator(question, eventID) {
  const messages = e$("#employee-chat-messages");
  messages.insertAdjacentHTML("beforeend", `<div class="message user">${employeeEscape(question)}</div><div class="message assistant loading">Checking your career data</div>`);
  messages.scrollTop = messages.scrollHeight;
  try {
    const result = await employeeAPI("/navigator/chat", { method: "POST", body: JSON.stringify({ employee_id: employeeState.user.employee_id, event_id: eventID || undefined, question }) });
    e$(".loading", messages)?.remove(); messages.insertAdjacentHTML("beforeend", `<div class="message assistant">${employeeEscape(result.message)}</div>`);
  } catch (error) { e$(".loading", messages)?.remove(); messages.insertAdjacentHTML("beforeend", `<div class="message assistant">${employeeEscape(error.message)}</div>`); }
  messages.scrollTop = messages.scrollHeight;
}

function renderEmployeeProfile() {
  const person = employeeState.profile.employee, goal = person.career_goal;
  e$("#employee-profile-card").innerHTML = `<div class="large-avatar">${employeeEscape(employeeInitials(person.full_name))}</div><h2>${employeeEscape(person.full_name)}</h2><p>${employeeEscape(person.role)} · ${employeeEscape(person.grade)}</p><dl><div><dt>Department</dt><dd>${employeeEscape(person.department)}</dd></div><div><dt>Work format</dt><dd>${employeeEscape(employeeTitle(person.work_format))}</dd></div><div><dt>Career goal</dt><dd>${goal ? `${employeeEscape(goal.target_role)} · ${employeeEscape(goal.target_grade)}` : "Not set"}</dd></div><div><dt>Last skill review</dt><dd>${employeeEscape(person.last_review_date)}</dd></div></dl>`;
  e$("#employee-profile-card dl").insertAdjacentHTML("beforeend", `<div><dt>Email</dt><dd>${employeeEscape(person.email || "")}</dd></div><div><dt>Phone</dt><dd>${employeeEscape(person.phone || "")}</dd></div><div><dt>Team</dt><dd>${employeeEscape(person.team || "")}</dd></div><div><dt>Manager</dt><dd>${employeeEscape(person.manager_name || "")}</dd></div><div><dt>Location</dt><dd>${employeeEscape(person.location || "")}</dd></div>`);
  const skillNames = Object.fromEntries(employeeState.catalog.skills.map((skill) => [skill.skill_id, skill.name]));
  e$("#employee-all-skills").innerHTML = Object.entries(employeeState.profile.effective_skills).sort((a, b) => (skillNames[a[0]] || a[0]).localeCompare(skillNames[b[0]] || b[0])).map(([id, level]) => `<div class="profile-skill-row"><span>${employeeEscape(skillNames[id] || id)}</span>${employeeLevelTrack(level, level)}<strong>${level}</strong></div>`).join("");
}

function openEmployeeGoal() { const goal = employeeState.profile.employee.career_goal; e$("#employee-goal-role").value = goal?.target_role || employeeState.profile.employee.role; e$("#employee-goal-grade").value = goal?.target_grade || nextEmployeeGrade(employeeState.profile.employee.grade); e$("#employee-clear-goal").hidden = !goal; e$("#employee-goal-dialog").showModal(); }
async function saveEmployeeGoal(event) { event.preventDefault(); await updateEmployeeGoal({ target_role: e$("#employee-goal-role").value, target_grade: e$("#employee-goal-grade").value }); }
async function clearEmployeeGoal() { await updateEmployeeGoal({ clear: true }); }
async function updateEmployeeGoal(body) { try { await employeeAPI(`/employees/${encodeURIComponent(employeeState.user.employee_id)}/career-goal`, { method: "PUT", body: JSON.stringify(body) }); e$("#employee-goal-dialog").close(); await loadMyCareer(); employeeToast("Career goal updated"); } catch (error) { employeeToast(error.message, true); } }
function populateEmployeeGoals() { e$("#employee-goal-role").innerHTML = employeeState.catalog.roles.map((role) => `<option>${employeeEscape(role)}</option>`).join(""); e$("#employee-goal-grade").innerHTML = employeeState.catalog.grades.map((grade) => `<option>${employeeEscape(grade)}</option>`).join(""); }
function showEmployeeView(view) { e$$('.view').forEach((section) => section.classList.toggle("active", section.id === `employee-view-${view}`)); e$$('[data-employee-view]').forEach((button) => button.classList.toggle("active", button.dataset.employeeView === view)); e$("#employee-page-context").textContent = ({ career: "My Career", quests: "My Quests", skills: "Skills", path: "Career Path", navigator: "AI Navigator", profile: "Profile" })[view]; scrollTo({ top: 0, behavior: "smooth" }); }
async function employeeLogout() { await employeeAPI("/auth/logout", { method: "POST", body: "{}" }); location.replace("/"); }
async function employeeAPI(path, options = {}) { const response = await fetch(path, { headers: { "Content-Type": "application/json", ...(options.headers || {}) }, ...options }); const data = await response.json().catch(() => ({})); if (response.status === 401) { location.replace("/"); throw new Error("Session expired"); } if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`); return data; }
function employeeSkillRow(gap) { return `<div class="skill-row"><div class="skill-row-head"><strong>${employeeEscape(gap.skill_name)}</strong><span>Level ${gap.current_level} → ${gap.required_level}</span></div><div class="skill-bar"><span style="width:${gap.current_level / 5 * 100}%"></span><i style="left:${gap.required_level / 5 * 100}%"></i></div><div class="skill-bar-labels"><span>0 · No knowledge</span><span>5 · Expert</span></div></div>`; }
function employeeLevelTrack(current, required) { return `<div class="level-track">${[1,2,3,4,5].map((level) => `<i class="${level <= current ? "filled" : level <= required ? "required" : ""}"></i>`).join("")}</div>`; }
function employeeSummary(icon, value, label) { return `<div class="skill-summary-card"><div class="summary-icon">${icon}</div><div><strong>${employeeEscape(value)}</strong><span>${employeeEscape(label)}</span></div></div>`; }
function employeeEmpty(title, text) { return `<div class="empty-state"><div><strong>${employeeEscape(title)}</strong>${employeeEscape(text)}</div></div>`; }
function employeeInitials(name) { return name.split(/\s+/).slice(0,2).map((part) => part[0]).join("").toUpperCase(); }
function employeeTitle(value) { return String(value || "").replaceAll("_", " ").replace(/\b\w/g, (char) => char.toUpperCase()); }
function nextEmployeeGrade(grade) { const grades = ["Junior","Middle","Senior","Lead"]; return grades[Math.min(grades.indexOf(grade) + 1, 3)] || "Middle"; }
function employeeEscape(value) { return String(value ?? "").replace(/[&<>'"]/g, (char) => ({ "&":"&amp;", "<":"&lt;", ">":"&gt;", "'":"&#39;", '"':"&quot;" })[char]); }
function employeeDebounce(fn, delay) { let timer; return (...args) => { clearTimeout(timer); timer = setTimeout(() => fn(...args), delay); }; }
let employeeToastTimer; function employeeToast(message, error = false) { const toast = e$("#employee-toast"); toast.textContent = message; toast.classList.toggle("error", error); toast.classList.add("show"); clearTimeout(employeeToastTimer); employeeToastTimer = setTimeout(() => toast.classList.remove("show"), 2800); }
