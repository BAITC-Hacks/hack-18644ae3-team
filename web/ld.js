const ldState = { user: null, catalog: null, events: [] };
const l$ = (selector, root = document) => root.querySelector(selector);
const l$$ = (selector, root = document) => [...root.querySelectorAll(selector)];

document.addEventListener("DOMContentLoaded", ldInit);

async function ldInit() {
  bindLDUI();
  try {
    const session = await ldAPI("/auth/me");
    if (session.user.role !== "ld") return location.replace(session.redirect);
    ldState.user = session.user;
    const [catalog, eventResult] = await Promise.all([ldAPI("/catalog"), ldAPI("/events")]);
    ldState.catalog = catalog; ldState.events = eventResult.events || [];
    setupEventForm(); renderLDWorkspace();
  } catch (error) {
    ldToast(error.message, true); l$("#ld-loading p").textContent = "Could not load the L&D workspace.";
  } finally { l$("#ld-loading").classList.add("hidden"); }
}

function bindLDUI() {
  l$$('[data-ld-view]').forEach((button) => button.addEventListener("click", () => showLDView(button.dataset.ldView)));
  l$$('[data-ld-create]').forEach((button) => button.addEventListener("click", () => openEventForm()));
  l$("#event-dialog-close").addEventListener("click", closeEventForm); l$("#event-cancel").addEventListener("click", closeEventForm);
  l$("#event-form").addEventListener("submit", saveEvent);
  l$("#add-effect").addEventListener("click", () => addEffectRow());
  l$("#add-prerequisite").addEventListener("click", () => addPrerequisiteRow());
  l$("#event-effects").addEventListener("click", removeMetadataRow); l$("#event-prerequisites").addEventListener("click", removeMetadataRow);
  l$("#ld-event-search").addEventListener("input", renderLDEvents); l$("#ld-event-filter").addEventListener("change", renderLDEvents);
  l$("#ld-event-table").addEventListener("click", (event) => { const edit = event.target.closest("[data-edit-event]"); if (edit) openEventForm(edit.dataset.editEvent); });
  l$("#ld-analytics-event").addEventListener("change", renderEventAnalytics);
  l$("#ld-logout").addEventListener("click", async () => { await ldAPI("/auth/logout", { method: "POST", body: "{}" }); location.replace("/"); });
}

function setupEventForm() {
  l$("#event-roles").innerHTML = ldState.catalog.roles.map((role) => `<label><input type="checkbox" value="${ldEscape(role)}"> ${ldEscape(role)}</label>`).join("");
  l$("#event-grades").innerHTML = ldState.catalog.grades.map((grade) => `<label><input type="checkbox" value="${ldEscape(grade)}"> ${ldEscape(grade)}</label>`).join("");
  const types = ["course","workshop","mentoring","certification","meetup","compliance","onboarding"];
  l$("#ld-event-filter").innerHTML += types.map((type) => `<option value="${type}">${ldEscape(ldTitle(type))}</option>`).join("");
}

function renderLDWorkspace() {
  l$("#ld-name").textContent = ldState.user.name; l$("#ld-avatar").textContent = ldInitials(ldState.user.name); l$("#ld-event-count").textContent = ldState.events.length;
  renderLDMetrics(); renderLDEvents(); renderLDSessions(); renderLDAnalyticsSelector();
  l$("#ld-account").innerHTML = `<div class="account-avatar ld-avatar">${ldEscape(ldInitials(ldState.user.name))}</div><div><span class="eyebrow">L&amp;D ACCOUNT</span><h2>${ldEscape(ldState.user.name)}</h2><p>${ldEscape(ldState.user.email)}</p><span class="access-badge">Course and event management access</span><p class="scope-note">This role cannot open private employee profiles or the HR directory.</p></div>`;
}

function renderLDMetrics() {
  const optional = ldState.events.filter((event) => !event.mandatory).length, withLinks = ldState.events.filter((event) => event.learning_link).length;
  const sessions = ldState.events.reduce((sum, event) => sum + event.upcoming_sessions.length, 0);
  l$("#ld-metrics").innerHTML = [[ldState.events.length,"Catalog activities"],[optional,"Optional development"],[sessions,"Upcoming sessions"],[withLinks,"External links configured"]].map(([value,label]) => `<div class="panel metric-card"><strong>${value}</strong><span>${label}</span></div>`).join("");
  const upcoming = allSessions().slice(0, 5);
  l$("#ld-session-preview").innerHTML = upcoming.length ? upcoming.map(sessionRow).join("") : ldEmpty("No sessions", "Add dates to scheduled activities.");
  const formatCounts = ldState.events.reduce((acc, event) => ({ ...acc, [event.format]: (acc[event.format] || 0) + 1 }), {}), max = Math.max(...Object.values(formatCounts), 1);
  l$("#ld-format-bars").innerHTML = Object.entries(formatCounts).map(([format,count]) => `<div class="chart-row"><span>${ldEscape(ldTitle(format))}</span><div><i style="width:${count/max*100}%"></i></div><strong>${count}</strong></div>`).join("");
}

function renderLDEvents() {
  const query = l$("#ld-event-search").value.trim().toLowerCase(), type = l$("#ld-event-filter").value;
  const skillNames = Object.fromEntries(ldState.catalog.skills.map((skill) => [skill.skill_id, skill.name]));
  const events = ldState.events.filter((event) => (!query || `${event.title} ${event.description} ${event.develops_skills.map((effect) => skillNames[effect.skill_id]).join(" ")}`.toLowerCase().includes(query)) && (!type || event.type === type));
  l$("#ld-event-table").innerHTML = events.length ? events.map((event) => `<tr><td><div class="event-table-title"><strong>${ldEscape(event.title)}</strong><small>${event.mandatory ? "Mandatory" : "Optional"} · ${event.develops_skills.length} skills${event.learning_link ? " · Link ready" : ""}</small></div></td><td>${ldEscape(ldTitle(event.type))}<br><span class="muted">${ldEscape(ldTitle(event.format))}</span></td><td>${event.target_roles.length} roles<br><span class="muted">${event.target_grades.map(ldEscape).join(", ")}</span></td><td>${event.duration_hours}h</td><td>${event.upcoming_sessions.length || (event.format === "self_paced" ? "Anytime" : "—")}</td><td><button class="table-action" data-edit-event="${ldEscape(event.event_id)}">Edit →</button></td></tr>`).join("") : `<tr><td colspan="6">${ldEmpty("No activities found", "Change your search or create a new activity.")}</td></tr>`;
}

function renderLDSessions() { const sessions = allSessions(); l$("#ld-session-list").innerHTML = sessions.length ? sessions.map((session) => `<div class="panel session-card"><div class="session-date"><strong>${new Date(session.date + "T00:00:00Z").getUTCDate()}</strong><span>${new Date(session.date + "T00:00:00Z").toLocaleString("en", { month: "short", timeZone: "UTC" })}</span></div><div><span class="eyebrow">${ldEscape(ldTitle(session.event.type))}</span><h3>${ldEscape(session.event.title)}</h3><p>${ldEscape(session.event.target_roles.join(", "))} · ${session.event.duration_hours} hours</p></div><span class="grade-badge">${ldEscape(ldTitle(session.event.format))}</span></div>`).join("") : ldEmpty("No upcoming sessions", "Self-paced activities remain available anytime."); }
function allSessions() { return ldState.events.flatMap((event) => event.upcoming_sessions.map((date) => ({ date, event }))).sort((a,b) => a.date.localeCompare(b.date)); }
function sessionRow(session) { return `<div class="mandatory-item"><div class="session-mini-date">${session.date.slice(5)}</div><div><strong>${ldEscape(session.event.title)}</strong><span>${ldEscape(ldTitle(session.event.format))} · ${session.event.duration_hours}h</span></div></div>`; }

function renderLDAnalyticsSelector() { l$("#ld-analytics-event").innerHTML = ldState.events.map((event) => `<option value="${ldEscape(event.event_id)}">${ldEscape(event.title)}</option>`).join(""); if (ldState.events.length) renderEventAnalytics(); }
async function renderEventAnalytics() { const id = l$("#ld-analytics-event").value; if (!id) return; l$("#ld-analytics-content").innerHTML = `<div class="empty-state">Loading analytics…</div>`; try { const data = await ldAPI(`/events/${encodeURIComponent(id)}/analytics`); const statuses = Object.entries(data.status_counts || {}); l$("#ld-analytics-content").innerHTML = `<div class="analytics-grid"><div class="panel metric-card"><strong>${data.unique_participants}</strong><span>Participants</span></div><div class="panel metric-card"><strong>${data.total_records}</strong><span>Activity records</span></div><div class="panel metric-card"><strong>${Math.round(data.average_completion_pct)}%</strong><span>Average completion</span></div><div class="panel metric-card"><strong>${data.status_counts.completed || 0}</strong><span>Completions</span></div></div><div class="panel chart-panel"><div class="panel-title"><div><h2>Status distribution</h2><p>${ldEscape(data.title)}</p></div></div>${statuses.map(([status,count]) => `<div class="chart-row"><span>${ldEscape(ldTitle(status))}</span><div><i style="width:${count / Math.max(...statuses.map(([,value]) => value),1)*100}%"></i></div><strong>${count}</strong></div>`).join("")}</div>`; } catch (error) { l$("#ld-analytics-content").innerHTML = ldEmpty("Could not load analytics", error.message); } }

function openEventForm(eventID = "") {
  l$("#event-form").reset(); l$("#event-id").value = eventID; l$("#event-effects").innerHTML = ""; l$("#event-prerequisites").innerHTML = "";
  const event = ldState.events.find((item) => item.event_id === eventID);
  l$("#event-form-title").textContent = event ? "Edit activity" : "Create activity";
  if (event) {
    l$("#event-title").value = event.title; l$("#event-description").value = event.description; l$("#event-type").value = event.type; l$("#event-format").value = event.format; l$("#event-duration").value = event.duration_hours; l$("#event-mandatory").checked = event.mandatory; l$("#event-link").value = event.learning_link || ""; l$("#event-sessions").value = event.upcoming_sessions.join("\n");
    l$$("#event-roles input").forEach((input) => input.checked = event.target_roles.includes(input.value)); l$$("#event-grades input").forEach((input) => input.checked = event.target_grades.includes(input.value));
    event.develops_skills.forEach((effect) => addEffectRow(effect)); Object.entries(event.prerequisites).forEach(([skill, level]) => addPrerequisiteRow(skill, level));
  } else {
    l$("#event-duration").value = 8; addEffectRow(); l$$("#event-grades input").forEach((input) => input.checked = ["Junior","Middle"].includes(input.value));
  }
  l$("#event-dialog").showModal();
}
function closeEventForm() { l$("#event-dialog").close(); }
function skillOptions(selected = "") { return ldState.catalog.skills.map((skill) => `<option value="${ldEscape(skill.skill_id)}" ${skill.skill_id === selected ? "selected" : ""}>${ldEscape(skill.name)}</option>`).join(""); }
function addEffectRow(effect = {}) { l$("#event-effects").insertAdjacentHTML("beforeend", `<div class="metadata-row"><select data-field="skill">${skillOptions(effect.skill_id)}</select><label>Gain<input data-field="gain" type="number" min="1" max="5" value="${effect.gain || 1}"></label><label>Max level<input data-field="max" type="number" min="1" max="5" value="${effect.max_level || 4}"></label><button type="button" data-remove-row>×</button></div>`); }
function addPrerequisiteRow(skill = "", level = 1) { l$("#event-prerequisites").insertAdjacentHTML("beforeend", `<div class="metadata-row prerequisite-row"><select data-field="skill">${skillOptions(skill)}</select><label>Minimum level<input data-field="level" type="number" min="0" max="5" value="${level}"></label><button type="button" data-remove-row>×</button></div>`); }
function removeMetadataRow(event) { const button = event.target.closest("[data-remove-row]"); if (button) button.closest(".metadata-row").remove(); }

async function saveEvent(event) {
  event.preventDefault();
  const roles = l$$("#event-roles input:checked").map((input) => input.value), grades = l$$("#event-grades input:checked").map((input) => input.value);
  if (!roles.length || !grades.length) return ldToast("Select at least one target role and grade", true);
  const effects = l$$("#event-effects .metadata-row").map((row) => ({ skill_id: l$('[data-field="skill"]', row).value, gain: Number(l$('[data-field="gain"]', row).value), max_level: Number(l$('[data-field="max"]', row).value) }));
  const prerequisites = {}; l$$("#event-prerequisites .metadata-row").forEach((row) => prerequisites[l$('[data-field="skill"]', row).value] = Number(l$('[data-field="level"]', row).value));
  const format = l$("#event-format").value;
  const payload = { title: l$("#event-title").value.trim(), description: l$("#event-description").value.trim(), type: l$("#event-type").value, format, duration_hours: Number(l$("#event-duration").value), mandatory: l$("#event-mandatory").checked, target_roles: roles, target_grades: grades, develops_skills: effects, prerequisites, upcoming_sessions: format === "self_paced" ? [] : l$("#event-sessions").value.split(/\s+/).filter(Boolean), learning_link: l$("#event-link").value.trim() };
  const id = l$("#event-id").value;
  try { await ldAPI(id ? `/events/${encodeURIComponent(id)}` : "/events", { method: id ? "PUT" : "POST", body: JSON.stringify(payload) }); const result = await ldAPI("/events"); ldState.events = result.events || []; closeEventForm(); renderLDWorkspace(); showLDView("events"); ldToast(id ? "Activity updated" : "Activity created"); } catch (error) { ldToast(error.message, true); }
}

function showLDView(view) { l$$('.view').forEach((section) => section.classList.toggle("active", section.id === `ld-view-${view}`)); l$$('[data-ld-view]').forEach((button) => button.classList.toggle("active", button.dataset.ldView === view)); l$("#ld-page-context").textContent = ({ dashboard:"Dashboard",events:"Courses & Events",sessions:"Upcoming Sessions",analytics:"Course Analytics",profile:"Profile" })[view]; scrollTo({ top:0, behavior:"smooth" }); }
async function ldAPI(path, options = {}) { const response = await fetch(path, { headers: { "Content-Type":"application/json", ...(options.headers || {}) }, ...options }); const data = await response.json().catch(() => ({})); if (response.status === 401) { location.replace("/"); throw new Error("Session expired"); } if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`); return data; }
function ldTitle(value) { return String(value || "").replaceAll("_", " ").replace(/\b\w/g, (char) => char.toUpperCase()); }
function ldInitials(name) { return name.split(/\s+/).slice(0,2).map((part) => part[0]).join("").toUpperCase(); }
function ldEscape(value) { return String(value ?? "").replace(/[&<>'"]/g, (char) => ({ "&":"&amp;", "<":"&lt;", ">":"&gt;", "'":"&#39;", '"':"&quot;" })[char]); }
function ldEmpty(title, text) { return `<div class="empty-state"><div><strong>${ldEscape(title)}</strong>${ldEscape(text)}</div></div>`; }
let ldToastTimer; function ldToast(message, error = false) { const toast = l$("#ld-toast"); toast.textContent = message; toast.classList.toggle("error", error); toast.classList.add("show"); clearTimeout(ldToastTimer); ldToastTimer = setTimeout(() => toast.classList.remove("show"), 2800); }
